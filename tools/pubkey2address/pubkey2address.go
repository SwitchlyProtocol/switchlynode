package main

import (
	"flag"
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
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
		fmt.Println("Switchly mocknet:")
		config := cosmos.GetConfig()
		config.SetBech32PrefixForAccount("tswitch", "tswitchpub")
		config.SetBech32PrefixForValidator("tswitchv", "tswitchvpub")
		config.SetBech32PrefixForConsensusNode("tswitchc", "tswitchcpub")
		config.Seal()
	case common.MainNet:
		fmt.Println("Switchly mainnet:")
		config := cosmos.GetConfig()
		config.SetBech32PrefixForAccount("switch", "switchpub")
		config.SetBech32PrefixForValidator("switchv", "switchvpub")
		config.SetBech32PrefixForConsensusNode("switchc", "switchcpub")
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
