package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"google.golang.org/protobuf/proto"

	"github.com/switchlyprotocol/switchlynode/v1/api/types"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

const (
	// MinKeysharesBackupEntropy was selected based on a few spot checks of the entropy in
	// encrypted keyshares for mocknet, which were always greater than 7, this is just a
	// sanity check and is safe to set lower.
	MinKeysharesBackupEntropy float64 = 7
)

// MatchMnemonic will match substrings that look like a 12+ word mnemonic.
var MatchMnemonic = regexp.MustCompile(`([a-zA-Z]+ ){11}[a-zA-Z]+`)

var (
	_ sdk.Msg              = &MsgTssPool{}
	_ sdk.HasValidateBasic = &MsgTssPool{}
	_ sdk.LegacyMsg        = &MsgTssPool{}
)

// NewMsgTssPool is a constructor function for MsgTssPool
func NewMsgTssPool(pks []string, poolpk common.PubKey, secp256k1Signature, keysharesBackup []byte, keygenType KeygenType, height int64, bl Blame, chains []string, signer cosmos.AccAddress, keygenTime int64) (*MsgTssPool, error) {
	id, err := getTssID(pks, poolpk, height, bl)
	if err != nil {
		return nil, fmt.Errorf("fail to get tss id: %w", err)
	}
	return &MsgTssPool{
		ID:                 id,
		PubKeys:            pks,
		PoolPubKey:         poolpk,
		Height:             height,
		KeygenType:         keygenType,
		Blame:              bl,
		Chains:             chains,
		Signer:             signer,
		KeygenTime:         keygenTime,
		KeysharesBackup:    keysharesBackup,
		Secp256K1Signature: secp256k1Signature,
	}, nil
}

// getTssID
func getTssID(members []string, poolPk common.PubKey, height int64, bl Blame) (string, error) {
	// ensure input pubkeys list is deterministically sorted
	sort.SliceStable(members, func(i, j int) bool {
		return members[i] < members[j]
	})

	pubkeys := make([]string, len(bl.BlameNodes))
	for i, node := range bl.BlameNodes {
		pubkeys[i] = node.Pubkey
	}
	sort.SliceStable(pubkeys, func(i, j int) bool {
		return pubkeys[i] < pubkeys[j]
	})

	sb := strings.Builder{}
	for _, item := range members {
		sb.WriteString("m:" + item)
	}
	for _, item := range pubkeys {
		sb.WriteString("p:" + item)
	}
	sb.WriteString(poolPk.String())
	sb.WriteString(fmt.Sprintf("%d", height))
	hash := sha256.New()
	_, err := hash.Write([]byte(sb.String()))
	if err != nil {
		return "", fmt.Errorf("fail to get tss id: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ValidateBasic implements HasValidateBasic
// ValidateBasic is now ran in the message service router handler for messages that
// used to be routed using the external handler and only when HasValidateBasic is implemented.
// No versioning is used there.
func (m *MsgTssPool) ValidateBasic() error {
	if m.Signer.Empty() {
		return cosmos.ErrInvalidAddress(m.Signer.String())
	}
	if len(m.ID) == 0 {
		return cosmos.ErrUnknownRequest("ID cannot be blank")
	}
	if len(m.PubKeys) < 2 {
		return cosmos.ErrUnknownRequest("Must have at least 2 pub keys")
	}
	if len(m.PubKeys) > 100 {
		return cosmos.ErrUnknownRequest("Must have no more then 100 pub keys")
	}
	pks := m.GetPubKeys()
	if len(m.PubKeys) != len(pks) {
		return cosmos.ErrUnknownRequest("One or more pubkeys were not valid")
	}
	isSignerInPubKeys := false
	for _, pk := range pks {
		if pk.IsEmpty() {
			return cosmos.ErrUnknownRequest("Pubkey cannot be empty")
		}
		signerAddress, err := pk.GetThorAddress()
		if err != nil {
			return cosmos.ErrUnknownRequest("invalid pub key")
		}
		if signerAddress.Equals(m.Signer) {
			isSignerInPubKeys = true
		}
	}
	if !isSignerInPubKeys {
		return cosmos.ErrUnknownRequest("signer is not part of keygen member")
	}
	// PoolPubKey can't be empty only when keygen success
	if m.IsSuccess() {
		if m.PoolPubKey.IsEmpty() {
			return cosmos.ErrUnknownRequest("Pool pubkey cannot be empty")
		}
	}
	// ensure pool pubkey is a valid bech32 pubkey
	if _, err := common.NewPubKey(m.PoolPubKey.String()); err != nil {
		return cosmos.ErrUnknownRequest(err.Error())
	}
	chains := m.GetChains()
	if len(chains) != len(m.Chains) {
		return cosmos.ErrUnknownRequest("One or more chains were not valid")
	}
	if !chains.Has(common.SwitchNative.Chain) {
		return cosmos.ErrUnknownRequest("must support SwitchNative asset chain")
	}
	if len(chains) != len(chains.Distinct()) {
		return cosmos.ErrUnknownRequest("cannot have duplicate chains")
	}

	if len(m.KeysharesBackup) != 0 {
		// sanity check encrypted keyshares do not a mnemonic
		if MatchMnemonic.Match(m.KeysharesBackup) {
			return cosmos.ErrUnknownRequest("invalid keyshares backup")
		}

		// sanity check encrypted keyshares are over 1Kb
		if len(m.KeysharesBackup) < 1024 {
			return cosmos.ErrUnknownRequest("invalid keyshares backup")
		}

		// sanity check probability distribution of keyshares backup bytes
		entropy := common.Entropy(m.KeysharesBackup)
		if entropy < MinKeysharesBackupEntropy {
			return cosmos.ErrUnknownRequest("invalid keyshares backup")
		}
	}

	return nil
}

// IsSuccess when blame is empty , then treat it as success
func (m MsgTssPool) IsSuccess() bool {
	return m.Blame.IsEmpty()
}

func (m MsgTssPool) GetChains() common.Chains {
	chains := make(common.Chains, 0)
	for _, c := range m.Chains {
		chain, err := common.NewChain(c)
		if err != nil {
			continue
		}
		chains = append(chains, chain)
	}
	return chains
}

func (m MsgTssPool) GetPubKeys() common.PubKeys {
	pubkeys := make(common.PubKeys, 0)
	for _, pk := range m.PubKeys {
		pk, err := common.NewPubKey(pk)
		if err != nil {
			continue
		}
		pubkeys = append(pubkeys, pk)
	}
	return pubkeys
}

// GetSigners defines whose signature is required
func (m *MsgTssPool) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{m.Signer}
}

func MsgTssPoolCustomGetSigners(m proto.Message) ([][]byte, error) {
	msg, ok := m.(*types.MsgTssPool)
	if !ok {
		return nil, fmt.Errorf("can't cast as MsgTssPool: %T", m)
	}
	return [][]byte{msg.Signer}, nil
}
