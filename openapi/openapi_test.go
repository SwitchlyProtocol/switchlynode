package openapi

// The openapi package contains generated types based on the OpenAPI spec. These types
// are leveraged in the thornode querier handlers where applicable, but many of the
// querier responses leverage existing types generated from protobuf definitions. In
// these cases we add tests to ensure that the generated types from the API spec should
// at least have matching struct tags with those from the types used in the querier
// responses to ensure the API spec is accurate and can be used to generate clients.

import (
	"reflect"
	"testing"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	gen "github.com/switchlyprotocol/switchlynode/v1/openapi/gen"
	types "github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"

	. "gopkg.in/check.v1"
)

// -------------------------------------------------------------------------------------
// Init
// -------------------------------------------------------------------------------------

func TestPackage(t *testing.T) { TestingT(t) }

type Test struct{}

var _ = Suite(&Test{})

// -------------------------------------------------------------------------------------
// Tests
// -------------------------------------------------------------------------------------

func (Test) TestJSONSpec(c *C) {
	// common
	assertJSONStructTagsMatch(c, common.Coin{}, gen.Coin{})
	assertJSONStructTagsMatch(c, common.Tx{}, gen.Tx{})

	// queue and lp
	assertJSONStructTagsMatch(c, types.MsgSwap{}, gen.MsgSwap{})

	// txs
	assertJSONStructTagsMatch(c, types.TxOut{}, gen.KeysignInfo{})
	// TODO: Check that TxOutItem struct tags match
	// if or when the THORNode struct includes its (scheduled) Height field.
	// assertJSONStructTagsMatch(c, types.TxOutItem{}, gen.TxOutItem{})

	// tss
	assertJSONStructTagsMatch(c, types.NodeTssTime{}, gen.NodeKeygenMetric{})
	assertJSONStructTagsMatch(c, types.TssKeygenMetric{}, gen.KeygenMetric{})
	assertJSONStructTagsMatch(c, types.TssKeysignMetric{}, gen.TssKeysignMetric{})

	// miscellaneous
	assertJSONStructTagsMatch(c, types.BanVoter{}, gen.BanResponse{})
}

// -------------------------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------------------------

func assertJSONStructTagsMatch(c *C, thor, spec interface{}) {
	thorType := reflect.TypeOf(thor)
	specType := reflect.TypeOf(spec)
	comment := Commentf("thorType=%s; specType=%s", thorType.Name(), specType.Name())

	c.Assert(specType.NumField(), Equals, thorType.NumField(), comment)
	for i := 0; i < thorType.NumField(); i++ {
		specTag := specType.Field(i).Tag.Get("json")
		thorTag := thorType.Field(i).Tag.Get("json")
		c.Assert(specTag, Equals, thorTag, comment)
	}
}
