package common

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

type KeyDataAddr struct {
	mainnet string
	mocknet string
}

type KeyData struct {
	priv     string
	pub      string
	addrBTC  KeyDataAddr
	addrLTC  KeyDataAddr
	addrBCH  KeyDataAddr
	addrETH  KeyDataAddr
	addrDOGE KeyDataAddr
	addrXLM  KeyDataAddr
}

type PubKeyTestSuite struct {
	keyData []KeyData
}

var _ = Suite(&PubKeyTestSuite{})

func (s *PubKeyTestSuite) SetUpSuite(_ *C) {
	s.keyData = []KeyData{
		{
			priv: "ef235aacf90d9f4aadd8c92e4b2562e1d9eb97f0df9ba3b508258739cb013db2",
			pub:  "02b4632d08485ff1df2db55b9dafd23347d1c47a457072a1e87be26896549a8737",
			addrETH: KeyDataAddr{
				mainnet: "0x3fd2d4ce97b082d4bce3f9fee2a3d60668d2f473",
				mocknet: "0x3fd2d4ce97b082d4bce3f9fee2a3d60668d2f473",
			},
			addrBTC: KeyDataAddr{
				mainnet: "bc1qj08ys4ct2hzzc2hcz6h2hgrvlmsjynawlht528",
				mocknet: "bcrt1qj08ys4ct2hzzc2hcz6h2hgrvlmsjynawhcf2xa",
			},
			addrLTC: KeyDataAddr{
				mainnet: "ltc1qj08ys4ct2hzzc2hcz6h2hgrvlmsjynawmt3sjh",
				mocknet: "rltc1qj08ys4ct2hzzc2hcz6h2hgrvlmsjynawf4nr3r",
			},
			addrBCH: KeyDataAddr{
				mainnet: "qzfuujzhpd2ugtp2lqt2a2aqdnlwzgj04cswjhml4x",
				mocknet: "qzfuujzhpd2ugtp2lqt2a2aqdnlwzgj04cwqq36m3u",
			},
			addrDOGE: KeyDataAddr{
				mainnet: "DJcczDr7oNvfj5qP17Qa7p9ZUNTfnYYDJC",
				mocknet: "mtzUk1zTJzTdyC8Pz6PPPyCHTEL5RLVyDJ",
			},
			addrXLM: KeyDataAddr{
				mainnet: "GAEWJCDU62BZNYCS4FDTTLOH7FI6EID6U7GEEOWRAMJTEUWN6VGYVX76",
				mocknet: "GAEWJCDU62BZNYCS4FDTTLOH7FI6EID6U7GEEOWRAMJTEUWN6VGYVX76",
			},
		},
		{
			priv: "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032",
			pub:  "037db227d7094ce215c3a0f57e1bcc732551fe351f94249471934567e0f5dc1bf7",
			addrETH: KeyDataAddr{
				mainnet: "0x970e8128ab834e8eac17ab8e3812f010678cf791",
				mocknet: "0x970e8128ab834e8eac17ab8e3812f010678cf791",
			},
			addrBTC: KeyDataAddr{
				mainnet: "bc1qzupk5lmc84r2dh738a9g3zscavannjy38ghlxu",
				mocknet: "bcrt1qzupk5lmc84r2dh738a9g3zscavannjy3084p2x",
			},
			addrLTC: KeyDataAddr{
				mainnet: "ltc1qzupk5lmc84r2dh738a9g3zscavannjy3r5dm7v",
				mocknet: "rltc1qzupk5lmc84r2dh738a9g3zscavannjy3320gac",
			},
			addrBCH: KeyDataAddr{
				mainnet: "qqtsx6nl0q75dfkl6yl54zy2rr4nkwwgjyzgwf5228",
				mocknet: "qqtsx6nl0q75dfkl6yl54zy2rr4nkwwgjyuxu04wwa",
			},
			addrDOGE: KeyDataAddr{
				mainnet: "D7EnB23qxiWTBGD1z9N6Ui7VXonCgY9eeE",
				mocknet: "mhcdvpCBUL3RRNW2y8LuksADWfecEmkzju",
			},
			addrXLM: KeyDataAddr{
				mainnet: "GAH3B6E4KHN3ABKLK7YRXYJW2OGJV6WHUNKHFWRKGUJAKJMPL7JEF5XT",
				mocknet: "GAH3B6E4KHN3ABKLK7YRXYJW2OGJV6WHUNKHFWRKGUJAKJMPL7JEF5XT",
			},
		},
		{
			priv: "e810f1d7d6691b4a7a73476f3543bd87d601f9a53e7faf670eac2c5b517d83bf",
			pub:  "03f98464e8d3fc8e275e34c6f8dc9b99aa244e37b0d695d0dfb8884712ed6d4d35",
			addrETH: KeyDataAddr{
				mainnet: "0xf6da288748ec4c77642f6c5543717539b3ae001b",
				mocknet: "0xf6da288748ec4c77642f6c5543717539b3ae001b",
			},
			addrBTC: KeyDataAddr{
				mainnet: "bc1qqqnde7kqe5sf96j6zf8jpzwr44dh4gkdek4rvd",
				mocknet: "bcrt1qqqnde7kqe5sf96j6zf8jpzwr44dh4gkd3ehaqh",
			},
			addrLTC: KeyDataAddr{
				mainnet: "ltc1qqqnde7kqe5sf96j6zf8jpzwr44dh4gkda2085a",
				mocknet: "rltc1qqqnde7kqe5sf96j6zf8jpzwr44dh4gkd05d5hf",
			},
			addrBCH: KeyDataAddr{
				mainnet: "qqqzdh86crxjpyh2tgfy7gyfcwk4k74ze522f8panm",
				mocknet: "qqqzdh86crxjpyh2tgfy7gyfcwk4k74ze55ympqehp",
			},
			addrDOGE: KeyDataAddr{
				mainnet: "D59u6XAtQCybdvFB4ZLWH4h8cK5SY8show",
				mocknet: "mfXkrKKDupWZt2YC3YKKZDjrbAwrBFwj8W",
			},
			addrXLM: KeyDataAddr{
				mainnet: "GDLXUHNFH4KFLJO77TGZ7RGAS2IX3YKP4IEWVLKCSFPHAX5RW5XKFZ65",
				mocknet: "GDLXUHNFH4KFLJO77TGZ7RGAS2IX3YKP4IEWVLKCSFPHAX5RW5XKFZ65",
			},
		},
		{
			priv: "a96e62ed3955e65be32703f12d87b6b5cf26039ecfa948dc5107a495418e5330",
			pub:  "02950e1cdfcb133d6024109fd489f734eeb4502418e538c28481f22bce276f248c",
			addrETH: KeyDataAddr{
				mainnet: "0xfabb9cc6ec839b1214bb11c53377a56a6ed81762",
				mocknet: "0xfabb9cc6ec839b1214bb11c53377a56a6ed81762",
			},
			addrBTC: KeyDataAddr{
				mainnet: "bc1q0s4mg25tu6termrk8egltfyme4q7sg3h0e56p3",
				mocknet: "bcrt1q0s4mg25tu6termrk8egltfyme4q7sg3h8kkydt",
			},
			addrLTC: KeyDataAddr{
				mainnet: "ltc1q0s4mg25tu6termrk8egltfyme4q7sg3ht9w7ep",
				mocknet: "rltc1q0s4mg25tu6termrk8egltfyme4q7sg3hemvd64",
			},
			addrBCH: KeyDataAddr{
				mainnet: "qp7zhdp230nf0y0vwcl9radyn0x5r6pzxuqvhx5vs3",
				mocknet: "qp7zhdp230nf0y0vwcl9radyn0x5r6pzxu7z9q4g5t",
			},
			addrDOGE: KeyDataAddr{
				mainnet: "DGTegdtiJ6Y9fteiWtHNS5bpjCJSrY4Kiz",
				mocknet: "mrqWSS33oi57uzwjVsGBiEeYi4ArRRWHV4",
			},
			addrXLM: KeyDataAddr{
				mainnet: "GDZTMC23WN3V6TIYM3PMC2RGZ34QDGS3BHGESWZDFPTXM4E7PKY7IVK5",
				mocknet: "GDZTMC23WN3V6TIYM3PMC2RGZ34QDGS3BHGESWZDFPTXM4E7PKY7IVK5",
			},
		},
		{
			priv: "9294f4d108465fd293f7fe299e6923ef71a77f2cb1eb6d4394839c64ec25d5c0",
			pub:  "0238383ee4d60176d27cf46f0863bfc6aea624fe9bfc7f4273cc5136d9eb483e4a",
			addrETH: KeyDataAddr{
				mainnet: "0x1f30a82340f08177aba70e6f48054917c74d7d38",
				mocknet: "0x1f30a82340f08177aba70e6f48054917c74d7d38",
			},
			addrBTC: KeyDataAddr{
				mainnet: "bc1qjw8h4l3dtz5xxc7uyh5ys70qkezspgfurtjs2p",
				mocknet: "bcrt1qjw8h4l3dtz5xxc7uyh5ys70qkezspgfutyswxm",
			},
			addrLTC: KeyDataAddr{
				mainnet: "ltc1qjw8h4l3dtz5xxc7uyh5ys70qkezspgfu8hg5j3",
				mocknet: "rltc1qjw8h4l3dtz5xxc7uyh5ys70qkezspgfu4f2839",
			},
			addrBCH: KeyDataAddr{
				mainnet: "qzfc77h794v2scmrmsj7sjreuzmy2q9p8sxs8lu34e",
				mocknet: "qzfc77h794v2scmrmsj7sjreuzmy2q9p8sc74ea43r",
			},
			addrDOGE: KeyDataAddr{
				mainnet: "DJbKker23xfz3ufxAbqUuQwp1EBibGJJHu",
				mocknet: "mtyBWSzMZaCxJ1xy9apJBZzXz648BZrpJg",
			},
			addrXLM: KeyDataAddr{
				mainnet: "GBGPFKHJGDSPAQYB6QLNVIHJMZN66P7ORZ5BHNTBTBWSJA24C7K56KW5",
				mocknet: "GBGPFKHJGDSPAQYB6QLNVIHJMZN66P7ORZ5BHNTBTBWSJA24C7K56KW5",
			},
		},
	}
}

// TestPubKey implementation
func (s *PubKeyTestSuite) TestPubKey(c *C) {
	_, pubKey, _ := testdata.KeyTestPubAddr()
	spk, err := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeAccPub, pubKey)
	c.Assert(err, IsNil)
	pk, err := NewPubKey(spk)
	c.Assert(err, IsNil)
	hexStr := pk.String()
	c.Assert(len(hexStr) > 0, Equals, true)
	pk1, err := NewPubKey(hexStr)
	c.Assert(err, IsNil)
	c.Assert(pk.Equals(pk1), Equals, true)

	result, err := json.Marshal(pk)
	c.Assert(err, IsNil)
	c.Log(result, Equals, fmt.Sprintf(`"%s"`, hexStr))
	var pk2 PubKey
	err = json.Unmarshal(result, &pk2)
	c.Assert(err, IsNil)
	c.Assert(pk2.Equals(pk), Equals, true)
}

func (s *PubKeyTestSuite) TestPubKeySet(c *C) {
	_, pubKey, _ := testdata.KeyTestPubAddr()
	spk, err := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeAccPub, pubKey)
	c.Assert(err, IsNil)
	pk, err := NewPubKey(spk)
	c.Assert(err, IsNil)

	c.Check(PubKeySet{}.Contains(pk), Equals, false)

	pks := PubKeySet{
		Secp256k1: pk,
	}
	c.Check(pks.Contains(pk), Equals, true)
	pks = PubKeySet{
		Ed25519: pk,
	}
	c.Check(pks.Contains(pk), Equals, true)
}

func (s *PubKeyTestSuite) TestEquals(c *C) {
	var pk1, pk2, pk3, pk4 PubKey
	_, pubKey1, _ := testdata.KeyTestPubAddr()
	tpk1, err1 := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeAccPub, pubKey1)
	c.Assert(err1, IsNil)
	pk1 = PubKey(tpk1)

	_, pubKey2, _ := testdata.KeyTestPubAddr()
	tpk2, err2 := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeAccPub, pubKey2)
	c.Assert(err2, IsNil)
	pk2 = PubKey(tpk2)

	_, pubKey3, _ := testdata.KeyTestPubAddr()
	tpk3, err3 := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeAccPub, pubKey3)
	c.Assert(err3, IsNil)
	pk3 = PubKey(tpk3)

	_, pubKey4, _ := testdata.KeyTestPubAddr()
	tpk4, err4 := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeAccPub, pubKey4)
	c.Assert(err4, IsNil)
	pk4 = PubKey(tpk4)

	c.Assert(PubKeys{
		pk1, pk2,
	}.Equals(nil), Equals, false)

	c.Assert(PubKeys{
		pk1, pk2, pk3,
	}.Equals(PubKeys{
		pk1, pk2,
	}), Equals, false)

	c.Assert(PubKeys{
		pk1, pk2, pk3, pk4,
	}.Equals(PubKeys{
		pk4, pk3, pk2, pk1,
	}), Equals, true)

	c.Assert(PubKeys{ // nolint
		pk1, pk2, pk3, pk4,
	}.Equals(PubKeys{
		pk1, pk2, pk3, pk4,
	}), Equals, true)
}
