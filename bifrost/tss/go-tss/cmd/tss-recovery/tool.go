package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/btcsuite/btcd/btcec"
	coskey "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bech32 "github.com/cosmos/cosmos-sdk/types/bech32/legacybech32"
	"github.com/switchlyprotocol/switchlynode/v3/cmd"
)

type (
	KeygenLocalState struct {
		PubKey          string                    `json:"pub_key"`
		LocalData       keygen.LocalPartySaveData `json:"local_data"`
		ParticipantKeys []string                  `json:"participant_keys"` // the paticipant of last key gen
		LocalPartyKey   string                    `json:"local_party_key"`
	}
)

func getTssSecretFile(file string) (KeygenLocalState, error) {
	_, err := os.Stat(file)
	if err != nil {
		return KeygenLocalState{}, err
	}
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return KeygenLocalState{}, fmt.Errorf("file to read from file(%s): %w", file, err)
	}
	var localState KeygenLocalState
	if err := json.Unmarshal(buf, &localState); nil != err {
		return KeygenLocalState{}, fmt.Errorf("fail to unmarshal KeygenLocalState: %w", err)
	}
	return localState, nil
}

func setupBech32Prefix() {
	config := sdk.GetConfig()
	// switchly will import go-tss as a library , thus this is not needed, we copy the prefix here to avoid go-tss to import switchly
	config.SetBech32PrefixForAccount(cmd.Bech32PrefixAccAddr, cmd.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(cmd.Bech32PrefixValAddr, cmd.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(cmd.Bech32PrefixConsAddr, cmd.Bech32PrefixConsPub)
}

func getTssPubKey(x, y *big.Int) (string, sdk.AccAddress, error) {
	if x == nil || y == nil {
		return "", sdk.AccAddress{}, errors.New("invalid points")
	}
	tssPubKey := btcec.PublicKey{
		Curve: btcec.S256(),
		X:     x,
		Y:     y,
	}
	pubKeyCompressed := coskey.PubKey{
		Key: tssPubKey.SerializeCompressed(),
	}

	pubKey, err := bech32.MarshalPubKey(bech32.AccPK, &pubKeyCompressed)
	addr := sdk.AccAddress(pubKeyCompressed.Address().Bytes())
	return pubKey, addr, err
}

func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	// AES-128 is selected due to size of encryptKey.
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}
	return b, nil
}
