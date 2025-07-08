//go:build !mocknet && !stagenet
// +build !mocknet,!stagenet

package cmd

const (
	Bech32PrefixAccAddr  = "swtc"
	Bech32PrefixAccPub   = "swtcpub"
	Bech32PrefixValAddr  = "swtcv"
	Bech32PrefixValPub   = "swtcvpub"
	Bech32PrefixConsAddr = "swtcc"
	Bech32PrefixConsPub  = "swtccpub"
)
