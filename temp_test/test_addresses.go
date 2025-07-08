package main

import (
	"fmt"
	"log"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
)

func main() {
	types.SetupConfigForTest()
	
	pubkey, err := common.NewPubKey("swtcpub1addwnpepqfpgf05gts34mk3mdq3dc7qz5yjssydw9xy3237ny30jzkd6v78qs35ztdg")
	if err != nil {
		log.Fatal("Invalid pubkey:", err)
	}
	
	fmt.Printf("Valid pubkey: %s\n", pubkey)
	
	chains := []common.Chain{common.BTCChain, common.LTCChain, common.DOGEChain, common.BCHChain}
	for _, chain := range chains {
		addr, err := pubkey.GetAddress(chain)
		if err != nil {
			log.Printf("Failed to get %s address: %v", chain, err)
			continue
		}
		fmt.Printf("%s address: %s\n", chain, addr)
	}
}
