package stellar

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/config"
)

// RouterEventScanner scans for Stellar smart contract events from the Switchly router
type RouterEventScanner struct {
	cfg              config.BifrostBlockScannerConfiguration
	logger           zerolog.Logger
	horizonClient    *horizonclient.Client
	sorobanRPCClient *SorobanRPCClient
	routerAddress    string
	retryConfig      RetryConfig
	bridge           switchlyclient.SwitchlyBridge
}

// RetryConfig for handling rate limits
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

// StellarEvent represents a parsed Stellar contract event
type StellarEvent struct {
	Type      string
	Topics    []string
	Data      interface{}
	TxHash    string
	Ledger    int64
	Operation string
}

// RouterDepositEvent represents a deposit event from the router
type RouterDepositEvent struct {
	Vault  string `json:"vault"`
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
	Memo   string `json:"memo"`
}

// RouterTransferOutEvent represents a transfer out event from the router
type RouterTransferOutEvent struct {
	Vault  string `json:"vault"`
	To     string `json:"to"`
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
	Memo   string `json:"memo"`
}

// NewRouterEventScanner creates a new router event scanner
func NewRouterEventScanner(
	cfg config.BifrostBlockScannerConfiguration,
	horizonClient *horizonclient.Client,
	sorobanRPCClient *SorobanRPCClient,
	routerAddress string,
	bridge switchlyclient.SwitchlyBridge,
) *RouterEventScanner {
	logger := log.Logger.With().Str("module", "router-event-scanner").Str("chain", cfg.ChainID.String()).Logger()

	return &RouterEventScanner{
		cfg:              cfg,
		logger:           logger,
		horizonClient:    horizonClient,
		sorobanRPCClient: sorobanRPCClient,
		routerAddress:    routerAddress,
		bridge:           bridge,
		retryConfig: RetryConfig{
			MaxRetries: cfg.MaxHTTPRequestRetry,
			BaseDelay:  cfg.BlockHeightDiscoverBackoff,
			MaxDelay:   30 * time.Second,
			Multiplier: 2.0,
		},
	}
}

// ScanRouterEvents scans for router contract events in a specific ledger
func (r *RouterEventScanner) ScanRouterEvents(height int64) ([]*types.TxInItem, error) {
	var txInItems []*types.TxInItem

	// Use Soroban RPC to get contract events if available
	if r.sorobanRPCClient != nil {
		ctx := context.Background()
		routerEvents, err := r.sorobanRPCClient.GetRouterEvents(ctx, uint32(height), []string{r.routerAddress})
		if err != nil {
			// Check if the error is due to the height being too old
			if strings.Contains(err.Error(), "startLedger must be within the ledger range") {
				r.logger.Debug().
					Int64("height", height).
					Msg("height is outside available ledger range, skipping router event scan")
				return txInItems, nil // Return empty results for old heights
			}

			r.logger.Error().Err(err).Int64("height", height).Msg("failed to get router events from Soroban RPC")
			// Fall back to Horizon API
			return r.scanRouterEventsFromHorizon(height)
		}

		// Process Soroban contract events
		for _, event := range routerEvents {
			txInItem, err := r.processRouterEventFromSoroban(event, height)
			if err != nil {
				r.logger.Error().Err(err).Str("tx_hash", event.TransactionHash).Msg("failed to process router event from Soroban")
				continue
			}
			if txInItem != nil {
				txInItems = append(txInItems, txInItem)
			}
		}

		r.logger.Debug().
			Int64("height", height).
			Int("event_count", len(routerEvents)).
			Int("processed_count", len(txInItems)).
			Msg("processed router events from Soroban RPC")

		return txInItems, nil
	}

	// Fall back to Horizon API if Soroban RPC is not available
	return r.scanRouterEventsFromHorizon(height)
}

// scanRouterEventsFromHorizon scans for router events using Horizon API (fallback)
func (r *RouterEventScanner) scanRouterEventsFromHorizon(height int64) ([]*types.TxInItem, error) {
	var txInItems []*types.TxInItem

	// Get all transactions for this ledger
	txs, err := r.getTransactionsForLedger(height)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions for ledger %d: %w", height, err)
	}

	// Process each transaction for router events
	for _, tx := range txs {
		if !tx.Successful {
			continue
		}

		// Get operations for this transaction
		ops, err := r.getOperationsForTransaction(tx.Hash)
		if err != nil {
			r.logger.Error().Err(err).Str("tx_hash", tx.Hash).Msg("failed to get operations")
			continue
		}

		// Process each operation for router events
		for _, op := range ops {
			txInItem, err := r.processRouterOperation(tx, op, height)
			if err != nil {
				r.logger.Error().Err(err).Str("tx_hash", tx.Hash).Msg("failed to process router operation")
				continue
			}
			if txInItem != nil {
				txInItems = append(txInItems, txInItem)
			}
		}
	}

	return txInItems, nil
}

// getTransactionsForLedger retrieves all transactions for a specific ledger
func (r *RouterEventScanner) getTransactionsForLedger(height int64) ([]horizon.Transaction, error) {
	var allTxs []horizon.Transaction

	err := r.retryCall("transactions", func() error {
		txRequest := horizonclient.TransactionRequest{
			ForLedger: uint(height),
			Limit:     200,
		}

		txPage, err := r.horizonClient.Transactions(txRequest)
		if err != nil {
			return err
		}

		allTxs = append(allTxs, txPage.Embedded.Records...)

		// Handle pagination
		for len(txPage.Embedded.Records) == 200 {
			txPage, err = r.horizonClient.NextTransactionsPage(txPage)
			if err != nil {
				break
			}
			allTxs = append(allTxs, txPage.Embedded.Records...)
		}

		return nil
	})

	return allTxs, err
}

// getOperationsForTransaction retrieves all operations for a specific transaction
func (r *RouterEventScanner) getOperationsForTransaction(txHash string) ([]operations.Operation, error) {
	var ops []operations.Operation

	err := r.retryCall("operations", func() error {
		operationsPage, err := r.horizonClient.Operations(horizonclient.OperationRequest{
			ForTransaction: txHash,
		})
		if err != nil {
			return err
		}

		ops = operationsPage.Embedded.Records
		return nil
	})

	return ops, err
}

// processRouterOperation processes a single operation for router events
func (r *RouterEventScanner) processRouterOperation(tx horizon.Transaction, op operations.Operation, height int64) (*types.TxInItem, error) {
	// Check if this is an invoke contract operation
	invokeOp, ok := op.(operations.InvokeHostFunction)
	if !ok {
		return nil, nil // Not a contract invocation
	}

	// Check if this operation involves our router contract
	if !r.isRouterOperation(invokeOp) {
		return nil, nil // Not a router operation
	}

	// Get contract events for this operation
	events, err := r.getContractEvents(tx.Hash, invokeOp)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract events: %w", err)
	}

	// Process events to extract router-specific events
	for _, event := range events {
		txInItem, err := r.processRouterEvent(tx, event, height)
		if err != nil {
			r.logger.Error().Err(err).Str("tx_hash", tx.Hash).Msg("failed to process router event")
			continue
		}
		if txInItem != nil {
			return txInItem, nil // Return first valid event
		}
	}

	return nil, nil
}

// isRouterOperation checks if an operation involves our router contract
func (r *RouterEventScanner) isRouterOperation(op operations.InvokeHostFunction) bool {
	// Check if the contract address matches our router
	// This would need to be implemented based on how Stellar exposes contract addresses in operations
	// For now, we'll assume all invoke operations could be router operations
	return true
}

// getContractEvents retrieves contract events for a specific transaction/operation
func (r *RouterEventScanner) getContractEvents(txHash string, op operations.InvokeHostFunction) ([]StellarEvent, error) {
	var events []StellarEvent

	// For now, we'll create a placeholder implementation
	// In a real implementation, this would parse the contract events from the operation
	// or from the transaction effects

	// This is a simplified approach - in practice, we'd need to:
	// 1. Parse the operation's XDR data to extract contract events
	// 2. Or use Stellar RPC to get contract events
	// 3. Or parse from transaction effects if available

	r.logger.Debug().
		Str("tx_hash", txHash).
		Msg("contract event parsing not fully implemented - placeholder")

	return events, nil
}

// processRouterEvent processes a router-specific event into a TxInItem
func (r *RouterEventScanner) processRouterEvent(tx horizon.Transaction, event StellarEvent, height int64) (*types.TxInItem, error) {
	switch event.Type {
	case "deposit":
		return r.processDepositEvent(tx, event, height)
	case "xfer_out":
		return r.processTransferOutEvent(tx, event, height)
	default:
		return nil, nil // Unknown event type
	}
}

// processDepositEvent processes a deposit event from the router
func (r *RouterEventScanner) processDepositEvent(tx horizon.Transaction, event StellarEvent, height int64) (*types.TxInItem, error) {
	// Parse deposit event data
	var depositEvent RouterDepositEvent
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", event.Data)), &depositEvent); err != nil {
		return nil, fmt.Errorf("failed to parse deposit event: %w", err)
	}

	// Find the asset mapping
	mapping, found := r.findAssetMappingByAddress(depositEvent.Asset)
	if !found {
		return nil, fmt.Errorf("unsupported asset: %s", depositEvent.Asset)
	}

	// Convert amount
	coin, err := mapping.ConvertToSwitchlyProtocolAmount(depositEvent.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to convert amount: %w", err)
	}

	// Create addresses - use the account field from transaction
	fromAddr, err := common.NewAddress(tx.Account)
	if err != nil {
		return nil, fmt.Errorf("failed to parse from address: %w", err)
	}

	toAddr, err := common.NewAddress(depositEvent.Vault)
	if err != nil {
		return nil, fmt.Errorf("failed to parse to address: %w", err)
	}

	// Get the vault public key for XLM chain
	vaultPubKey, err := r.getVaultPubKeyForXLM()
	if err != nil {
		r.logger.Warn().Err(err).
			Str("tx_hash", tx.Hash).
			Msg("failed to get vault public key for XLM chain, skipping transaction")
		return nil, nil
	}

	// Verify that we have a valid vault pub key
	if vaultPubKey.IsEmpty() {
		r.logger.Warn().
			Str("tx_hash", tx.Hash).
			Msg("received empty vault pub key for XLM chain, skipping transaction")
		return nil, nil
	}

	// Verify we can get a valid XLM address from this vault pub key
	xlmAddress, err := vaultPubKey.GetAddress(common.StellarChain)
	if err != nil {
		r.logger.Warn().Err(err).
			Str("tx_hash", tx.Hash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("failed to get XLM address from vault pub key, skipping transaction")
		return nil, nil
	}

	if xlmAddress.IsEmpty() {
		r.logger.Warn().
			Str("tx_hash", tx.Hash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("vault pub key returned empty XLM address, skipping transaction")
		return nil, nil
	}

	r.logger.Debug().
		Str("tx_hash", tx.Hash).
		Str("vault_pubkey", vaultPubKey.String()).
		Str("xlm_address", xlmAddress.String()).
		Msg("processing deposit event with valid vault pub key")

	// Create TxInItem
	txInItem := &types.TxInItem{
		BlockHeight:         height,
		Tx:                  tx.Hash,
		Sender:              fromAddr.String(),
		To:                  toAddr.String(),
		Coins:               common.Coins{coin},
		Memo:                depositEvent.Memo,
		ObservedVaultPubKey: vaultPubKey,
		Gas:                 common.Gas{coin}, // Use same coin for gas
	}

	return txInItem, nil
}

// processTransferOutEvent processes a transfer out event from the router
func (r *RouterEventScanner) processTransferOutEvent(tx horizon.Transaction, event StellarEvent, height int64) (*types.TxInItem, error) {
	// Transfer out events are typically for tracking outbound transactions
	// They might not generate TxInItems but could be used for monitoring
	r.logger.Info().
		Str("tx_hash", tx.Hash).
		Int64("height", height).
		Msg("router transfer out event detected")

	return nil, nil
}

// findAssetMappingByAddress finds asset mapping by Stellar contract address
func (r *RouterEventScanner) findAssetMappingByAddress(address string) (StellarAssetMapping, bool) {
	// Use the centralized network-aware asset mapping function
	mapping, found := GetAssetByAddress(address)
	if found {
		r.logger.Debug().
			Str("address", address).
			Str("network", string(GetCurrentNetwork())).
			Str("asset_code", mapping.StellarAssetCode).
			Str("asset_type", mapping.StellarAssetType).
			Msg("successfully resolved asset mapping")
		return mapping, true
	}

	// Log detailed debugging information
	r.logger.Debug().
		Str("address", address).
		Str("current_network", string(GetCurrentNetwork())).
		Msg("no asset mapping found for address on current network")

	// Try to find the asset on other networks for debugging
	for _, network := range []StellarNetwork{StellarMainnet, StellarTestnet} {
		if network == GetCurrentNetwork() {
			continue // Skip current network as we already tried it
		}

		if otherMapping, otherFound := GetAssetByAddressAndNetwork(address, network); otherFound {
			r.logger.Warn().
				Str("address", address).
				Str("found_on_network", string(network)).
				Str("current_network", string(GetCurrentNetwork())).
				Str("asset_code", otherMapping.StellarAssetCode).
				Str("asset_type", otherMapping.StellarAssetType).
				Msg("asset found on different network - network configuration mismatch")
			break
		}
	}

	return StellarAssetMapping{}, false
}

// retryCall executes a function with retry logic for rate limits
func (r *RouterEventScanner) retryCall(operation string, fn func() error) error {
	for attempt := 0; attempt <= r.retryConfig.MaxRetries; attempt++ {
		err := fn()
		if err != nil {
			// Check if it's a rate limit error
			if strings.Contains(err.Error(), "Rate Limit Exceeded") ||
				strings.Contains(err.Error(), "429") {
				if attempt < r.retryConfig.MaxRetries {
					// Exponential backoff for rate limits
					delay := time.Duration(float64(r.retryConfig.BaseDelay) *
						math.Pow(r.retryConfig.Multiplier, float64(attempt)))
					if delay > r.retryConfig.MaxDelay {
						delay = r.retryConfig.MaxDelay
					}

					r.logger.Warn().
						Str("operation", operation).
						Int("attempt", attempt+1).
						Int("max_retries", r.retryConfig.MaxRetries+1).
						Dur("delay", delay).
						Msg("rate limit hit, retrying after delay")
					time.Sleep(delay)
					continue
				}
			}
			return err
		}
		return nil
	}

	return fmt.Errorf("max retries exceeded for operation: %s", operation)
}

// processRouterEventFromSoroban processes a router event from Soroban RPC
func (r *RouterEventScanner) processRouterEventFromSoroban(event *RouterEvent, height int64) (*types.TxInItem, error) {
	// Determine event type based on the event signature
	eventType := strings.ToLower(event.Type)

	// Log the event for debugging
	r.logger.Debug().
		Str("event_type", eventType).
		Str("tx_hash", event.TransactionHash).
		Str("asset", event.Asset).
		Str("amount", event.Amount).
		Str("from", event.FromAddress).
		Str("to", event.ToAddress).
		Str("memo", event.Memo).
		Msg("processing router event from Soroban")

	switch eventType {
	case "deposit", "router_deposit":
		return r.processDepositEventFromSoroban(event, height)
	case "transfer_out", "router_transfer_out", "transferout":
		return r.processTransferOutEventFromSoroban(event, height)
	case "deposit_with_expiry", "depositwithexpiry":
		// Handle deposit with expiry as a regular deposit for now
		return r.processDepositEventFromSoroban(event, height)
	case "transfer_allowance", "transferallowance":
		// Handle vault rotation events
		return r.processTransferAllowanceEventFromSoroban(event, height)
	case "return_vault_assets", "returnvaultassets":
		// Handle vault asset return events
		return r.processReturnVaultAssetsEventFromSoroban(event, height)
	default:
		r.logger.Debug().
			Str("event_type", eventType).
			Str("tx_hash", event.TransactionHash).
			Msg("unknown router event type - skipping")
		return nil, nil
	}
}

// processDepositEventFromSoroban processes a deposit event from Soroban RPC
func (r *RouterEventScanner) processDepositEventFromSoroban(event *RouterEvent, height int64) (*types.TxInItem, error) {
	// Find the asset mapping
	assetAddress := event.Asset
	if assetAddress == "" {
		return nil, fmt.Errorf("missing asset address in deposit event")
	}

	mapping, found := GetAssetByAddress(assetAddress)
	if !found {
		r.logger.Warn().
			Str("asset_address", assetAddress).
			Str("current_network", string(GetCurrentNetwork())).
			Msg("unsupported asset - adding dynamic mapping")

		// Try to create a dynamic mapping for unknown assets
		if len(assetAddress) == 56 && strings.HasPrefix(assetAddress, "C") {
			// This looks like a Soroban contract address
			mapping = StellarAssetMapping{
				StellarAssetType:   "contract",
				StellarAssetCode:   "UNKNOWN",
				StellarAssetIssuer: assetAddress,
				StellarDecimals:    7, // Default to 7 decimals
				SwitchlyAsset:      common.Asset{Chain: common.StellarChain, Symbol: "UNKNOWN", Ticker: "UNKNOWN"},
			}
			r.logger.Info().
				Str("asset_address", assetAddress).
				Msg("created dynamic asset mapping for unknown contract")
		} else {
			return nil, fmt.Errorf("unsupported asset: %s", assetAddress)
		}
	}

	// Convert amount
	coin, err := mapping.ConvertToSwitchlyProtocolAmount(event.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to convert amount: %w", err)
	}

	// Create addresses
	fromAddr, err := common.NewAddress(event.FromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse from address: %w", err)
	}

	// For deposits, the vault address is the destination
	toAddr, err := common.NewAddress(event.ToAddress)
	if err != nil {
		// Use destination if ToAddress is not available
		if event.Destination != "" {
			toAddr, err = common.NewAddress(event.Destination)
			if err != nil {
				return nil, fmt.Errorf("failed to parse destination address: %w", err)
			}
		} else {
			return nil, fmt.Errorf("missing destination address in deposit event")
		}
	}

	// Get the vault public key for XLM chain
	vaultPubKey, err := r.getVaultPubKeyForXLM()
	if err != nil {
		r.logger.Warn().Err(err).
			Str("tx_hash", event.TransactionHash).
			Msg("failed to get vault public key for XLM chain, skipping transaction")
		return nil, nil
	}

	// Verify that we have a valid vault pub key
	if vaultPubKey.IsEmpty() {
		r.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Msg("received empty vault pub key for XLM chain, skipping transaction")
		return nil, nil
	}

	// Verify we can get a valid XLM address from this vault pub key
	xlmAddress, err := vaultPubKey.GetAddress(common.StellarChain)
	if err != nil {
		r.logger.Warn().Err(err).
			Str("tx_hash", event.TransactionHash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("failed to get XLM address from vault pub key, skipping transaction")
		return nil, nil
	}

	if xlmAddress.IsEmpty() {
		r.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("vault pub key returned empty XLM address, skipping transaction")
		return nil, nil
	}

	r.logger.Debug().
		Str("tx_hash", event.TransactionHash).
		Str("vault_pubkey", vaultPubKey.String()).
		Str("xlm_address", xlmAddress.String()).
		Msg("processing deposit event from Soroban RPC with valid vault pub key")

	// Create TxInItem
	txInItem := &types.TxInItem{
		BlockHeight:         height,
		Tx:                  event.TransactionHash,
		Sender:              fromAddr.String(),
		To:                  toAddr.String(),
		Coins:               common.Coins{coin},
		Memo:                event.Memo,
		ObservedVaultPubKey: vaultPubKey,
		Gas:                 common.Gas{coin}, // Use same coin for gas
	}

	r.logger.Info().
		Str("tx_hash", event.TransactionHash).
		Str("from", fromAddr.String()).
		Str("to", toAddr.String()).
		Str("asset", mapping.SwitchlyAsset.String()).
		Str("amount", coin.Amount.String()).
		Str("memo", event.Memo).
		Msg("processed deposit event from Soroban RPC")

	return txInItem, nil
}

// processTransferOutEventFromSoroban processes a transfer out event from Soroban RPC
func (r *RouterEventScanner) processTransferOutEventFromSoroban(event *RouterEvent, height int64) (*types.TxInItem, error) {
	// Transfer out events are typically for tracking outbound transactions
	// They might not generate TxInItems but could be used for monitoring
	r.logger.Info().
		Str("tx_hash", event.TransactionHash).
		Int64("height", height).
		Str("asset", event.Asset).
		Str("amount", event.Amount).
		Str("destination", event.Destination).
		Msg("router transfer out event detected from Soroban RPC")

	return nil, nil
}

// processTransferAllowanceEventFromSoroban processes a transfer allowance event (vault rotation)
func (r *RouterEventScanner) processTransferAllowanceEventFromSoroban(event *RouterEvent, height int64) (*types.TxInItem, error) {
	r.logger.Info().
		Str("tx_hash", event.TransactionHash).
		Int64("height", height).
		Str("asset", event.Asset).
		Str("amount", event.Amount).
		Str("from", event.FromAddress).
		Str("to", event.ToAddress).
		Msg("router transfer allowance event detected from Soroban RPC")

	return nil, nil
}

// processReturnVaultAssetsEventFromSoroban processes a return vault assets event
func (r *RouterEventScanner) processReturnVaultAssetsEventFromSoroban(event *RouterEvent, height int64) (*types.TxInItem, error) {
	r.logger.Info().
		Str("tx_hash", event.TransactionHash).
		Int64("height", height).
		Str("asset", event.Asset).
		Str("amount", event.Amount).
		Msg("router return vault assets event detected from Soroban RPC")

	return nil, nil
}

// getVaultPubKeyForXLM retrieves the vault public key for XLM chain from the bridge
func (r *RouterEventScanner) getVaultPubKeyForXLM() (common.PubKey, error) {
	if r.bridge == nil {
		return common.EmptyPubKey, fmt.Errorf("bridge not available")
	}

	// Get vault public keys from the bridge
	vaultPubKeyPairs, err := r.bridge.GetAsgardPubKeys()
	if err != nil {
		return common.EmptyPubKey, fmt.Errorf("failed to get vault public keys: %w", err)
	}

	// Find a vault that has a contract for XLM chain
	for _, vaultPair := range vaultPubKeyPairs {
		if vaultPair.PubKey.IsEmpty() {
			continue
		}

		// Check if this vault has a contract for XLM chain
		if contractAddr, hasContract := vaultPair.Contracts[common.StellarChain]; hasContract {
			r.logger.Debug().
				Str("vault_pubkey", vaultPair.PubKey.String()).
				Str("xlm_contract", contractAddr.String()).
				Msg("found vault with XLM chain contract")
			return vaultPair.PubKey, nil
		}
	}

	return common.EmptyPubKey, fmt.Errorf("no vault found with XLM chain contract")
}
