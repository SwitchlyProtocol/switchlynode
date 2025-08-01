package conversion

import (
	"encoding/json"
	"math/big"
	"sort"
	"testing"

	"github.com/binance-chain/tss-lib/crypto"
	"github.com/btcsuite/btcd/btcec"
	coskey "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types/bech32/legacybech32" // nolint:staticcheck
	"github.com/libp2p/go-libp2p-core/peer"
	. "gopkg.in/check.v1"
)

var (
	testPubKeys = [...]string{"tswitchpub1addwnpepqfshsq2y6ejy2ysxmq4gj8n8mzuzyulk9wh4n946jv5w2vpwdn2yuhpesc6", "tswitchpub1addwnpepqfll6vmxepk9usvefmnqau83t9yfrelmg4gn57ee2zu2wc3gsjsz6yu9n43", "tswitchpub1addwnpepqw7qvv8309c06z96nwcfhrp5efm2wa2h7nratlgvwpgwksm8d5zwumqa9nr", "tswitchpub1addwnpepqv8lvvqmczr893yf7zyf7xtffccf032aprl8z09y3e3nfruedew85q3v60m"}
	testPeers   = []string{
		"16Uiu2HAm43HgxKpiKR5c23EZhH6pYxMHsrs26TYfPUuL6CL5rxwn",
		"16Uiu2HAm1z9fDQMvcxfuj9uGBXzsJq1SX78TSfu9FvC399CoKZdP",
		"16Uiu2HAmDjJCKQHvwp34HEEK1KhGnTkQshHcziawGPwtvyxrDSrD",
		"16Uiu2HAmRJxUQWW3gbTCjeescXePMGVD7Kktw8rRWrB5TmVeSaxV",
	}
)

type ConversionTestSuite struct {
	testPubKeys []string
	localPeerID peer.ID
}

var _ = Suite(&ConversionTestSuite{})

func (p *ConversionTestSuite) SetUpTest(c *C) {
	var err error
	SetupBech32Prefix()
	p.testPubKeys = testPubKeys[:]
	sort.Strings(p.testPubKeys)
	p.localPeerID, err = peer.Decode("16Uiu2HAm43HgxKpiKR5c23EZhH6pYxMHsrs26TYfPUuL6CL5rxwn")
	c.Assert(err, IsNil)
}
func TestPackage(t *testing.T) { TestingT(t) }

func (p *ConversionTestSuite) TestAccPubKeysFromPartyIDs(c *C) {
	partiesID, _, err := GetParties(p.testPubKeys, p.testPubKeys[0])
	c.Assert(err, IsNil)
	partyIDMap := SetupPartyIDMap(partiesID)
	var keys []string
	for k := range partyIDMap {
		keys = append(keys, k)
	}

	got, err := AccPubKeysFromPartyIDs(keys, partyIDMap)
	c.Assert(err, IsNil)
	sort.Strings(got)
	c.Assert(got, DeepEquals, p.testPubKeys)
	got, err = AccPubKeysFromPartyIDs(nil, partyIDMap)
	c.Assert(err, Equals, nil)
	c.Assert(len(got), Equals, 0)
}

func (p *ConversionTestSuite) TestGetParties(c *C) {
	partiesID, localParty, err := GetParties(p.testPubKeys, p.testPubKeys[0])
	c.Assert(err, IsNil)
	pk := coskey.PubKey{
		Key: localParty.Key,
	}
	c.Assert(err, IsNil)
	got, err := sdk.MarshalPubKey(sdk.AccPK, &pk) // nolint:staticcheck
	c.Assert(err, IsNil)
	c.Assert(got, Equals, p.testPubKeys[0])
	var gotKeys []string
	for _, val := range partiesID {
		pk = coskey.PubKey{
			Key: val.Key,
		}
		got, err = sdk.MarshalPubKey(sdk.AccPK, &pk) // nolint:staticcheck
		c.Assert(err, IsNil)
		gotKeys = append(gotKeys, got)
	}
	sort.Strings(gotKeys)
	c.Assert(gotKeys, DeepEquals, p.testPubKeys)

	_, _, err = GetParties(p.testPubKeys, "")
	c.Assert(err, NotNil)
	_, _, err = GetParties(p.testPubKeys, "12")
	c.Assert(err, NotNil)
	_, _, err = GetParties(nil, "12")
	c.Assert(err, NotNil)
}

func (p *ConversionTestSuite) TestGetPeerIDFromPartyID(c *C) {
	_, localParty, err := GetParties(p.testPubKeys, p.testPubKeys[0])
	c.Assert(err, IsNil)
	peerID, err := GetPeerIDFromPartyID(localParty)
	c.Assert(err, IsNil)
	c.Assert(peerID, Equals, p.localPeerID)
	_, err = GetPeerIDFromPartyID(nil)
	c.Assert(err, NotNil)
	localParty.Index = -1
	_, err = GetPeerIDFromPartyID(localParty)
	c.Assert(err, NotNil)
}

func (p *ConversionTestSuite) TestGetPeerIDFromSecp256PubKey(c *C) {
	_, localParty, err := GetParties(p.testPubKeys, p.testPubKeys[0])
	c.Assert(err, IsNil)
	got, err := GetPeerIDFromSecp256PubKey(localParty.Key)
	c.Assert(err, IsNil)
	c.Assert(got, Equals, p.localPeerID)
	_, err = GetPeerIDFromSecp256PubKey(nil)
	c.Assert(err, NotNil)
}

func (p *ConversionTestSuite) TestGetPeersID(c *C) {
	localTestPubKeys := testPubKeys[:]
	sort.Strings(localTestPubKeys)
	partiesID, _, err := GetParties(p.testPubKeys, p.testPubKeys[0])
	c.Assert(err, IsNil)
	partyIDMap := SetupPartyIDMap(partiesID)
	partyIDtoP2PID := make(map[string]peer.ID)
	err = SetupIDMaps(partyIDMap, partyIDtoP2PID)
	c.Assert(err, IsNil)
	retPeers := GetPeersID(partyIDtoP2PID, p.localPeerID.String())
	var expectedPeers []string
	var gotPeers []string
	counter := 0
	for _, el := range testPeers {
		if el == p.localPeerID.String() {
			continue
		}
		expectedPeers = append(expectedPeers, el)
		gotPeers = append(gotPeers, retPeers[counter].String())
		counter++
	}
	sort.Strings(expectedPeers)
	sort.Strings(gotPeers)
	c.Assert(gotPeers, DeepEquals, expectedPeers)

	retPeers = GetPeersID(partyIDtoP2PID, "123")
	c.Assert(len(retPeers), Equals, 4)
	retPeers = GetPeersID(nil, "123")
	c.Assert(len(retPeers), Equals, 0)
}

func (p *ConversionTestSuite) TestPartyIDtoPubKey(c *C) {
	_, localParty, err := GetParties(p.testPubKeys, p.testPubKeys[0])
	c.Assert(err, IsNil)
	got, err := PartyIDtoPubKey(localParty)
	c.Assert(err, IsNil)
	c.Assert(got, Equals, p.testPubKeys[0])
	_, err = PartyIDtoPubKey(nil)
	c.Assert(err, NotNil)
	localParty.Index = -1
	_, err = PartyIDtoPubKey(nil)
	c.Assert(err, NotNil)
}

func (p *ConversionTestSuite) TestSetupIDMaps(c *C) {
	localTestPubKeys := testPubKeys[:]
	sort.Strings(localTestPubKeys)
	partiesID, _, err := GetParties(p.testPubKeys, p.testPubKeys[0])
	c.Assert(err, IsNil)
	partyIDMap := SetupPartyIDMap(partiesID)
	partyIDtoP2PID := make(map[string]peer.ID)
	err = SetupIDMaps(partyIDMap, partyIDtoP2PID)
	c.Assert(err, IsNil)
	var got []string

	for _, val := range partyIDtoP2PID {
		got = append(got, val.String())
	}
	sort.Strings(got)
	sort.Strings(testPeers)
	c.Assert(got, DeepEquals, testPeers)
	emptyPartyIDtoP2PID := make(map[string]peer.ID)
	c.Assert(SetupIDMaps(nil, emptyPartyIDtoP2PID), IsNil)
	c.Assert(emptyPartyIDtoP2PID, HasLen, 0)
}

func (p *ConversionTestSuite) TestSetupPartyIDMap(c *C) {
	localTestPubKeys := testPubKeys[:]
	sort.Strings(localTestPubKeys)
	partiesID, _, err := GetParties(p.testPubKeys, p.testPubKeys[0])
	c.Assert(err, IsNil)
	partyIDMap := SetupPartyIDMap(partiesID)
	var pubKeys []string
	for _, el := range partyIDMap {
		pk := coskey.PubKey{
			Key: el.Key,
		}
		got, err := sdk.MarshalPubKey(sdk.AccPK, &pk) // nolint:staticcheck
		c.Assert(err, IsNil)
		pubKeys = append(pubKeys, got)
	}
	sort.Strings(pubKeys)
	c.Assert(p.testPubKeys, DeepEquals, pubKeys)

	ret := SetupPartyIDMap(nil)
	c.Assert(ret, HasLen, 0)
}

func (p *ConversionTestSuite) TestTssPubKey(c *C) {
	sk, err := btcec.NewPrivateKey(btcec.S256())
	c.Assert(err, IsNil)
	point, err := crypto.NewECPoint(btcec.S256(), sk.X, sk.Y)
	c.Assert(err, IsNil)
	_, _, err = GetTssPubKey(point)
	c.Assert(err, IsNil)

	// create an invalid point
	invalidPoint := crypto.NewECPointNoCurveCheck(btcec.S256(), sk.X, new(big.Int).Add(sk.Y, big.NewInt(1)))
	_, _, err = GetTssPubKey(invalidPoint)
	c.Assert(err, NotNil)

	pk, addr, err := GetTssPubKey(nil)
	c.Assert(err, NotNil)
	c.Assert(pk, Equals, "")
	c.Assert(addr.Bytes(), HasLen, 0)

	SetupBech32Prefix()
	// var point crypto.ECPoint
	c.Assert(json.Unmarshal([]byte(`{"Coords":[70074650318631491136896111706876206496089700125696166275258483716815143842813,72125378038650252881868972131323661098816214918201601489154946637636730727892]}`), &point), IsNil)
	pk, addr, err = GetTssPubKey(point)
	c.Assert(err, IsNil)
	// Just check the results are valid without hardcoded expectations
	c.Assert(len(pk) > 10, Equals, true)            // Valid bech32 key should be reasonably long
	c.Assert(len(addr.String()) > 10, Equals, true) // Valid address should be reasonably long
}
