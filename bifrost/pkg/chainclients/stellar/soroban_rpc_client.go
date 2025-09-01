package stellar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"

	"github.com/stellar/go/xdr"
	"github.com/switchlyprotocol/switchlynode/v3/config"
)

// SorobanRPCClient handles Soroban RPC calls for contract events
//
// Configuration:
// - Uses cfg.RPCHost from environment variables (e.g., BIFROST_CHAINS_XLM_RPC_HOST)
// - For Docker/local environments: "http://stellar:8000" -> "http://stellar:8000/soroban/rpc"
// - For public networks: Falls back to "https://soroban-testnet.stellar.org" or "https://soroban-mainnet.stellar.org"
// - Stellar quickstart container with --enable-soroban-rpc exposes both Horizon and Soroban RPC on port 8000
type SorobanRPCClient struct {
	rpcURL      string
	httpClient  *retryablehttp.Client
	logger      zerolog.Logger
	networkType StellarNetwork
}

// SorobanRPCRequest represents a JSON-RPC request to Soroban
type SorobanRPCRequest struct {
	JSONRpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// SorobanRPCResponse represents a JSON-RPC response from Soroban
type SorobanRPCResponse struct {
	JSONRpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents an RPC error response
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// GetEventsRequest represents parameters for getEvents RPC call
type GetEventsRequest struct {
	StartLedger uint32                `json:"startLedger"`
	EndLedger   *uint32               `json:"endLedger,omitempty"`
	Filters     []ContractEventFilter `json:"filters,omitempty"`
	Pagination  *EventPagination      `json:"pagination,omitempty"`
}

// ContractEventFilter filters contract events by type and contract ID
type ContractEventFilter struct {
	Type        string   `json:"type"`
	ContractIDs []string `json:"contractIds,omitempty"`
	Topics      []string `json:"topics,omitempty"`
}

// EventPagination handles pagination for event queries
type EventPagination struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  uint32 `json:"limit,omitempty"`
}

// GetEventsResponse represents the response from getEvents RPC call
type GetEventsResponse struct {
	Events           []ContractEvent `json:"events"`
	LatestLedger     uint32          `json:"latestLedger"`
	LatestLedgerTime string          `json:"latestLedgerCloseTime"`
}

// ContractEvent represents a Soroban contract event
type ContractEvent struct {
	Type                     string   `json:"type"`
	Ledger                   uint32   `json:"ledger"`
	LedgerTime               string   `json:"ledgerCloseTime"`
	ContractID               string   `json:"contractId"`
	ID                       string   `json:"id"`
	PagingToken              string   `json:"pagingToken"`
	Topic                    []string `json:"topic"`
	Value                    string   `json:"value"`
	InSuccessfulContractCall bool     `json:"inSuccessfulContractCall"`
	TransactionHash          string   `json:"txHash"`
}

// RouterEvent represents a parsed router event
type RouterEvent struct {
	Type            string
	ContractAddress string
	TransactionHash string
	Ledger          uint32
	LedgerTime      time.Time

	// Router-specific fields
	Asset       string
	Amount      string
	Destination string
	Memo        string
	FromAddress string
	ToAddress   string
}

// NewSorobanRPCClient creates a new Soroban RPC client
func NewSorobanRPCClient(cfg config.BifrostChainConfiguration, logger zerolog.Logger, networkType StellarNetwork) *SorobanRPCClient {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 10 * time.Second
	retryClient.Logger = nil // Disable retryablehttp logging

	// Use the configured RPC host from environment
	rpcURL := cfg.RPCHost
	if rpcURL != "" {
		// For local/Docker environments, Soroban RPC is typically at /soroban/rpc path
		// Check if the URL already includes the Soroban RPC path
		if !strings.Contains(rpcURL, "/soroban/rpc") {
			// Append the Soroban RPC path for local Stellar quickstart containers
			rpcURL = strings.TrimSuffix(rpcURL, "/") + "/soroban/rpc"
		}
		logger.Info().
			Str("base_rpc_host", cfg.RPCHost).
			Str("soroban_rpc_url", rpcURL).
			Msg("using configured RPC host for Soroban RPC")
	} else {
		// Only fall back to public networks if no RPC host is configured
		logger.Warn().Msg("no RPC host configured, falling back to public Stellar networks")
		switch networkType {
		case StellarTestnet:
			rpcURL = "https://soroban-testnet.stellar.org"
		case StellarMainnet:
			rpcURL = "https://soroban-mainnet.stellar.org"
		default:
			rpcURL = "https://soroban-testnet.stellar.org"
		}
	}

	logger.Info().
		Str("rpc_url", rpcURL).
		Str("network_type", string(networkType)).
		Msg("initialized Soroban RPC client")

	return &SorobanRPCClient{
		rpcURL:      rpcURL,
		httpClient:  retryClient,
		logger:      logger.With().Str("module", "soroban_rpc").Logger(),
		networkType: networkType,
	}
}

// GetContractEvents retrieves contract events from Soroban RPC
func (s *SorobanRPCClient) GetContractEvents(ctx context.Context, startLedger uint32, contractIDs []string, eventTypes []string) (*GetEventsResponse, error) {
	s.logger.Debug().
		Uint32("start_ledger", startLedger).
		Strs("contract_ids", contractIDs).
		Strs("event_types", eventTypes).
		Msg("Getting contract events from Soroban RPC")

	filters := make([]ContractEventFilter, 0)

	// Create filters for each contract and event type combination
	for _, eventType := range eventTypes {
		filter := ContractEventFilter{
			Type:        eventType,
			ContractIDs: contractIDs,
		}
		filters = append(filters, filter)
	}

	// Get the latest ledger to ensure we scan up to the current point
	latestLedger, err := s.GetLatestLedger(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get latest ledger, proceeding without endLedger")
	}

	request := GetEventsRequest{
		StartLedger: startLedger,
		Filters:     filters,
		Pagination: &EventPagination{
			Limit: 100, // Reasonable batch size
		},
	}

	// If we successfully got the latest ledger, set endLedger to ensure we scan the full range
	if err == nil && latestLedger > startLedger {
		request.EndLedger = &latestLedger
		s.logger.Debug().
			Uint32("start_ledger", startLedger).
			Uint32("end_ledger", latestLedger).
			Msg("Scanning ledger range for events")
	}

	s.logger.Info().
		Interface("request", request).
		Msg("Making RPC request to get contract events")

	rpcReq := SorobanRPCRequest{
		JSONRpc: "2.0",
		ID:      1,
		Method:  "getEvents",
		Params:  request,
	}

	var response GetEventsResponse
	if err := s.makeRPCCall(ctx, rpcReq, &response); err != nil {
		s.logger.Error().
			Err(err).
			Interface("request", request).
			Msg("Failed to make RPC call")
		return nil, fmt.Errorf("failed to get contract events: %w", err)
	}

	s.logger.Info().
		Int("event_count", len(response.Events)).
		Uint32("latest_ledger", response.LatestLedger).
		Str("latest_ledger_time", response.LatestLedgerTime).
		Msg("Retrieved contract events from Soroban RPC")

	// Log details of each event for debugging
	for i, event := range response.Events {
		s.logger.Debug().
			Int("event_index", i).
			Str("event_id", event.ID).
			Str("contract_id", event.ContractID).
			Str("tx_hash", event.TransactionHash).
			Uint32("ledger", event.Ledger).
			Str("event_type", event.Type).
			Bool("successful", event.InSuccessfulContractCall).
			Strs("topics", event.Topic).
			Str("value", event.Value).
			Msg("Event details")
	}

	return &response, nil
}

// GetLatestLedger retrieves the latest ledger number
func (s *SorobanRPCClient) GetLatestLedger(ctx context.Context) (uint32, error) {
	rpcReq := SorobanRPCRequest{
		JSONRpc: "2.0",
		ID:      1,
		Method:  "getLatestLedger",
	}

	var response struct {
		Sequence uint32 `json:"sequence"`
	}

	if err := s.makeRPCCall(ctx, rpcReq, &response); err != nil {
		return 0, fmt.Errorf("failed to get latest ledger: %w", err)
	}

	return response.Sequence, nil
}

// makeRPCCall performs a JSON-RPC call to Soroban
func (s *SorobanRPCClient) makeRPCCall(ctx context.Context, request SorobanRPCRequest, result interface{}) error {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	s.logger.Debug().
		Str("rpc_url", s.rpcURL).
		Str("method", request.Method).
		Str("request_body", string(reqBody)).
		Msg("Making RPC call")

	httpReq, err := retryablehttp.NewRequestWithContext(ctx, "POST", s.rpcURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("rpc_url", s.rpcURL).
			Msg("HTTP request failed")
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	s.logger.Debug().
		Str("rpc_url", s.rpcURL).
		Int("status_code", resp.StatusCode).
		Str("status", resp.Status).
		Msg("Received HTTP response")

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Error().
			Int("status_code", resp.StatusCode).
			Str("status", resp.Status).
			Str("response_body", string(body)).
			Msg("HTTP error response")
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	s.logger.Debug().
		Str("rpc_url", s.rpcURL).
		Str("response_body", string(body)).
		Msg("Received RPC response body")

	var rpcResp SorobanRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		s.logger.Error().
			Err(err).
			Str("response_body", string(body)).
			Msg("Failed to unmarshal RPC response")
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		s.logger.Error().
			Int("error_code", rpcResp.Error.Code).
			Str("error_message", rpcResp.Error.Message).
			Str("error_data", rpcResp.Error.Data).
			Msg("RPC error response")
		return fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	// Marshal result back to JSON and unmarshal into target struct
	resultBytes, err := json.Marshal(rpcResp.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(resultBytes, result); err != nil {
		s.logger.Error().
			Err(err).
			Str("result_json", string(resultBytes)).
			Msg("Failed to unmarshal result")
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	s.logger.Debug().
		Str("rpc_url", s.rpcURL).
		Str("method", request.Method).
		Msg("RPC call completed successfully")

	return nil
}

// ParseContractEvent parses a raw contract event into a RouterEvent
func (s *SorobanRPCClient) ParseContractEvent(event ContractEvent) (*RouterEvent, error) {
	s.logger.Debug().
		Str("event_id", event.ID).
		Str("tx_hash", event.TransactionHash).
		Str("event_type", event.Type).
		Strs("topics", event.Topic).
		Str("value", event.Value).
		Msg("Starting to parse contract event")

	// Parse ledger time
	ledgerTime, err := time.Parse(time.RFC3339, event.LedgerTime)
	if err != nil {
		s.logger.Warn().Err(err).Str("ledger_time", event.LedgerTime).Msg("Failed to parse ledger time")
		ledgerTime = time.Now()
	}

	routerEvent := &RouterEvent{
		Type:            event.Type,
		ContractAddress: event.ContractID,
		TransactionHash: event.TransactionHash,
		Ledger:          event.Ledger,
		LedgerTime:      ledgerTime,
	}

	s.logger.Debug().
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Msg("Created base router event")

	// Parse event data based on type
	switch event.Type {
	case "contract":
		s.logger.Debug().
			Str("event_id", event.ID).
			Msg("Parsing contract event data")
		if err := s.parseContractEventData(event, routerEvent); err != nil {
			s.logger.Warn().
				Err(err).
				Str("event_id", event.ID).
				Str("tx_hash", event.TransactionHash).
				Msg("Failed to parse contract event data")
			return nil, fmt.Errorf("failed to parse contract event data: %w", err)
		}
	case "system":
		// Handle system events if needed
		s.logger.Debug().Str("event_type", "system").Msg("Skipping system event")
		return nil, nil
	default:
		s.logger.Warn().Str("event_type", event.Type).Msg("Unknown event type")
		return nil, nil
	}

	s.logger.Debug().
		Str("event_id", event.ID).
		Str("final_type", routerEvent.Type).
		Str("asset", routerEvent.Asset).
		Str("amount", routerEvent.Amount).
		Msg("Finished parsing contract event")

	return routerEvent, nil
}

// parseContractEventData parses contract-specific event data
func (s *SorobanRPCClient) parseContractEventData(event ContractEvent, routerEvent *RouterEvent) error {
	s.logger.Debug().
		Str("event_id", event.ID).
		Int("topics_count", len(event.Topic)).
		Str("event_value", event.Value).
		Msg("Parsing contract event data")

	// Parse topics (event signature and indexed parameters)
	if len(event.Topic) > 0 {
		// First topic contains the event signature as XDR
		eventSignature, err := s.parseXDRValue(event.Topic[0])
		if err != nil {
			s.logger.Warn().Err(err).Str("topic", event.Topic[0]).Msg("Failed to parse event signature")
		} else {
			// The event signature should be the event name (e.g., "deposit", "transfer_out")
			routerEvent.Type = eventSignature
			s.logger.Debug().Str("event_signature", eventSignature).Msg("Parsed event signature")
		}
	} else {
		s.logger.Debug().Msg("No topics found in event")
	}

	// Parse event value (the main event data)
	if event.Value != "" {
		s.logger.Debug().
			Str("event_id", event.ID).
			Str("value", event.Value).
			Msg("Parsing event value")
		if err := s.parseEventValue(event.Value, routerEvent); err != nil {
			s.logger.Warn().Err(err).Str("value", event.Value).Msg("Failed to parse event value")
			return err
		}
	} else {
		s.logger.Debug().Msg("No event value found")
	}

	s.logger.Debug().
		Str("event_id", event.ID).
		Str("final_type", routerEvent.Type).
		Str("asset", routerEvent.Asset).
		Str("amount", routerEvent.Amount).
		Str("from", routerEvent.FromAddress).
		Str("to", routerEvent.ToAddress).
		Str("memo", routerEvent.Memo).
		Msg("Finished parsing contract event data")

	return nil
}

// parseEventValue parses the event value XDR data
func (s *SorobanRPCClient) parseEventValue(valueXDR string, routerEvent *RouterEvent) error {
	// Decode XDR value
	var scVal xdr.ScVal
	if err := xdr.SafeUnmarshalBase64(valueXDR, &scVal); err != nil {
		return fmt.Errorf("failed to unmarshal XDR: %w", err)
	}

	// Parse the ScVal based on its type
	return s.parseScVal(scVal, routerEvent)
}

// parseScVal parses an XDR ScVal and extracts router event data
func (s *SorobanRPCClient) parseScVal(scVal xdr.ScVal, routerEvent *RouterEvent) error {
	switch scVal.Type {
	case xdr.ScValTypeScvMap:
		// Parse map structure (like our deposit event)
		if scVal.Map != nil && *scVal.Map != nil {
			mapVal := *scVal.Map
			for _, pair := range *mapVal {
				key := s.scValToString(pair.Key)
				value := s.parseScValForEventField(pair.Val)

				// Map the key-value pairs to router event fields
				switch key {
				case "amount":
					routerEvent.Amount = value
				case "asset":
					routerEvent.Asset = value
				case "from":
					routerEvent.FromAddress = value
				case "to":
					routerEvent.ToAddress = value
				case "vault":
					routerEvent.ToAddress = value // Vault is the destination for deposits
				case "memo":
					routerEvent.Memo = value
				case "destination":
					routerEvent.Destination = value
				}

				s.logger.Debug().
					Str("key", key).
					Str("value", value).
					Msg("Parsed event data field")
			}
		}
	case xdr.ScValTypeScvVec:
		// Parse vector/array structure
		if scVal.Vec != nil && *scVal.Vec != nil {
			vec := *scVal.Vec
			for i, item := range *vec {
				s.logger.Debug().
					Int("index", i).
					Str("value", s.scValToString(item)).
					Msg("Parsed event array item")
			}
		}
	default:
		// For simple values, treat as string
		value := s.scValToString(scVal)
		if routerEvent.Amount == "" {
			routerEvent.Amount = value
		}
	}

	return nil
}

// parseScValForEventField parses ScVal specifically for event fields, handling complex types
func (s *SorobanRPCClient) parseScValForEventField(val xdr.ScVal) string {
	switch val.Type {
	case xdr.ScValTypeScvI128:
		// Handle I128 amounts properly
		if val.I128 != nil {
			// Extract the actual numeric value from I128
			i128Val := *val.I128
			// Convert to uint64 for the low part (assuming high part is 0 for reasonable amounts)
			if i128Val.Hi == 0 {
				return strconv.FormatUint(uint64(i128Val.Lo), 10)
			}
			// For larger amounts, we'd need proper big integer handling
			return fmt.Sprintf("%d", i128Val.Lo) // Simplified for now
		}
	case xdr.ScValTypeScvU128:
		// Handle U128 amounts
		if val.U128 != nil {
			u128Val := *val.U128
			if u128Val.Hi == 0 {
				return strconv.FormatUint(uint64(u128Val.Lo), 10)
			}
			return fmt.Sprintf("%d", u128Val.Lo) // Simplified for now
		}
	case xdr.ScValTypeScvAddress:
		// Handle Stellar addresses (including contract addresses)
		if val.Address != nil {
			if addr, err := val.Address.String(); err == nil {
				return addr
			}
		}
	case xdr.ScValTypeScvString:
		// Handle string values
		if val.Str != nil {
			return string(*val.Str)
		}
	case xdr.ScValTypeScvSymbol:
		// Handle symbol values
		if val.Sym != nil {
			return string(*val.Sym)
		}
	}

	// Fall back to the general string conversion
	return s.scValToString(val)
}

// parseXDRValue parses an XDR-encoded value to string
func (s *SorobanRPCClient) parseXDRValue(xdrStr string) (string, error) {
	// Decode XDR value
	var scVal xdr.ScVal
	if err := xdr.SafeUnmarshalBase64(xdrStr, &scVal); err != nil {
		return "", fmt.Errorf("failed to unmarshal XDR: %w", err)
	}

	// Convert XDR value to string representation
	return s.scValToString(scVal), nil
}

// scValToString converts an XDR ScVal to string representation
func (s *SorobanRPCClient) scValToString(val xdr.ScVal) string {
	switch val.Type {
	case xdr.ScValTypeScvBool:
		if val.B != nil && *val.B {
			return "true"
		}
		return "false"
	case xdr.ScValTypeScvVoid:
		return ""
	case xdr.ScValTypeScvU32:
		if val.U32 != nil {
			return strconv.FormatUint(uint64(*val.U32), 10)
		}
	case xdr.ScValTypeScvI32:
		if val.I32 != nil {
			return strconv.FormatInt(int64(*val.I32), 10)
		}
	case xdr.ScValTypeScvU64:
		if val.U64 != nil {
			return strconv.FormatUint(uint64(*val.U64), 10)
		}
	case xdr.ScValTypeScvI64:
		if val.I64 != nil {
			return strconv.FormatInt(int64(*val.I64), 10)
		}
	case xdr.ScValTypeScvU128:
		if val.U128 != nil {
			// Convert U128 to string using the built-in String method
			return val.String()
		}
	case xdr.ScValTypeScvI128:
		if val.I128 != nil {
			// Convert I128 to string using the built-in String method
			return val.String()
		}
	case xdr.ScValTypeScvBytes:
		if val.Bytes != nil {
			return string(*val.Bytes)
		}
	case xdr.ScValTypeScvString:
		if val.Str != nil {
			return string(*val.Str)
		}
	case xdr.ScValTypeScvSymbol:
		if val.Sym != nil {
			return string(*val.Sym)
		}
	case xdr.ScValTypeScvVec:
		if val.Vec != nil && *val.Vec != nil {
			// Convert vector to JSON-like string
			var parts []string
			vec := *val.Vec
			for _, item := range *vec {
				parts = append(parts, s.scValToString(item))
			}
			return "[" + strings.Join(parts, ",") + "]"
		}
	case xdr.ScValTypeScvMap:
		if val.Map != nil && *val.Map != nil {
			// Convert map to JSON-like string
			var parts []string
			mapVal := *val.Map
			for _, entry := range *mapVal {
				key := s.scValToString(entry.Key)
				value := s.scValToString(entry.Val)
				parts = append(parts, fmt.Sprintf("\"%s\":\"%s\"", key, value))
			}
			return "{" + strings.Join(parts, ",") + "}"
		}
	case xdr.ScValTypeScvAddress:
		if val.Address != nil {
			// Convert address to string using the String method
			if addr, err := val.Address.String(); err == nil {
				return addr
			}
		}
	case xdr.ScValTypeScvContractInstance:
		return "contract_instance"
	case xdr.ScValTypeScvLedgerKeyContractInstance:
		return "ledger_key_contract_instance"
	case xdr.ScValTypeScvLedgerKeyNonce:
		return "ledger_key_nonce"
	}

	return ""
}

// IsRouterEvent checks if an event is from a router contract
func (s *SorobanRPCClient) IsRouterEvent(event ContractEvent, routerAddresses []string) bool {
	for _, addr := range routerAddresses {
		if event.ContractID == addr {
			return true
		}
	}
	return false
}

// GetRouterEvents retrieves events specifically from router contracts
func (s *SorobanRPCClient) GetRouterEvents(ctx context.Context, startLedger uint32, routerAddresses []string) ([]*RouterEvent, error) {
	// Define router event types we're interested in
	eventTypes := []string{"contract"}

	s.logger.Debug().
		Uint32("start_ledger", startLedger).
		Strs("router_addresses", routerAddresses).
		Msg("Getting router events from Soroban RPC")

	response, err := s.GetContractEvents(ctx, startLedger, routerAddresses, eventTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to get router events: %w", err)
	}

	s.logger.Debug().
		Uint32("start_ledger", startLedger).
		Int("total_events_received", len(response.Events)).
		Strs("router_addresses", routerAddresses).
		Msg("Received contract events from Soroban RPC")

	var routerEvents []*RouterEvent
	for i, event := range response.Events {
		// Log each event for debugging
		s.logger.Debug().
			Int("event_index", i).
			Str("contract_id", event.ContractID).
			Str("tx_hash", event.TransactionHash).
			Uint32("ledger", event.Ledger).
			Str("event_type", event.Type).
			Bool("successful", event.InSuccessfulContractCall).
			Strs("topics", event.Topic).
			Str("value", event.Value).
			Msg("Processing contract event")

		// Only process events from successful transactions
		if !event.InSuccessfulContractCall {
			s.logger.Debug().
				Str("tx_hash", event.TransactionHash).
				Msg("Skipping event from failed transaction")
			continue
		}

		// Check if this is a router event
		isRouter := s.IsRouterEvent(event, routerAddresses)
		s.logger.Debug().
			Str("contract_id", event.ContractID).
			Strs("expected_addresses", routerAddresses).
			Bool("is_router_event", isRouter).
			Msg("Router event check")

		if !isRouter {
			s.logger.Debug().
				Str("contract_id", event.ContractID).
				Strs("expected_addresses", routerAddresses).
				Msg("Skipping non-router event")
			continue
		}

		routerEvent, err := s.ParseContractEvent(event)
		if err != nil {
			s.logger.Warn().
				Err(err).
				Int("event_index", i).
				Str("event_id", event.ID).
				Str("tx_hash", event.TransactionHash).
				Str("event_value", event.Value).
				Msg("Failed to parse router event")
			continue
		}

		if routerEvent != nil {
			s.logger.Info().
				Str("event_type", routerEvent.Type).
				Str("tx_hash", routerEvent.TransactionHash).
				Str("asset", routerEvent.Asset).
				Str("amount", routerEvent.Amount).
				Str("from", routerEvent.FromAddress).
				Str("to", routerEvent.ToAddress).
				Str("memo", routerEvent.Memo).
				Msg("Successfully parsed router event")
			routerEvents = append(routerEvents, routerEvent)
		} else {
			s.logger.Warn().
				Int("event_index", i).
				Str("event_id", event.ID).
				Str("tx_hash", event.TransactionHash).
				Msg("ParseContractEvent returned nil")
		}
	}

	s.logger.Info().
		Uint32("ledger", startLedger).
		Int("total_events", len(response.Events)).
		Int("router_events", len(routerEvents)).
		Msg("Processed router events")

	return routerEvents, nil
}

// GetContractEventsInRange retrieves contract events within a specific ledger range
func (s *SorobanRPCClient) GetContractEventsInRange(ctx context.Context, startLedger, endLedger uint32, contractIDs []string, eventTypes []string) (*GetEventsResponse, error) {
	filters := make([]ContractEventFilter, 0)

	// Create filters for each contract and event type combination
	for _, eventType := range eventTypes {
		filter := ContractEventFilter{
			Type:        eventType,
			ContractIDs: contractIDs,
		}
		filters = append(filters, filter)
	}

	request := GetEventsRequest{
		StartLedger: startLedger,
		EndLedger:   &endLedger,
		Filters:     filters,
		Pagination: &EventPagination{
			Limit: 100, // Reasonable batch size
		},
	}

	s.logger.Debug().
		Uint32("start_ledger", startLedger).
		Uint32("end_ledger", endLedger).
		Strs("contract_ids", contractIDs).
		Strs("event_types", eventTypes).
		Msg("Scanning specific ledger range for contract events")

	rpcReq := SorobanRPCRequest{
		JSONRpc: "2.0",
		ID:      1,
		Method:  "getEvents",
		Params:  request,
	}

	var response GetEventsResponse
	if err := s.makeRPCCall(ctx, rpcReq, &response); err != nil {
		return nil, fmt.Errorf("failed to get contract events in range: %w", err)
	}

	s.logger.Debug().
		Int("event_count", len(response.Events)).
		Uint32("latest_ledger", response.LatestLedger).
		Uint32("scan_start", startLedger).
		Uint32("scan_end", endLedger).
		Msg("Retrieved contract events from ledger range")

	return &response, nil
}

// GetRouterEventsInRange retrieves router events within a specific ledger range
func (s *SorobanRPCClient) GetRouterEventsInRange(ctx context.Context, startLedger, endLedger uint32, routerAddresses []string) ([]*RouterEvent, error) {
	// Define router event types we're interested in
	eventTypes := []string{"contract"}

	s.logger.Debug().
		Uint32("start_ledger", startLedger).
		Uint32("end_ledger", endLedger).
		Strs("router_addresses", routerAddresses).
		Msg("Getting router events from specific ledger range")

	response, err := s.GetContractEventsInRange(ctx, startLedger, endLedger, routerAddresses, eventTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to get router events in range: %w", err)
	}

	var routerEvents []*RouterEvent
	for _, event := range response.Events {
		// Log each event for debugging
		s.logger.Debug().
			Str("contract_id", event.ContractID).
			Str("tx_hash", event.TransactionHash).
			Uint32("ledger", event.Ledger).
			Str("event_type", event.Type).
			Bool("successful", event.InSuccessfulContractCall).
			Strs("topics", event.Topic).
			Str("value", event.Value).
			Msg("Processing contract event from ledger range")

		// Only process events from successful transactions
		if !event.InSuccessfulContractCall {
			s.logger.Debug().
				Str("tx_hash", event.TransactionHash).
				Msg("Skipping event from failed transaction")
			continue
		}

		if !s.IsRouterEvent(event, routerAddresses) {
			s.logger.Debug().
				Str("contract_id", event.ContractID).
				Strs("expected_addresses", routerAddresses).
				Msg("Skipping non-router event")
			continue
		}

		routerEvent, err := s.ParseContractEvent(event)
		if err != nil {
			s.logger.Warn().
				Err(err).
				Str("event_id", event.ID).
				Str("tx_hash", event.TransactionHash).
				Msg("Failed to parse router event")
			continue
		}

		if routerEvent != nil {
			s.logger.Info().
				Str("event_type", routerEvent.Type).
				Str("tx_hash", routerEvent.TransactionHash).
				Str("asset", routerEvent.Asset).
				Str("amount", routerEvent.Amount).
				Str("from", routerEvent.FromAddress).
				Str("to", routerEvent.ToAddress).
				Str("memo", routerEvent.Memo).
				Uint32("ledger", routerEvent.Ledger).
				Msg("Successfully parsed router event from ledger range")
			routerEvents = append(routerEvents, routerEvent)
		}
	}

	s.logger.Info().
		Uint32("start_ledger", startLedger).
		Uint32("end_ledger", endLedger).
		Int("total_events", len(response.Events)).
		Int("router_events", len(routerEvents)).
		Msg("Processed router events from ledger range")

	return routerEvents, nil
}

// New helpers for send/get transaction

type sendTransactionParams struct {
	Transaction string `json:"transaction"`
}

type sendTransactionResult struct {
	Hash string `json:"hash"`
}

type getTransactionParams struct {
	Hash string `json:"hash"`
}

type getTransactionResult struct {
	Status string `json:"status"`
}

// prepareTransaction structures

type prepareTransactionParams struct {
	Transaction string `json:"transaction"`
}

type prepareTransactionResult struct {
	TransactionData string `json:"transactionData"`
	MinResourceFee  string `json:"minResourceFee"`
}

func (s *SorobanRPCClient) SendTransaction(ctx context.Context, txXDR string) (string, error) {
	req := SorobanRPCRequest{JSONRpc: "2.0", ID: 1, Method: "sendTransaction", Params: sendTransactionParams{Transaction: txXDR}}
	var res sendTransactionResult
	if err := s.makeRPCCall(ctx, req, &res); err != nil {
		return "", err
	}
	if res.Hash == "" {
		return "", fmt.Errorf("sendTransaction returned empty hash")
	}
	return res.Hash, nil
}

func (s *SorobanRPCClient) GetTransaction(ctx context.Context, hash string) (string, error) {
	req := SorobanRPCRequest{JSONRpc: "2.0", ID: 1, Method: "getTransaction", Params: getTransactionParams{Hash: hash}}
	var res getTransactionResult
	if err := s.makeRPCCall(ctx, req, &res); err != nil {
		return "", err
	}
	s.logger.Debug().Str("tx_hash", hash).Str("status", res.Status).Msg("polled soroban tx status")
	return res.Status, nil
}

func (s *SorobanRPCClient) PrepareTransaction(ctx context.Context, txXDR string) (prepareTransactionResult, error) {
	req := SorobanRPCRequest{JSONRpc: "2.0", ID: 1, Method: "prepareTransaction", Params: prepareTransactionParams{Transaction: txXDR}}
	var res prepareTransactionResult
	if err := s.makeRPCCall(ctx, req, &res); err != nil {
		return prepareTransactionResult{}, err
	}
	return res, nil
}
