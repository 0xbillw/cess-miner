/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package confile

import (
	"fmt"
	"os"
	"path"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const DefaultProfile = "conf.yaml"
const TempleteProfile = `# The rpc endpoint of the chain node
Rpc:
  - "ws://127.0.0.1:9948/"
  - "wss://testnet-rpc0.cess.cloud/ws/"
  - "wss://testnet-rpc1.cess.cloud/ws/"
# Staking account mnemonic
Mnemonic: "xxx xxx ... xxx"
# earnings account
EarningsAcc: cXxxx...xxx
# Service workspace
Workspace: /
# Service listening port
Port: 15001
# Maximum space used, the unit is GiB
UseSpace: 2000`

type Confile interface {
	Parse(fpath string, port int) error
	GetRpcAddr() []string
	GetServicePort() int
	GetWorkspace() string
	GetMnemonic() string
	GetEarningsAcc() string
	GetUseSpace() uint64
	GetPublickey() []byte
	GetAccount() string
	SetEarningsAcc(earnings string) error
}

type confile struct {
	Rpc         []string `name:"Rpc" toml:"Rpc" yaml:"Rpc"`
	Mnemonic    string   `name:"Mnemonic" toml:"Mnemonic" yaml:"Mnemonic"`
	EarningsAcc string   `name:"EarningsAcc" toml:"EarningsAcc" yaml:"EarningsAcc"`
	Workspace   string   `name:"Workspace" toml:"Workspace" yaml:"Workspace"`
	Port        int      `name:"Port" toml:"Port" yaml:"Port"`
	UseSpace    uint64   `name:"UseSpace" toml:"UseSpace" yaml:"UseSpace"`
}

func NewConfigfile() *confile {
	return &confile{}
}

func (c *confile) Parse(fpath string, port int) error {
	fstat, err := os.Stat(fpath)
	if err != nil {
		return errors.Errorf("Parse: %v", err)
	}
	if fstat.IsDir() {
		return errors.Errorf("The '%v' is not a file", fpath)
	}

	viper.SetConfigFile(fpath)
	viper.SetConfigType(path.Ext(fpath)[1:])

	err = viper.ReadInConfig()
	if err != nil {
		return errors.Errorf("ReadInConfig: %v", err)
	}
	err = viper.Unmarshal(c)
	if err != nil {
		return errors.Errorf("Unmarshal: %v", err)
	}

	_, err = signature.KeyringPairFromSecret(c.Mnemonic, 0)
	if err != nil {
		return errors.Errorf("Secret: %v", err)
	}

	if len(c.Rpc) == 0 {
		return errors.New("Rpc endpoint is empty")
	}

	if port != 0 {
		c.Port = port
	}
	if c.Port < 1024 {
		return errors.Errorf("Prohibit the use of system reserved port: %v", c.Port)
	}
	if c.Port > 65535 {
		return errors.New("The port number cannot exceed 65535")
	}

	utils.VerityAddress(c.EarningsAcc, utils.CESSChainTestPrefix)

	fstat, err = os.Stat(c.Workspace)
	if err != nil {
		err = os.MkdirAll(c.Workspace, configs.DirMode)
		if err != nil {
			return err
		}
	}

	if !fstat.IsDir() {
		return errors.Errorf("The '%v' is not a directory", c.Workspace)
	}

	return nil
}

func (c *confile) SetRpcAddr(rpc []string) {
	c.Rpc = rpc
}

func (c *confile) SetUseSpace(useSpace uint64) {
	c.UseSpace = useSpace
}

func (c *confile) SetServicePort(port int) error {
	if utils.OpenedPort(port) {
		return errors.New("This port is in use")
	}

	if port < 1024 {
		return errors.Errorf("Prohibit the use of system reserved port: %v", port)
	}
	if port > 65535 {
		return errors.New("The port number cannot exceed 65535")
	}
	c.Port = port
	return nil
}

func (c *confile) SetWorkspace(workspace string) error {
	fstat, err := os.Stat(workspace)
	if err != nil {
		fmt.Println(">>1")
		err = os.MkdirAll(workspace, configs.DirMode)
		if err != nil {
			fmt.Println(">>2")
			return err
		}
	} else {
		if !fstat.IsDir() {
			return fmt.Errorf("%s is not a directory", workspace)
		}
	}
	c.Workspace = workspace
	return nil
}

func (c *confile) SetEarningsAcc(earnings string) error {
	var err error
	if earnings != "" {
		err = utils.VerityAddress(earnings, utils.CESSChainTestPrefix)
		if err != nil {
			return err
		}
	}
	c.EarningsAcc = earnings
	return nil
}

func (c *confile) SetMnemonic(mnemonic string) error {
	_, err := signature.KeyringPairFromSecret(mnemonic, 0)
	if err != nil {
		return err
	}
	c.Mnemonic = mnemonic
	return nil
}

func (c *confile) GetRpcAddr() []string {
	return c.Rpc
}

func (c *confile) GetServicePort() int {
	return c.Port
}

func (c *confile) GetWorkspace() string {
	return c.Workspace
}

func (c *confile) GetMnemonic() string {
	return c.Mnemonic
}

func (c *confile) GetEarningsAcc() string {
	return c.EarningsAcc
}

func (c *confile) GetPublickey() []byte {
	key, _ := signature.KeyringPairFromSecret(c.GetMnemonic(), 0)
	return key.PublicKey
}

func (c *confile) GetAccount() string {
	acc, _ := utils.EncodeToCESSAddr(c.GetPublickey())
	return acc
}

func (c *confile) GetUseSpace() uint64 {
	return c.UseSpace
}
