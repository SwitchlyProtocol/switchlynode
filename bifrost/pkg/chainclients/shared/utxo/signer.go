package utxo

import (
	"errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient"
	stypes "github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/tss"
)

// SignCheckpoint is used to checkpoint the built transaction before signing, for use in
// round 7 signing errors which must reuse the same inputs.
type SignCheckpoint struct {
	UnsignedTx        []byte           `json:"unsigned_tx"`
	IndividualAmounts map[string]int64 `json:"individual_amounts"`
}

func PostKeysignFailure(
	switchlyBridge switchlyclient.SwitchlyBridge,
	tx stypes.TxOutItem,
	logger zerolog.Logger,
	switchlyHeight int64,
	utxoErr error,
) error {
	// PostKeysignFailure only once per SignTx, to not broadcast duplicate messages.
	var keysignError tss.KeysignError
	if errors.As(utxoErr, &keysignError) {
		if len(keysignError.Blame.BlameNodes) == 0 {
			// TSS doesn't know which node to blame
			utxoErr = multierror.Append(utxoErr, fmt.Errorf("fail to sign UTXO"))
			return fmt.Errorf("fail to sign the message: %w", utxoErr)
		}

		// key sign error forward the keysign blame to switchly
		txID, err := switchlyBridge.PostKeysignFailure(keysignError.Blame, switchlyHeight, tx.Memo, tx.Coins, tx.VaultPubKey)
		if err != nil {
			logger.Error().Err(err).Msg("fail to post keysign failure to switchly")
			utxoErr = multierror.Append(utxoErr, fmt.Errorf("fail to post keysign failure to SWITCHLYChain: %w", err))
			return fmt.Errorf("fail to sign the message: %w", utxoErr)
		}
		logger.Info().Str("tx_id", txID.String()).Msgf("post keysign failure to switchly")
	}
	return utxoErr
}
