/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/sdk-go/core/event"
	"github.com/CESSProject/sdk-go/core/pattern"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/mr-tron/base58/base58"
)

func (n *Node) subscribeMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	var err error
	var startBlock uint32
	var peerid string
	var stakingAcc string
	var storageNode pattern.MinerInfo
	var events = event.EventRecords{}

	for {
		startBlock, err = n.parsingOldBlocks(startBlock)
		if err == nil {
			break
		}
		time.Sleep(pattern.BlockInterval)
	}
	for {
		if n.GetChainState() {
			sub, err := n.GetSubstrateAPI().RPC.Chain.SubscribeNewHeads()
			if err != nil {
				time.Sleep(pattern.BlockInterval)
				continue
			}
			defer sub.Unsubscribe()
			for {
				head := <-sub.Chan()
				fmt.Printf("Chain is at block: #%v\n", head.Number)
				blockhash, err := n.GetSubstrateAPI().RPC.Chain.GetBlockHash(uint64(head.Number))
				if err != nil {
					continue
				}

				h, err := n.GetSubstrateAPI().RPC.State.GetStorageRaw(n.GetKeyEvents(), blockhash)
				if err != nil {
					continue
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(n.GetMetadata(), &events)
				if err != nil {
					continue
				}

				// Corresponding processing according to different events
				for _, v := range events.Sminer_Registered {
					storageNode, err = n.QueryStorageMiner(v.Acc[:])
					if err != nil {
						continue
					}
					stakingAcc, _ = utils.EncodeToCESSAddr(v.Acc[:])
					peerid = base58.Encode([]byte(string(storageNode.PeerId[:])))
					n.SaveStoragePeer(peerid, stakingAcc)
					configs.Tip(fmt.Sprintf("Record a storage node: %s", peerid))
				}

				// for _, v := range events.TeeWorker_RegistrationTeeWorker {
				// 	peerid = base58.Encode([]byte(string(v.Ip[:])))
				// 	n.SaveTeePeer(peerid, 0)
				// 	configs.Tip(fmt.Sprintf("Record a tee node: %s", peerid))
				// }
			}
		}
	}
}

func (n *Node) parsingOldBlocks(block uint32) (uint32, error) {
	var err error
	var peerid string
	var stakingAcc string
	var blockheight uint32
	var startBlock uint32 = block
	var storageNode pattern.MinerInfo
	var events = event.EventRecords{}
	for {
		blockheight, err = n.QueryBlockHeight("")
		if err != nil {
			return startBlock, err
		}
		if startBlock >= blockheight {
			return startBlock, nil
		}
		for i := startBlock; i <= blockheight; i++ {
			blockhash, err := n.GetSubstrateAPI().RPC.Chain.GetBlockHash(uint64(i))
			if err != nil {
				return startBlock, err
			}

			h, err := n.GetSubstrateAPI().RPC.State.GetStorageRaw(n.GetKeyEvents(), blockhash)
			if err != nil {
				return startBlock, err
			}

			err = types.EventRecordsRaw(*h).DecodeEventRecords(n.GetMetadata(), &events)
			if err != nil {
				return startBlock, err
			}

			for _, v := range events.Sminer_Registered {
				storageNode, err = n.QueryStorageMiner(v.Acc[:])
				if err != nil {
					return startBlock, err
				}
				stakingAcc, _ = utils.EncodeToCESSAddr(v.Acc[:])
				peerid = base58.Encode([]byte(string(storageNode.PeerId[:])))
				n.SaveStoragePeer(peerid, stakingAcc)
				configs.Tip(fmt.Sprintf("Record a storage node: %s", peerid))
			}

			for _, v := range events.TeeWorker_RegistrationScheduler {
				peerid = base58.Encode([]byte(string(v.Ip[:])))
				n.SaveTeePeer(peerid, 0)
				configs.Tip(fmt.Sprintf("Record a tee node: %s", peerid))
			}
			startBlock = i
		}
		startBlock = blockheight
	}
}
