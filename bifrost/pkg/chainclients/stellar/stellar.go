package stellar

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"

	tssp "gitlab.com/thorchain/thornode/bifrost/tss/go-tss/tss"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stellar/go/keypair"
	"gitlab.com/thorchain/thornode/bifrost/blockscanner"
	"gitlab.com/thorchain/thornode/bifrost/metrics"
	"gitlab.com/thorchain/thornode/bifrost/pkg/chainclients/shared/signercache"
	"gitlab.com/thorchain/thornode/bifrost/thorclient"
	"gitlab.com/thorchain/thornode/bifrost/thorclient/types"
	"gitlab.com/thorchain/thornode/bifrost/tss"
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/config"
	"gitlab.com/thorchain/thornode/constants"
)

// Client represents a Stellar blockchain client
type Client struct {
	logger              zerolog.Logger
	cfg                 config.BifrostChainConfiguration
	client              *horizonclient.Client
	blockScanner        *StellarBlockScanner
	globalTxsQueue      chan types.TxIn
	globalErrataQueue   chan types.ErrataBlock
	globalSolvencyQueue chan types.Solvency
	storage             *blockscanner.BlockScannerStorage
	thorchainBridge     thorclient.ThorchainBridge
	wg                  *sync.WaitGroup
	stopChan            chan struct{}
	tssKeySigner        *tss.KeySign
	networkPassphrase   string
	signerCacheManager  *signercache.CacheManager
	sequenceMutex       sync.Mutex
	sequenceMap         map[string]uint64
	metrics             struct {
		signedTxs    prometheus.Counter
		broadcastTxs prometheus.Counter
		failedTxs    prometheus.Counter
	}
}

// NewStellarClient creates a new Stellar client
func NewStellarClient(thorKeys *thorclient.Keys, cfg config.BifrostChainConfiguration, server *tssp.TssServer, thorchainBridge thorclient.ThorchainBridge, m *metrics.Metrics) (*Client, error) {
	logger := zerolog.New(os.Stdout).With().Str("module", "stellar").Logger()
	var client *horizonclient.Client
	var networkPassphrase string

	switch cfg.ChainNetwork {
	case "testnet":
		client = horizonclient.DefaultTestNetClient
		networkPassphrase = network.TestNetworkPassphrase
	case "mainnet":
		client = horizonclient.DefaultPublicNetClient
		networkPassphrase = network.PublicNetworkPassphrase
	default:
		return nil, fmt.Errorf("unsupported network: %s", cfg.ChainNetwork)
	}

	if networkPassphrase == "" {
		return nil, fmt.Errorf("network passphrase cannot be empty")
	}

	tssKm, err := tss.NewKeySign(server, thorchainBridge)
	if err != nil {
		return nil, fmt.Errorf("failed to create tss signer: %w", err)
	}

	c := &Client{
		logger:            logger,
		cfg:               cfg,
		client:            client,
		thorchainBridge:   thorchainBridge,
		wg:                &sync.WaitGroup{},
		stopChan:          make(chan struct{}),
		tssKeySigner:      tssKm,
		networkPassphrase: networkPassphrase,
		sequenceMap:       make(map[string]uint64),
	}

	var path string
	if len(c.cfg.BlockScanner.DBPath) > 0 {
		path = fmt.Sprintf("%s/%s", c.cfg.BlockScanner.DBPath, c.cfg.BlockScanner.ChainID)
	}
	c.storage, err = blockscanner.NewBlockScannerStorage(path, c.cfg.ScannerLevelDB)
	if err != nil {
		return nil, fmt.Errorf("fail to create scan storage: %w", err)
	}

	signerCacheManager, err := signercache.NewSignerCacheManager(c.storage.GetInternalDb())
	if err != nil {
		return nil, fmt.Errorf("fail to create signer cache manager")
	}
	c.signerCacheManager = signerCacheManager

	blockScanner, err := NewStellarBlockScanner(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create block scanner: %w", err)
	}
	c.blockScanner = blockScanner

	return c, nil
}

// Start starts the Stellar chain client
func (c *Client) Start(globalTxsQueue chan types.TxIn, globalErrataQueue chan types.ErrataBlock, globalSolvencyQueue chan types.Solvency) {
	c.globalTxsQueue = globalTxsQueue
	c.globalErrataQueue = globalErrataQueue
	c.globalSolvencyQueue = globalSolvencyQueue
	c.tssKeySigner.Start()
	c.wg.Add(1)
	go c.blockScanner.Start(globalTxsQueue)
}

// Stop stops the Stellar chain client
func (c *Client) Stop() {
	c.tssKeySigner.Stop()
	c.blockScanner.Stop()
	close(c.stopChan)
	c.wg.Wait()
}

// GetConfig returns the chain configuration
func (c *Client) GetConfig() config.BifrostChainConfiguration {
	return c.cfg
}

// GetChain returns the chain id
func (c *Client) GetChain() common.Chain {
	return c.cfg.ChainID
}

// GetHeight returns the current block height
func (c *Client) GetHeight() (int64, error) {
	req := horizonclient.LedgerRequest{
		Order: horizonclient.OrderDesc,
		Limit: 1,
	}
	ledgers, err := c.client.Ledgers(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest ledger: %w", err)
	}
	if len(ledgers.Embedded.Records) == 0 {
		return 0, fmt.Errorf("no ledgers found")
	}
	return int64(ledgers.Embedded.Records[0].Sequence), nil
}

// GetAccountByAddress returns account details for an address
func (c *Client) GetAccountByAddress(address string, height *big.Int) (common.Account, error) {
	account, err := c.client.AccountDetail(horizonclient.AccountRequest{
		AccountID: address,
	})
	if err != nil {
		return common.Account{}, fmt.Errorf("failed to get account details: %w", err)
	}

	coins := make(common.Coins, 0)
	for _, balance := range account.Balances {
		if balance.Asset.Type == "native" {
			amount, err := strconv.ParseFloat(balance.Balance, 64)
			if err != nil {
				return common.Account{}, fmt.Errorf("failed to parse balance: %w", err)
			}
			coins = append(coins, common.NewCoin(stellarAsset, cosmos.NewUint(uint64(amount*common.One))))
		}
	}

	return common.Account{
		Sequence:      int64(account.Sequence),
		AccountNumber: 0, // Stellar doesn't have account numbers
		Coins:         coins,
	}, nil
}

// GetAccount returns account details
func (c *Client) GetAccount(pkey common.PubKey, height *big.Int) (common.Account, error) {
	addr, err := pkey.GetAddress(c.GetChain())
	if err != nil {
		return common.Account{}, fmt.Errorf("failed to get address from pubkey: %w", err)
	}
	return c.GetAccountByAddress(addr.String(), height)
}

// IsBlockScannerHealthy returns the health status of the block scanner
func (c *Client) IsBlockScannerHealthy() bool {
	return c.blockScanner.IsHealthy()
}

// SignTx signs a Stellar transaction
func (c *Client) SignTx(tx types.TxOutItem, thorchainHeight int64) (signedTx, checkpoint []byte, _ *types.TxInItem, err error) {
	defer func() {
		if err != nil {
			c.metrics.failedTxs.Inc()
			var keysignError tss.KeysignError
			if errors.As(err, &keysignError) {
				if len(keysignError.Blame.BlameNodes) == 0 {
					c.logger.Err(err).Msg("TSS doesn't know which node to blame")
					return
				}
				var txID common.TxID
				txID, err = c.thorchainBridge.PostKeysignFailure(keysignError.Blame, thorchainHeight, tx.Memo, tx.Coins, tx.VaultPubKey)
				if err != nil {
					c.logger.Err(err).Msg("fail to post keysign failure to THORChain")
					return
				}
				c.logger.Info().Str("tx_id", txID.String()).Msgf("post keysign failure to thorchain")
			}
			c.logger.Err(err).Msg("failed to sign tx")
			return
		}
		c.metrics.signedTxs.Inc()
	}()

	// Validate transaction
	if err := c.validateTx(tx); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid transaction: %w", err)
	}

	// Get source account with retry
	horizonAccount, err := c.client.AccountDetail(horizonclient.AccountRequest{
		AccountID: tx.VaultPubKey.String(),
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get source account: %w", err)
	}

	sourceAccount := txnbuild.SimpleAccount{
		AccountID: horizonAccount.AccountID,
		Sequence:  horizonAccount.Sequence,
	}

	c.sequenceMutex.Lock()
	if seq, ok := c.sequenceMap[tx.VaultPubKey.String()]; ok {
		sourceAccount.Sequence = int64(seq)
	}
	c.sequenceMutex.Unlock()

	if c.signerCacheManager.HasSigned(tx.CacheHash()) {
		c.logger.Info().Interface("tx", tx).Msg("transaction already signed, ignoring...")
		return nil, nil, nil, nil
	}

	amount := strconv.FormatFloat(float64(tx.Coins[0].Amount.Uint64())/float64(common.One), 'f', 7, 64)
	payment := &txnbuild.Payment{
		SourceAccount: sourceAccount.GetAccountID(),
		Destination:   tx.ToAddress.String(),
		Amount:        amount,
		Asset:         txnbuild.NativeAsset{},
	}

	params := txnbuild.TransactionParams{
		SourceAccount:        &sourceAccount,
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{payment},
		BaseFee:              txnbuild.MinBaseFee,
		Memo:                 txnbuild.MemoText(tx.Memo),
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
	}

	tx1, err := txnbuild.NewTransaction(params)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	msgToSign, err := network.HashTransactionInEnvelope(tx1.ToXDR(), c.networkPassphrase)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to hash transaction: %w", err)
	}

	signedData, _, err := c.tssKeySigner.RemoteSign(msgToSign[:], tx.VaultPubKey.String())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to sign with TSS: %w", err)
	}

	// Verify the signature
	kp, err := keypair.ParseAddress(tx.VaultPubKey.String())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse vault pubkey: %w", err)
	}
	err = kp.Verify(msgToSign[:], signedData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("signature verification failed: %w", err)
	}

	updatedTx, err := tx1.AddSignatureBase64(c.networkPassphrase, tx.VaultPubKey.String(), base64.StdEncoding.EncodeToString(signedData))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to add signature: %w", err)
	}

	txeBase64, err := updatedTx.Base64()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to convert to XDR: %w", err)
	}

	c.sequenceMutex.Lock()
	c.sequenceMap[tx.VaultPubKey.String()] = uint64(sourceAccount.Sequence + 1)
	c.sequenceMutex.Unlock()

	return []byte(txeBase64), nil, nil, nil
}

func (c *Client) validateTx(tx types.TxOutItem) error {
	if tx.Chain != common.STELLARChain {
		return fmt.Errorf("chain %s not supported", tx.Chain)
	}
	if len(tx.Memo) == 0 || len(tx.Memo) > maxMemoLength {
		return fmt.Errorf("invalid memo length: %d", len(tx.Memo))
	}
	if len(tx.Coins) != 1 {
		return fmt.Errorf("stellar transactions must have exactly one coin")
	}
	if tx.Coins[0].Asset != common.XLMAsset {
		return fmt.Errorf("invalid asset: %s", tx.Coins[0].Asset)
	}
	// Check minimum balance requirements (0.5 XLM + fees)
	minBalance := uint64(minTxValue + (maxGasAmount * txnbuild.MinBaseFee))
	if tx.Coins[0].Amount.IsZero() || tx.Coins[0].Amount.Uint64() < minBalance {
		return fmt.Errorf("amount %d below minimum required %d", tx.Coins[0].Amount.Uint64(), minBalance)
	}
	return nil
}

// BroadcastTx broadcasts a signed transaction
func (c *Client) BroadcastTx(tx types.TxOutItem, signedTx []byte) (string, error) {
	if len(signedTx) == 0 {
		return "", fmt.Errorf("empty signed transaction")
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := c.client.SubmitTransactionXDR(string(signedTx))
		if err == nil {
			c.metrics.broadcastTxs.Inc()
			if err := c.signerCacheManager.SetSigned(tx.CacheHash(), tx.CacheVault(c.GetChain()), resp.Hash); err != nil {
				c.logger.Err(err).Msg("fail to update signer cache")
			}
			return resp.Hash, nil
		}

		var hError *horizonclient.Error
		if errors.As(err, &hError) {
			resultCodes := fmt.Sprintf("%v", hError.Problem.Extras["result_codes"])
			c.logger.Error().
				Str("result_codes", resultCodes).
				Str("tx_hash", hError.Problem.Extras["hash"].(string)).
				Msg("stellar transaction failed")

			// Don't retry permanent failures
			if strings.Contains(resultCodes, "tx_bad_auth") ||
				strings.Contains(resultCodes, "tx_bad_seq") {
				return "", fmt.Errorf("permanent failure: %w", err)
			}
		}

		lastErr = err
		time.Sleep(time.Second * time.Duration(i+1))
	}

	return "", fmt.Errorf("failed to broadcast after %d attempts: %w", maxRetries, lastErr)
}

// GetConfirmationCount returns the number of blocks needed for confirmation
func (c *Client) GetConfirmationCount(txIn types.TxIn) int64 {
	return 1 // Stellar has instant finality
}

// ConfirmationCountReady checks if a transaction has enough confirmations
func (c *Client) ConfirmationCountReady(txIn types.TxIn) bool {
	return true // Stellar has instant finality
}

// OnObservedTxIn gets called when a transaction is observed
func (c *Client) OnObservedTxIn(txIn types.TxInItem, blockHeight int64) {
	// Nothing to do for Stellar
}

// GetAddress returns the address of the Stellar client
func (c *Client) GetAddress(poolPubKey common.PubKey) string {
	addr, err := poolPubKey.GetAddress(common.STELLARChain)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to get address from pubkey")
		return ""
	}
	return addr.String()
}

// GetBlockScannerHeight returns the current block scanner height
func (c *Client) GetBlockScannerHeight() (int64, error) {
	return c.blockScanner.GetHeight()
}

// GetLatestTxForVault returns the latest transaction for a vault
func (c *Client) GetLatestTxForVault(vaultPubKey string) (string, string, error) {
	return "", "", fmt.Errorf("not implemented") // TODO: Implement this
}

// ReportSolvency reports solvency information
func (c *Client) ReportSolvency(height int64) error {
	if !c.ShouldReportSolvency(height) {
		return nil
	}

	asgardVaults, err := c.thorchainBridge.GetAsgards()
	if err != nil {
		return fmt.Errorf("fail to get asgards: %w", err)
	}

	for _, vault := range asgardVaults {
		acct, err := c.GetAccount(vault.PubKey, nil)
		if err != nil {
			c.logger.Err(err).Msg("fail to get account balance")
			continue
		}

		select {
		case c.globalSolvencyQueue <- types.Solvency{
			Chain:  common.STELLARChain,
			PubKey: vault.PubKey,
			Coins:  acct.Coins,
			Height: height,
		}:
		case <-time.After(constants.ThorchainBlockTime):
			c.logger.Info().Msg("fail to send solvency info to THORChain, timeout")
		}
	}
	return nil
}

// ShouldReportSolvency checks if solvency should be reported
func (c *Client) ShouldReportSolvency(height int64) bool {
	return height%20 == 0
}

// IsHealthy returns the health status of the Stellar client
func (c *Client) IsHealthy() bool {
	_, err := c.GetHeight()
	if err != nil {
		c.logger.Error().Err(err).Msg("stellar client not healthy")
		return false
	}
	return c.IsBlockScannerHealthy()
}
