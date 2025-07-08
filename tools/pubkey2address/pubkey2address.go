package main

import (
	"flag"
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

func main() {
	raw := flag.String("p", "", "switchly bech32 pubkey")
	flag.Parse()

	if len(*raw) == 0 {
		panic("no pubkey provided")
	}

	// Read in the configuration file for the sdk
	nw := common.CurrentChainNetwork
	switch nw {
	case common.MockNet:
		fmt.Println("SwitchlyProtocol mocknet:")
		config := cosmos.GetConfig()
		config.SetBech32PrefixForAccount("tswtc", "tswtcpub")
		config.SetBech32PrefixForValidator("tswtcv", "tswtcvpub")
		config.SetBech32PrefixForConsensusNode("tswtcc", "tswtccpub")
		config.Seal()
	case common.MainNet:
		fmt.Println("SwitchlyProtocol mainnet:")
		config := cosmos.GetConfig()
		config.SetBech32PrefixForAccount("swtc", "swtcpub")
		config.SetBech32PrefixForValidator("swtcv", "swtcvpub")
		config.SetBech32PrefixForConsensusNode("swtcc", "swtccpub")
		config.Seal()
	}

	pk, err := common.NewPubKey(*raw)
	if err != nil {
		panic(err)
	}

	for _, chain := range common.AllChains {
		var addr common.Address
		addr, err = pk.GetAddress(chain)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s Address: %s\n", chain.String(), addr)
	}
}
