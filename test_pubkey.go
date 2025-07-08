package main

import (
	"fmt"
	"log"

	"github.com/switchlyprotocol/switchlynode/v1/common"
)

func main() {
	// Test a pubkey from the signer tests
	pubkey, err := common.NewPubKey("swtcpub1addwnpepq06smgna9nln5432hudgaelwz67w8nygk3d69dhza8awt7zegcauv4qrdku")
	if err != nil {
		log.Fatal("Invalid pubkey:", err)
	}

	fmt.Printf("Valid pubkey: %s\n", pubkey)

	// Test getting BCH address
	addr, err := pubkey.GetAddress(common.BCHChain)
	if err != nil {
		log.Fatal("Failed to get BCH address:", err)
	}
	fmt.Printf("BCH address: %s\n", addr)
}
