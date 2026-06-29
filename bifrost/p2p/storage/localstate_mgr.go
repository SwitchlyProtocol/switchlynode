package storage

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-peerstore/addr"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/p2p/conversion"
)

// Keyshare algorithms. The empty value is treated as ECDSA for backward compatibility with state
// written before EdDSA support existed. These mirror bifrost/tss/go-tss/common.Algo but are kept as
// plain strings here to avoid a package import cycle.
const (
	AlgoECDSA = "ecdsa"
	AlgoEdDSA = "eddsa"
)

// KeygenLocalState is a structure used to represent the data we saved locally for different keygen.
//
// A vault has its own keyshare per signing algorithm: secp256k1 (ECDSA, used by every chain today) in
// LocalData, and optionally ed25519 (EdDSA, required for Stellar) in EdDSALocalData. Algo records
// which scheme this state is for; an empty Algo means ECDSA (legacy state).
//
// EdDSALocalData is stored as opaque JSON (the marshaled tss-lib eddsa/keygen.LocalPartySaveData) on
// purpose: this package is linked into every bifrost binary, and importing tss-lib's eddsa/keygen here
// alongside its ecdsa/keygen panics at init (both register protobuf messages under the same names in
// the "protob" package). The EdDSA keygen/keysign code paths marshal/unmarshal this field. See the
// proto-namespace blocker in docs/architecture/stellar-eddsa-tss.md.
type KeygenLocalState struct {
	PubKey          string                    `json:"pub_key"`
	LocalData       keygen.LocalPartySaveData `json:"local_data"`
	EdDSALocalData  json.RawMessage           `json:"eddsa_local_data,omitempty"`
	Algo            string                    `json:"algo,omitempty"`
	ParticipantKeys []string                  `json:"participant_keys"` // the participant of last key gen
	LocalPartyKey   string                    `json:"local_party_key"`
}

// LocalStateManager provide necessary methods to manage the local state, save it , and read it back
// LocalStateManager doesn't have any opinion in regards to where it should be persistent to
type LocalStateManager interface {
	SaveLocalState(state KeygenLocalState) error
	GetLocalState(pubKey string) (KeygenLocalState, error)
	SaveAddressBook(addressBook map[peer.ID]addr.AddrList) error
	RetrieveP2PAddresses() (addr.AddrList, error)
}

// FileStateMgr save the local state to file
type FileStateMgr struct {
	folder    string
	writeLock *sync.RWMutex
}

// NewFileStateMgr create a new instance of the FileStateMgr which implements LocalStateManager
func NewFileStateMgr(folder string) (*FileStateMgr, error) {
	if len(folder) > 0 {
		_, err := os.Stat(folder)
		if err != nil && os.IsNotExist(err) {
			if err := os.MkdirAll(folder, os.ModePerm); err != nil {
				return nil, err
			}
		}
	}
	return &FileStateMgr{
		folder:    folder,
		writeLock: &sync.RWMutex{},
	}, nil
}

func (fsm *FileStateMgr) getFilePathName(pubKey string) (string, error) {
	ret, err := conversion.CheckKeyOnCurve(pubKey)
	if err != nil {
		// EdDSA group keys are hex-encoded 32-byte ed25519 keys, not bech32 secp256k1 keys, so the
		// secp256k1 on-curve check above fails for them. Accept a well-formed ed25519 hex key (it is
		// filename-safe) so EdDSA keyshares can be persisted/loaded.
		if b, herr := hex.DecodeString(pubKey); herr == nil && len(b) == ed25519.PublicKeySize {
			ret = true
		} else {
			return "", err
		}
	}
	if !ret {
		return "", errors.New("invalid pubkey for file name")
	}

	localFileName := fmt.Sprintf("localstate-%s.json", pubKey)
	if len(fsm.folder) > 0 {
		return filepath.Join(fsm.folder, localFileName), nil
	}
	return localFileName, nil
}

// SaveLocalState save the local state to file
func (fsm *FileStateMgr) SaveLocalState(state KeygenLocalState) error {
	buf, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("fail to marshal KeygenLocalState to json: %w", err)
	}
	filePathName, err := fsm.getFilePathName(state.PubKey)
	if err != nil {
		return err
	}
	return os.WriteFile(filePathName, buf, 0o600)
}

// GetLocalState read the local state from file system
func (fsm *FileStateMgr) GetLocalState(pubKey string) (KeygenLocalState, error) {
	if len(pubKey) == 0 {
		return KeygenLocalState{}, errors.New("pub key is empty")
	}
	filePathName, err := fsm.getFilePathName(pubKey)
	if err != nil {
		return KeygenLocalState{}, err
	}
	if _, err := os.Stat(filePathName); os.IsNotExist(err) {
		return KeygenLocalState{}, err
	}

	buf, err := os.ReadFile(filePathName)
	if err != nil {
		return KeygenLocalState{}, fmt.Errorf("file to read from file(%s): %w", filePathName, err)
	}
	var localState KeygenLocalState
	if err := json.Unmarshal(buf, &localState); nil != err {
		return KeygenLocalState{}, fmt.Errorf("fail to unmarshal KeygenLocalState: %w", err)
	}
	return localState, nil
}

func (fsm *FileStateMgr) SaveAddressBook(address map[peer.ID]addr.AddrList) error {
	if len(fsm.folder) < 1 {
		return errors.New("base file path is invalid")
	}
	filePathName := filepath.Join(fsm.folder, "address_book.seed")
	var buf bytes.Buffer

	for peer, addrs := range address {
		for _, addr := range addrs {
			// we do not save the loopback addr
			if strings.Contains(addr.String(), "127.0.0.1") {
				continue
			}
			record := addr.String() + "/p2p/" + peer.String() + "\n"
			_, err := buf.WriteString(record)
			if err != nil {
				return errors.New("fail to write the record to buffer")
			}
		}
	}
	fsm.writeLock.Lock()
	defer fsm.writeLock.Unlock()
	return os.WriteFile(filePathName, buf.Bytes(), 0o600)
}

func (fsm *FileStateMgr) RetrieveP2PAddresses() (addr.AddrList, error) {
	if len(fsm.folder) < 1 {
		return nil, errors.New("base file path is invalid")
	}
	filePathName := filepath.Join(fsm.folder, "address_book.seed")

	_, err := os.Stat(filePathName)
	if err != nil {
		return nil, err
	}
	fsm.writeLock.RLock()
	input, err := os.ReadFile(filePathName)
	if err != nil {
		fsm.writeLock.RUnlock()
		return nil, err
	}
	fsm.writeLock.RUnlock()
	data := strings.Split(string(input), "\n")
	var peerAddresses []ma.Multiaddr
	for _, el := range data {
		// we skip the empty entry
		if len(el) == 0 {
			continue
		}
		addr, err := ma.NewMultiaddr(el)
		if err != nil {
			return nil, fmt.Errorf("invalid address in address book %w", err)
		}
		peerAddresses = append(peerAddresses, addr)
	}
	return peerAddresses, nil
}
