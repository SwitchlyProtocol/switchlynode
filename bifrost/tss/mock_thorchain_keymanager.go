package tss

import (
	"encoding/base64"

	"github.com/cometbft/cometbft/crypto"
	ctypes "github.com/cosmos/cosmos-sdk/types"
)

// MockSwitchlyKeymanager is to mock the TSS , so as we could test it
type MockSwitchlyKeyManager struct{}

func (k *MockSwitchlyKeyManager) GetPrivKey() crypto.PrivKey {
	return nil
}

func (k *MockSwitchlyKeyManager) GetAddr() ctypes.AccAddress {
	return nil
}

func (k *MockSwitchlyKeyManager) ExportAsMnemonic() (string, error) {
	return "", nil
}

func (k *MockSwitchlyKeyManager) ExportAsPrivateKey() (string, error) {
	return "", nil
}

func (k *MockSwitchlyKeyManager) ExportAsKeyStore(password string) (*EncryptedKeyJSON, error) {
	return nil, nil
}

func (k *MockSwitchlyKeyManager) RemoteSign(msg []byte, poolPubKey string) ([]byte, []byte, error) {
	// this is the key we are using to test TSS keysign result in BTC chain
	// switchpub1addwnpepqwznsrgk2t5vn2cszr6ku6zned6tqxknugzw3vhdcjza284d7djp5xqy5rtn
	if poolPubKey == "switchpub1addwnpepqwznsrgk2t5vn2cszr6ku6zned6tqxknugzw3vhdcjza284d7djp5xqy5rtn" {
		msgToSign := base64.StdEncoding.EncodeToString(msg)
		if msgToSign == "wqYuqkdeLjxtkKjmeAK0fOZygdw8zZgsDaJX7mrqWRE=" {
			sig, err := getSignature("ku/n0D18euwqkgM0kZn0OVX9+D7wfDBIWBMya1SGxWg=", "fw0sE6osjVN6vQtr9WxFrOpdxizPz9etSTOKGdjDY9A=")
			return sig, nil, err
		} else {
			sig, err := getSignature("256CpfiML7BDP1nXqKRc3Fq01PALeKwpXYv9P/H3Xhk=", "LoX6cVND0JN8bbZSTsoJcwLCysAKhyYtB2BFM3sdP98=")
			return sig, nil, err
		}
	}
	if poolPubKey == "switchpub1addwnpepqw2k68efthm08f0f5akhjs6fk5j2pze4wkwt4fmnymf9yd463puruhwcxkd5" {
		msgToSign := base64.StdEncoding.EncodeToString(msg)
		switch msgToSign {
		case "BMxXf+K+1dYu3qGgvH59GXoxwwFfTnLjB7hHf3qflPk=":
			sig, err := getSignature("WGSFUPPCN0kTcXcylAIQXyAxO7OUC5YRjDRz9wmzpkk=", "RUIoqdza5Od9nMfU2teqbZJAeC+pTyHIbKq+72jJMfM=")
			return sig, nil, err
		case "7zpXFp0KDBebXPNc2ZGim8NQAY7GMwS7iwr4hl2tFZQ=":
			sig, err := getSignature("tCR9TWnSxn/HPr0T3I9XeneJ0dRmi2DqbOkcFPWIkNs=", "VAxipOj6ogfBci+WwJy4n9QfAjjhJk6WhQ1I8n6xEo4=")
			return sig, nil, err
		case "isIqvmEs/otDI3NC2C8zFr1DGu3k/p8g/1RdlE0KzBI=":
			sig, err := getSignature("Nkb9ZFkPpSi1i/GaJe6FkMZmx1IH2oDtnr0jGsycBF8=", "ZAQ0qbPtPtdAin5HVOMmMO6oJxwWT4T0GvqpeyGG168=")
			return sig, nil, err
		default:
			sig, err := getSignature("gVxKdVgWR+4OZLxFAu5uWWOPCxhGPqFtQAyVujqSuh8=", "JXPU4Li4spnonssxJS52r/hEBwt1iPFlvjwu8ZOe+F0=")
			return sig, nil, err
		}
	}
	if poolPubKey == "switchpub1addwnpepqtvzm6wa6ezgjj9l4sdvzcf64wf0wzs8x9mgjfhjp6tkzcvkyfyqgddpqp4" {
		msgToSign := base64.StdEncoding.EncodeToString(msg)
		if msgToSign == "PIZUt687khEYQizRpYbLyQgDw1Ou+xzbSrLQ8fTKiaw=" {
			sig, err := base64.StdEncoding.DecodeString("HxT9xOyBYuhHfK8iLSbPniJq6u6KYfJVmq28iO+/Sa44ocAuckpzs3g6zBelr4pUaxatoKixAaPt2UtlgPP2sA==")
			return sig, nil, err
		}
	}
	return nil, nil, nil
}

func (k *MockSwitchlyKeyManager) Start() {}
func (k *MockSwitchlyKeyManager) Stop()  {}
