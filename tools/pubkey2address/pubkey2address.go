package main

import (
	"flag"
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

func main() {
	raw := flag.String("p", "", "thor bech32 pubkey")
	flag.Parse()

	if len(*raw) == 0 {
		panic("no pubkey provided")
	}

	// Read in the configuration file for the sdk
	nw := common.CurrentChainNetwork
	switch nw {
	case common.MockNet:
		fmt.Println("THORChain mocknet:")
		config := cosmos.GetConfig()
		config.SetBech32PrefixForAccount("tthor", "tthorpub")
		config.SetBech32PrefixForValidator("tthorv", "tthorvpub")
		config.SetBech32PrefixForConsensusNode("tthorc", "tthorcpub")
		config.Seal()
	case common.MainNet:
		fmt.Println("THORChain mainnet:")
		config := cosmos.GetConfig()
		config.SetBech32PrefixForAccount("thor", "thorpub")
		config.SetBech32PrefixForValidator("thorv", "thorvpub")
		config.SetBech32PrefixForConsensusNode("thorc", "thorcpub")
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
