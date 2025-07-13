//go:build !mocknet && !stagenet
// +build !mocknet,!stagenet

package cmd

const (
	Bech32PrefixAccAddr  = "switch"
	Bech32PrefixAccPub   = "switchpub"
	Bech32PrefixValAddr  = "switchv"
	Bech32PrefixValPub   = "switchvpub"
	Bech32PrefixConsAddr = "switchc"
	Bech32PrefixConsPub  = "switchcpub"
)
