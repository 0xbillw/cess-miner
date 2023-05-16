/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/pkg/proof"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/p2p-go/pb"
	"github.com/CESSProject/sdk-go/core/chain"
	"github.com/CESSProject/sdk-go/core/client"
	"github.com/CESSProject/sdk-go/core/rule"
)

// challengeMgr
func (n *Node) challengeMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Log.Pnc(utils.RecoverError(err))
		}
	}()

	var err error
	var txhash string
	var key *proof.RSAKeyPair
	var challenge client.ChallengeInfo
	var idleProofFileHash string
	var serviceProofFileHash string
	var idleSiama string
	var serviceSigma string

	for {
		pubkey, err := n.Cli.QueryTeePodr2Puk()
		if err != nil || len(pubkey) == 0 {
			time.Sleep(rule.BlockInterval)
			continue
		}
		n.Log.Chal("info", fmt.Sprintf("TEEKey: %v", pubkey))
		key = proof.GetKey(pubkey)
		break
	}

	for {
		challenge, err = n.Cli.QueryChallenge(n.Cfg.GetPublickey())
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				n.Log.Chal("err", err.Error())
				continue
			}
		}
		if challenge.Start == 0 {
			continue
		}

		n.Log.Chal("info", fmt.Sprintf("Challenge start: %v", challenge.Start))
		n.Log.Chal("info", fmt.Sprintf("Challenge random: %v", challenge.Random))

		idleSiama, idleProofFileHash, err = n.idleAggrProof(key, challenge.Random, challenge.Start)
		if err != nil {
			n.Log.Chal("err", err.Error())
			continue
		}
		_ = idleProofFileHash
		serviceSigma, serviceProofFileHash, err = n.serviceAggrProof(key, challenge.Random, challenge.Start)
		if err != nil {
			n.Log.Chal("err", err.Error())
			continue
		}
		_ = serviceProofFileHash

		n.Cach.Put([]byte(Cach_prefix_idleSiama), []byte(idleSiama))
		n.Cach.Put([]byte(Cach_prefix_serviceSiama), []byte(serviceSigma))

		//todo: report proof
		// txhash, err = n.Cli.Chain.ReportProof(idleSiama, serviceSigma)
		// if err != nil {

		// }
		_ = txhash
	}
}

func (n *Node) idleAggrProof(key *proof.RSAKeyPair, random []byte, start uint32) (string, string, error) {
	idleRoothashs, err := n.Cach.QueryPrefixKeyListByHeigh(Cach_prefix_idle, start)
	if err != nil {
		return "", "", err
	}
	var buf []byte
	var tag pb.Tag
	var ptags []proof.Tag = make([]proof.Tag, 0)
	var ptag proof.Tag
	var actualCount int
	var pf ProofFileType
	var pf_mu ProofMuFileType
	var proveResponse proof.GenProofResponse

	pf.Name = make([]string, len(idleRoothashs))
	pf.U = make([]string, len(idleRoothashs))
	pf_mu.Mu = make([]string, len(idleRoothashs))
	var qslice = []proof.QElement{
		{
			I: 0,
			V: string(random),
		},
	}

	for i := int(0); i < len(idleRoothashs); i++ {
		buf, err = os.ReadFile(filepath.Join(n.Cli.IdleTagDir, idleRoothashs[i]))
		if err != nil {
			continue
		}
		err = json.Unmarshal(buf, &tag)
		if err != nil {
			continue
		}

		matrix, _, err := proof.SplitByN(filepath.Join(n.Cli.IdleDataDir, idleRoothashs[i]), int64(len(tag.T.Phi)))
		if err != nil {
			continue
		}

		ptag.T.Name = tag.T.Name
		ptag.T.Phi = tag.T.Phi
		ptag.T.U = tag.T.U
		ptag.PhiHash = tag.PhiHash
		ptag.Attest = tag.Attest

		proveResponseCh := key.GenProof(qslice, nil, ptag, matrix)
		timeout := time.NewTimer(time.Duration(time.Minute))
		defer timeout.Stop()
		select {
		case proveResponse = <-proveResponseCh:
		case <-timeout.C:
			proveResponse.StatueMsg.StatusCode = 0
		}

		if proveResponse.StatueMsg.StatusCode != proof.Success {
			continue
		}
		ptags = append(ptags, ptag)
		pf.Name[actualCount] = tag.T.Name
		pf.Name[actualCount] = tag.T.U
		pf_mu.Mu[actualCount] = proveResponse.MU
		actualCount++
	}

	pf.Name = pf.Name[:actualCount]
	pf.U = pf.U[:actualCount]
	pf_mu.Mu = pf_mu.Mu[:actualCount]

	//
	buf, err = json.Marshal(&pf)
	if err != nil {
		return "", "", err
	}
	f, err := os.OpenFile(n.Cli.IproofFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return "", "", err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	_, err = f.Write(buf)
	if err != nil {
		return "", "", err
	}
	err = f.Sync()
	if err != nil {
		return "", "", err
	}
	f.Close()
	f = nil
	//
	buf, err = json.Marshal(&pf_mu)
	if err != nil {
		return "", "", err
	}
	f, err = os.OpenFile(n.Cli.IproofMuFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return "", "", err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	_, err = f.Write(buf)
	if err != nil {
		return "", "", err
	}
	err = f.Sync()
	if err != nil {
		return "", "", err
	}
	f.Close()
	f = nil
	hash, err := utils.CalcPathSHA256(n.Cli.IproofFile)
	if err != nil {
		return "", "", err
	}
	sigma := key.AggrGenProof(qslice, ptags)
	return sigma, hash, nil
}

func (n *Node) serviceAggrProof(key *proof.RSAKeyPair, random []byte, start uint32) (string, string, error) {
	serviceRoothashs, err := n.Cach.QueryPrefixKeyListByHeigh(Cach_prefix_metadata, start)
	if err != nil {
		return "", "", err
	}

	var buf []byte
	var tag pb.Tag
	var actualCount int
	var pf ProofFileType
	var ptags []proof.Tag = make([]proof.Tag, 0)
	var ptag proof.Tag
	var pf_mu ProofMuFileType
	var proveResponse proof.GenProofResponse
	pf.Name = make([]string, len(serviceRoothashs))
	pf.U = make([]string, len(serviceRoothashs))

	var qslice = []proof.QElement{
		{
			I: 0,
			V: string(random),
		},
	}

	for i := int(0); i < len(serviceRoothashs); i++ {
		files, err := utils.DirFiles(filepath.Join(n.Cli.FileDir, serviceRoothashs[i]), 0)
		if err != nil {
			continue
		}
		for j := 0; j < len(files); j++ {
			buf, err = os.ReadFile(files[j])
			if err != nil {
				continue
			}
			err = json.Unmarshal(buf, &tag)
			if err != nil {
				continue
			}
			matrix, _, err := proof.SplitByN(files[j], int64(len(tag.T.Phi)))
			if err != nil {
				continue
			}

			ptag.T.Name = tag.T.Name
			ptag.T.Phi = tag.T.Phi
			ptag.T.U = tag.T.U
			ptag.PhiHash = tag.PhiHash
			ptag.Attest = tag.Attest

			proveResponseCh := key.GenProof(qslice, nil, ptag, matrix)
			timeout := time.NewTimer(time.Duration(time.Minute))
			defer timeout.Stop()
			select {
			case proveResponse = <-proveResponseCh:
			case <-timeout.C:
				proveResponse.StatueMsg.StatusCode = 0
			}

			if proveResponse.StatueMsg.StatusCode != proof.Success {
				continue
			}
			ptags = append(ptags, ptag)
			pf.Name[actualCount] = tag.T.Name
			pf.Name[actualCount] = tag.T.U
			pf_mu.Mu[actualCount] = proveResponse.MU
			actualCount++
		}
	}
	buf, err = json.Marshal(&pf)
	if err != nil {
		return "", "", err
	}
	f, err := os.OpenFile(n.Cli.SproofFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return "", "", err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	_, err = f.Write(buf)
	if err != nil {
		return "", "", err
	}
	err = f.Sync()
	if err != nil {
		return "", "", err
	}
	f.Close()
	f = nil
	hash, err := utils.CalcPathSHA256(n.Cli.SproofFile)
	if err != nil {
		return "", "", err
	}
	sigma := key.AggrGenProof(qslice, ptags)
	return sigma, hash, nil
}

// func submitProofResult(proofs []chain.ProveInfo) {
// 	var (
// 		err      error
// 		tryCount uint8
// 		txhash   string
// 	)
// 	// submit proof results
// 	if len(proofs) > 0 {
// 		// fmt.Println("---------------")
// 		// fmt.Println("FileId:", string(proofs[0].FileId[:]))
// 		// fmt.Println("chal:", proofs[0].Cinfo)
// 		// fmt.Println("u:", proofs[0].U)
// 		// fmt.Println("mu:", proofs[0].Mu)
// 		// fmt.Println("sigma:", proofs[0].Sigma)
// 		// fmt.Println("Omega:", proofs[0].Omega)
// 		// fmt.Println("SigRoothash:", proofs[0].SigRootHash)
// 		// fmt.Println("HashMi:", proofs[0].HashMi)
// 		// fmt.Println("---------------")

// 		for {
// 			txhash, err = chain.SubmitProofs(proofs)
// 			if err != nil {
// 				tryCount++
// 				//Chg.Sugar().Errorf("Proof result submitted err: %v", err)
// 			}
// 			if txhash != "" {
// 				//Chg.Sugar().Infof("Proof result submitted suc: %v", txhash)
// 				return
// 			}
// 			if tryCount >= 3 {
// 				return
// 			}
// 			time.Sleep(configs.BlockInterval)
// 		}
// 	}
// 	return
// }

// func calcProof(challenge chain.ChallengesInfo) chain.ProveInfo {
// 	var (
// 		err             error
// 		fileid          string
// 		shardId         string
// 		fileFullPath    string
// 		fileTagFullPath string
// 		filetag         proof.StorageTagType
// 		proveResponse   proof.GenProofResponse
// 		proveInfoTemp   chain.ProveInfo
// 	)

// 	proveInfoTemp.Cinfo = challenge
// 	proveInfoTemp.FileId = challenge.File_id
// 	acc, _ := types.NewAccountID(pattern.GetMinerAcc())
// 	proveInfoTemp.MinerAcc = *acc

// 	fileid = string(challenge.File_id[:])
// 	if challenge.File_type == 1 {
// 		//space file
// 		fileFullPath = filepath.Join(configs.SpaceDir, fileid)
// 		fileTagFullPath = filepath.Join(configs.SpaceDir, fileid+".tag")
// 	} else {
// 		//user file
// 		shardId = string(challenge.Shard_id[:])
// 		fileid = strings.Split(shardId, ".")[0]
// 		fileFullPath = filepath.Join(configs.FileDir, shardId)
// 		fileTagFullPath = filepath.Join(configs.FileDir, shardId+".tag")
// 	}

// 	_, err = os.Stat(fileFullPath)
// 	if err != nil {
// 		//Chg.Sugar().Errorf("[%v] %v", fileid, err)
// 		return proveInfoTemp
// 	}

// 	qSlice, err := proof.PoDR2ChallengeGenerateFromChain(challenge.Block_list, challenge.Random)
// 	if err != nil {
// 		//Chg.Sugar().Errorf("[%v] %v", fileid, err)
// 		return proveInfoTemp
// 	}

// 	ftag, err := ioutil.ReadFile(fileTagFullPath)
// 	if err != nil {
// 		//Chg.Sugar().Errorf("[%v] %v", fileid, err)
// 		return proveInfoTemp
// 	}

// 	err = json.Unmarshal(ftag, &filetag)
// 	if err != nil {
// 		//Chg.Sugar().Errorf("[%v] %v", fileid, err)
// 		return proveInfoTemp
// 	}

// 	proveInfoTemp.U = filetag.T.U

// 	matrix, _, err := proof.SplitV2(fileFullPath, configs.BlockSize)
// 	if err != nil {
// 		//Chg.Sugar().Errorf("[%v] %v", fileid, err)
// 		return proveInfoTemp
// 	}

// 	E_bigint, _ := new(big.Int).SetString(filetag.E, 10)
// 	N_bigint, _ := new(big.Int).SetString(filetag.N, 10)

// 	fmt.Println("Will gen proof: ", string(challenge.File_id[:]))
// 	proveResponseCh := proof.GetKey(int(E_bigint.Int64()), N_bigint).GenProof(qSlice, filetag.T, filetag.Phi, matrix, filetag.SigRootHash)
// 	select {
// 	case proveResponse = <-proveResponseCh:
// 		if proveResponse.StatueMsg.StatusCode != proof.Success {
// 			return proveInfoTemp
// 		}
// 	}
// 	fmt.Println("Gen proof suc: ", string(challenge.File_id[:]))
// 	fmt.Println()

// 	// Chg.Sugar().Infof("fileid: %v", fileid)
// 	// Chg.Sugar().Infof("len(MU)", len(proveResponse.MU))
// 	// Chg.Sugar().Infof("len(Sigma)", len(proveResponse.Sigma))
// 	// Chg.Sugar().Infof("len(Omega)", len(proveResponse.Omega))
// 	// Chg.Sugar().Infof("len(SigRootHash)", len(proveResponse.SigRootHash))
// 	// Chg.Sugar().Infof("len(HashMi)", len(proveResponse.HashMi)*32)

// 	proveInfoTemp.Mu = proveResponse.MU
// 	proveInfoTemp.Sigma = proveResponse.Sigma
// 	proveInfoTemp.Omega = proveResponse.Omega
// 	proveInfoTemp.SigRootHash = proveResponse.SigRootHash
// 	proveInfoTemp.HashMi = make([]types.Bytes, len(proveResponse.HashMi))
// 	for i := 0; i < len(proveResponse.HashMi); i++ {
// 		proveInfoTemp.HashMi[i] = make(types.Bytes, 0)
// 		proveInfoTemp.HashMi[i] = append(proveInfoTemp.HashMi[i], proveResponse.HashMi[i]...)
// 	}
// 	return proveInfoTemp
// }

// func calcFileBlockSizeAndScanSize(fsize int64) (int64, int64) {
// 	var (
// 		blockSize     int64
// 		scanBlockSize int64
// 	)
// 	if fsize < configs.SIZE_1KiB {
// 		return fsize, fsize
// 	}
// 	if fsize > math.MaxUint32 {
// 		blockSize = math.MaxUint32
// 		scanBlockSize = blockSize / 8
// 		return blockSize, scanBlockSize
// 	}
// 	blockSize = fsize / 16
// 	scanBlockSize = blockSize / 8
// 	return blockSize, scanBlockSize
// }

// func split(filefullpath string, blocksize, filesize int64) ([][]byte, uint64, error) {
// 	file, err := os.Open(filefullpath)
// 	if err != nil {
// 		return nil, 0, err
// 	}
// 	defer file.Close()

// 	if filesize/blocksize == 0 {
// 		return nil, 0, errors.New("filesize invalid")
// 	}
// 	n := filesize / blocksize
// 	if n == 0 {
// 		n = 1
// 	}
// 	// matrix is indexed as m_ij, so the first dimension has n items and the second has s.
// 	matrix := make([][]byte, n)
// 	for i := int64(0); i < n; i++ {
// 		piece := make([]byte, blocksize)
// 		_, err := file.Read(piece)
// 		if err != nil {
// 			return nil, 0, err
// 		}
// 		matrix[i] = piece
// 	}
// 	return matrix, uint64(n), nil
// }