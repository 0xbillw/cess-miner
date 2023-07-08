/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/bytedance/sonic"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
)

// spaceMgt is a subtask for managing spaces
func (n *Node) spaceMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var err error
	var spacePath string
	var tagPath string
	var txhash string
	var filehash string
	var peerid string
	var blockheight uint32
	var teepuk []byte
	var idlefile pattern.IdleFileMeta

	n.Space("info", ">>>>> start spaceMgt <<<<<")

	timeout := time.NewTimer(time.Duration(time.Minute * 5))
	defer timeout.Stop()

	for {
		for n.GetChainState() {
			time.Sleep(pattern.BlockInterval)
			teepuk, peerid, err = n.requsetIdlefile()
			if err != nil || peerid == "" {
				n.Space("err", fmt.Sprintf("[%v] %v", peerid, err.Error()))
				continue
			}

			n.Space("info", fmt.Sprintf("Requset a idle file to: %s", peerid))

			spacePath = ""
			tagPath = ""

			timeout.Reset(time.Duration(time.Minute * 5))
			for err == nil {
				select {
				case <-timeout.C:
					n.Space("err", fmt.Sprintf("Response timeout: %s", peerid))
					err = errors.New("timeout")
				case spacePath = <-n.GetIdleDataCh():
				case tagPath = <-n.GetIdleTagCh():
				}

				if tagPath != "" && spacePath != "" {
					break
				}
			}

			if tagPath == "" || spacePath == "" {
				os.Remove(tagPath)
				os.Remove(spacePath)
				continue
			}

			n.Space("info", fmt.Sprintf("Receive a idle file tag: %s", tagPath))
			n.Space("info", fmt.Sprintf("Receive a idle file: %s", spacePath))

			err = verifyTagfile(tagPath, spacePath)
			if err != nil {
				os.Remove(tagPath)
				os.Remove(spacePath)
				n.Space("err", err.Error())
				continue
			}

			filehash, err = sutils.CalcPathSHA256(spacePath)
			if err != nil {
				n.Space("err", err.Error())
				os.Remove(spacePath)
				os.Remove(tagPath)
				continue
			}
			if filehash != filepath.Base(spacePath) {
				os.Remove(spacePath)
				os.Remove(tagPath)
				continue
			}

			idlefile.BlockNum = pattern.BlockNumber
			idlefile.Hash = filehash
			idlefile.MinerAcc = n.GetStakingPublickey()
			txhash, err = n.SubmitIdleFile(teepuk, []pattern.IdleFileMeta{idlefile})
			if err != nil {
				n.Space("err", fmt.Sprintf("Submit idlefile metadata err: %v", err.Error()))
				if txhash != "" {
					err = n.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filehash)), []byte(fmt.Sprintf("%s", txhash)))
					if err != nil {
						n.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filehash, err))
						continue
					}
				}
				n.Space("err", fmt.Sprintf("Submit idlefile [%s] err [%s] %v", filehash, txhash, err))
				continue
			}

			n.Space("info", fmt.Sprintf("Submit idle file %s suc: %s", filehash, txhash))

			blockheight, err = n.QueryBlockHeight(txhash)
			if err != nil {
				err = n.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filehash)), []byte(fmt.Sprintf("%s", txhash)))
				if err != nil {
					n.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filehash, err))
				}
				continue
			}

			err = n.Put([]byte(fmt.Sprintf("%s%s", Cach_prefix_idle, filepath.Base(spacePath))), []byte(fmt.Sprintf("%d", blockheight)))
			if err != nil {
				n.Space("err", fmt.Sprintf("Record idlefile [%s] failed [%v]", filehash, err))
				continue
			}

			n.Space("info", fmt.Sprintf("Record idle file %s suc: %d", filehash, blockheight))
		}
		time.Sleep(pattern.BlockInterval)
	}
}

func (n *Node) requsetIdlefile() ([]byte, string, error) {
	var err error
	var teePeerId string
	var freeSpace uint64

	freeSpace, err = utils.GetDirFreeSpace(n.GetWorkspace())
	if err != nil {
		return nil, "", errors.Wrapf(err, "[GetDirFreeSpace]")
	}

	if freeSpace < pattern.SIZE_1MiB*100 {
		return nil, "", errors.New("disk space will be used up soon")
	}

	usedSpace, err := utils.DirSize(n.Workspace())
	if err != nil {
		return nil, "", errors.Wrapf(err, "[DirSize]")
	}

	if usedSpace >= uint64(n.GetUseSpace()*pattern.SIZE_1GiB) {
		return nil, "", errors.New("the configured usage space limit is reached")
	}

	sign, err := n.Sign(n.GetPeerPublickey())
	if err != nil {
		return nil, teePeerId, err
	}

	tees := n.GetAllTeeWorkAccount()

	utils.RandSlice(tees)

	for _, tee := range tees {
		puk, err := sutils.ParsingPublickey(tee)
		if err != nil {
			continue
		}
		teepeerid, ok := n.GetTeeWork(tee)
		if !ok {
			continue
		}

		teePeerId = base58.Encode(teepeerid)
		addr, ok := n.GetPeer(teePeerId)
		if !ok {
			addr, err = n.DHTFindPeer(teePeerId)
			if err != nil {
				continue
			}
		}

		err = n.Connect(n.GetCtxQueryFromCtxCancel(), addr)
		if err != nil {
			continue
		}
		_, err = n.IdleReq(addr.ID, pattern.FragmentSize, pattern.BlockNumber, n.GetStakingPublickey(), sign)
		if err != nil {
			continue
		}
		return puk, teePeerId, nil
	}

	return nil, teePeerId, err
}

func verifyTagfile(tagfile, idlefile string) error {
	buf, err := os.ReadFile(tagfile)
	if err != nil {
		return err
	}
	var tagInfo tagInfo

	err = sonic.Unmarshal(buf, &tagInfo)
	if err != nil {
		return err
	}
	tagFileHash := strings.TrimSuffix(filepath.Base(tagfile), ".tag")
	if tagInfo.T.Name != tagFileHash {
		return fmt.Errorf("[%s] not equal tag name: [%s]", tagFileHash, tagInfo.T.Name)
	}
	if tagFileHash != filepath.Base(idlefile) {
		return fmt.Errorf("%s not equal idle file hash: [%s]", tagFileHash, filepath.Base(idlefile))
	}
	return nil
}
