package p2p

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/libp2p/go-libp2p-core/peer"
	maddr "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p-core/crypto"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/p2p/messages"
)

type CommunicationTestSuite struct{}

var _ = Suite(&CommunicationTestSuite{})

func (CommunicationTestSuite) TestBasicCommunication(c *C) {
	comm, err := NewCommunication(&Config{Port: 6688, RendezvousString: "rendezvous"}, nil)
	c.Assert(err, IsNil)
	c.Assert(comm, NotNil)
	comm.SetSubscribe(messages.TSSKeyGenMsg, "hello", make(chan *Message))
	c.Assert(comm.getSubscriber(messages.TSSKeySignMsg, "hello"), IsNil)
	c.Assert(comm.getSubscriber(messages.TSSKeyGenMsg, "hello"), NotNil)
	comm.CancelSubscribe(messages.TSSKeyGenMsg, "hello")
	comm.CancelSubscribe(messages.TSSKeyGenMsg, "whatever")
	comm.CancelSubscribe(messages.TSSKeySignMsg, "asdsdf")
}

func checkExist(a []maddr.Multiaddr, b string) bool {
	for _, el := range a {
		if el.String() == b {
			return true
		}
	}
	return false
}

func (CommunicationTestSuite) TestEstablishP2pCommunication(c *C) {
	bootstrapPeer := "/ip4/127.0.0.1/tcp/2220/p2p/16Uiu2HAm4TmEzUqy3q3Dv7HvdoSboHk5sFj2FH3npiN5vDbJC6gh"
	bootstrapPrivKey := "6LABmWB4iXqkqOJ9H0YFEA2CSSx6bA7XAKGyI/TDtas="
	fakeExternalIP := "11.22.33.44"
	fakeExternalMultiAddr := "/ip4/11.22.33.44/tcp/2220"
	validMultiAddr, err := maddr.NewMultiaddr(bootstrapPeer)
	c.Assert(err, IsNil)
	privKey, err := base64.StdEncoding.DecodeString(bootstrapPrivKey)
	c.Assert(err, IsNil)
	comm, err := NewCommunication(&Config{Port: 2220, RendezvousString: "commTest", ExternalIP: fakeExternalIP}, nil)
	c.Assert(err, IsNil)
	c.Assert(comm.Start(privKey), IsNil)

	defer func() {
		err := comm.Stop()
		c.Assert(err, IsNil)
	}()
	sk1, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	sk1raw, _ := sk1.Raw()
	c.Assert(err, IsNil)
	comm2, err := NewCommunication(&Config{Port: 2221, RendezvousString: "commTest", BootstrapPeers: []maddr.Multiaddr{validMultiAddr}}, nil)
	c.Assert(err, IsNil)
	err = comm2.Start(sk1raw)
	c.Assert(err, IsNil)
	defer func() {
		err := comm2.Stop()
		c.Assert(err, IsNil)
	}()

	// we connect to an invalid peer and see
	sk2, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	c.Assert(err, IsNil)
	id, err := peer.IDFromPrivateKey(sk2)
	c.Assert(err, IsNil)
	invalidAddr := "/ip4/127.0.0.1/tcp/2220/p2p/" + id.String()
	invalidMultiAddr, err := maddr.NewMultiaddr(invalidAddr)
	c.Assert(err, IsNil)
	comm3, err := NewCommunication(&Config{Port: 2222, RendezvousString: "commTest", BootstrapPeers: []maddr.Multiaddr{invalidMultiAddr}}, nil)
	c.Assert(err, IsNil)
	err = comm3.Start(sk1raw)
	c.Assert(err, ErrorMatches, "fail to connect to bootstrap peer: fail to connect to any peer")
	defer func() {
		err := comm3.Stop()
		c.Assert(err, IsNil)
	}()

	// we connect to one invalid and one valid address
	comm4, err := NewCommunication(&Config{Port: 2223, RendezvousString: "commTest", BootstrapPeers: []maddr.Multiaddr{invalidMultiAddr, validMultiAddr}}, nil)
	c.Assert(err, IsNil)
	err = comm4.Start(sk1raw)
	c.Assert(err, IsNil)
	defer func() {
		err := comm4.Stop()
		c.Assert(err, IsNil)
	}()

	// we add test for external ip advertising
	c.Assert(checkExist(comm.host.Addrs(), fakeExternalMultiAddr), Equals, true)
	ps := comm2.host.Peerstore()
	c.Assert(checkExist(ps.Addrs(comm.host.ID()), fakeExternalMultiAddr), Equals, true)
	ps = comm4.host.Peerstore()
	c.Assert(checkExist(ps.Addrs(comm.host.ID()), fakeExternalMultiAddr), Equals, true)
}
