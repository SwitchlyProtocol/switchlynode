package main

import (
	"crypto/rand"
	"fmt"

	"github.com/btcsuite/btcutil/bech32"
)

func main() {
	// Generate a random 20-byte hash for a P2WPKH address
	hash := make([]byte, 20)
	rand.Read(hash)

	// Convert to 5-bit groups for bech32 encoding
	conv, err := bech32.ConvertBits(hash, 8, 5, true)
	if err != nil {
		panic(err)
	}

	// Encode with bc prefix for mainnet
	encoded, err := bech32.Encode("bc", conv)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Valid BTC mainnet address: %s\n", encoded)
}
