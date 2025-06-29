package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/tokenlist"
	"github.com/switchlyprotocol/switchlynode/v1/config"
)

func main() {
	config.Init()

	failed := false
	for chain, chainConfig := range config.GetBifrost().GetChains() {
		if !chain.IsEVM() {
			continue
		}

		fmt.Printf("checking %s whitelist tokens...\n", chain)

		var tl tokenlist.EVMTokenList
		switch chain.String() {
		case common.ETHChain.String():
			tl = tokenlist.GetETHTokenList()
		case common.AVAXChain.String():
			tl = tokenlist.GetAVAXTokenList()
		case common.BSCChain.String():
			tl = tokenlist.GetBSCTokenList()
		case common.BASEChain.String():
			tl = tokenlist.GetBASETokenList()
		default:
			fmt.Printf("unsupported chain %s\n", chain)
			os.Exit(1)
		}

		tokenAddrs := map[string]bool{}
		for _, token := range tl.Tokens {
			// NOTE: Token lists are inconsistent on whether the addresses is EIP-55, but all
			// internal usage performs EqualFold, so it is safe to compare case-insensitively.
			tokenAddrs[strings.ToLower(token.Address)] = true
		}
		for _, token := range chainConfig.BlockScanner.WhitelistTokens {
			if !tokenAddrs[strings.ToLower(token)] {
				fmt.Printf("  \033[31m%s not in %s token list\033[0m\n", token, chain)
				failed = true
			} else {
				fmt.Printf("  %s in %s token list\n", token, chain)
			}
		}
	}

	if failed {
		os.Exit(1)
	}
}
