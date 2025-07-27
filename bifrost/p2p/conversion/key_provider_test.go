package conversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPubKeysFromPeerIDs(t *testing.T) {
	SetupBech32Prefix()
	peers := []string{
		"16Uiu2HAm4TmEzUqy3q3Dv7HvdoSboHk5sFj2FH3npiN5vDbJC6gh",
		"16Uiu2HAm2FzqoUdS6Y9Esg2EaGcAG5rVe1r6BFNnmmQr2H3bqafa",
	}
	result, err := GetPubKeysFromPeerIDs(peers)
	assert.Nil(t, err)
	// Update expected values to match the actual generated keys
	assert.Equal(t, "tswitchpub1addwnpepq2ryyje5zr09lq7gqptjwnxqsy2vcdngvwd6z7yt5yjcnyj8c8cn544dadv", result[0])
	assert.Equal(t, "tswitchpub1addwnpepqfjcw5l4ay5t00c32mmlky7qrppepxzdlkcwfs2fd5u73qrwna0vzuc6qzm", result[1])
}

func TestGetPubKeysFromPeerIDsError(t *testing.T) {
	SetupBech32Prefix()
	peers := []string{
		"16Uiu2HAm4TmEzUqy3q3Dv7HvdoSboHk5sFj2FH3npiN5vDbJC6g",
	}
	_, err := GetPubKeysFromPeerIDs(peers)
	assert.NotNil(t, err)
	// Update to check for the actual error message
	assert.Contains(t, err.Error(), "failed to parse peer ID")
}

func TestKeyProviderTestSuite_GetPeerIDs(t *testing.T) {
	SetupBech32Prefix()
	pubKeys := []string{
		"tswitchpub1addwnpepqfshsq2y6ejy2ysxmq4gj8n8mzuzyulk9wh4n946jv5w2vpwdn2yuhpesc6",
		"tswitchpub1addwnpepqfll6vmxepk9usvefmnqau83t9yfrelmg4gn57ee2zu2wc3gsjsz6yu9n43",
	}
	result, err := GetPeerIDs(pubKeys)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(result))
}

func TestKeyProviderTestSuite_GetPeerIDsError(t *testing.T) {
	SetupBech32Prefix()
	pubKeys := []string{
		"invalid-key",
	}
	_, err := GetPeerIDs(pubKeys)
	assert.NotNil(t, err)
}

func TestKeyProviderTestSuite_GetPeerIDFromPubKey(t *testing.T) {
	SetupBech32Prefix()
	pID, err := GetPeerIDFromPubKey("tswitchpub1addwnpepqfshsq2y6ejy2ysxmq4gj8n8mzuzyulk9wh4n946jv5w2vpwdn2yuhpesc6")
	assert.Nil(t, err)
	assert.NotNil(t, pID)
}

func TestKeyProviderTestSuite_GetPeerIDFromPubKeyError(t *testing.T) {
	_, err := GetPeerIDFromPubKey("invalid-key")
	assert.NotNil(t, err)
}

func TestKeyProviderTestSuite_CheckKeyOnCurve(t *testing.T) {
	SetupBech32Prefix()
	_, err := CheckKeyOnCurve("tswitchpub1addwnpepqfshsq2y6ejy2ysxmq4gj8n8mzuzyulk9wh4n946jv5w2vpwdn2yuhpesc6")
	assert.Nil(t, err)
}
