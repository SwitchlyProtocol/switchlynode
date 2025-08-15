package switchlyclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blang/semver"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/switchlyprotocol/switchlynode/v3/app"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/config"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
	openapi "github.com/switchlyprotocol/switchlynode/v3/openapi/gen"
	stypes "github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

// Endpoint urls
const (
	AuthAccountEndpoint      = "/cosmos/auth/v1beta1/accounts"
	BroadcastTxsEndpoint     = "/"
	KeygenEndpoint           = "/switchly/keygen"
	KeysignEndpoint          = "/switchly/keysign"
	LastBlockEndpoint        = "/switchly/lastblock"
	NodeAccountEndpoint      = "/switchly/node"
	NodeAccountsEndpoint     = "/switchly/nodes"
	SignerMembershipEndpoint = "/switchly/vaults/%s/signers"
	StatusEndpoint           = "/status"
	VaultEndpoint            = "/switchly/vault/%s"
	AsgardVault              = "/switchly/vaults/asgard"
	PubKeysEndpoint          = "/switchly/vaults/pubkeys"
	SwitchlyConstants        = "/switchly/constants"
	RagnarokEndpoint         = "/switchly/ragnarok"
	MimirEndpoint            = "/switchly/mimir"
	ChainVersionEndpoint     = "/switchly/version"
	InboundAddressesEndpoint = "/switchly/inbound_addresses"
	PoolsEndpoint            = "/switchly/pools"
	SWITCHNameEndpoint       = "/switchly/switchlyname/%s"
)

// switchlyBridge will be used to send tx to SWITCHLYChain
type switchlyBridge struct {
	logger        zerolog.Logger
	cfg           config.BifrostClientConfiguration
	keys          *Keys
	errCounter    *prometheus.CounterVec
	m             *metrics.Metrics
	blockHeight   int64
	accountNumber uint64
	seqNumber     uint64
	httpClient    *retryablehttp.Client
	broadcastLock *sync.RWMutex
}

type SwitchlyBridge interface {
	EnsureNodeWhitelisted() error
	EnsureNodeWhitelistedWithTimeout() error
	FetchNodeStatus() (stypes.NodeStatus, error)
	FetchActiveNodes() ([]common.PubKey, error)
	GetAsgards() (stypes.Vaults, error)
	GetVault(pubkey string) (stypes.Vault, error)
	GetConfig() config.BifrostClientConfiguration
	GetConstants() (map[string]int64, error)
	GetContext() client.Context
	GetContractAddress() ([]PubKeyContractAddressPair, error)
	GetErrataMsg(txID common.TxID, chain common.Chain) sdk.Msg
	GetKeygenStdTx(poolPubKey common.PubKey, secp256k1Signature, keysharesBackup []byte, blame stypes.Blame, inputPks common.PubKeys, keygenType stypes.KeygenType, chains common.Chains, height, keygenTime int64) (sdk.Msg, error)
	GetKeysignParty(vaultPubKey common.PubKey) (common.PubKeys, error)
	GetMimir(key string) (int64, error)
	GetMimirWithRef(template, ref string) (int64, error)
	GetInboundOutbound(txIns common.ObservedTxs) (common.ObservedTxs, common.ObservedTxs, error)
	GetPools() (stypes.Pools, error)
	GetPubKeys() ([]PubKeyContractAddressPair, error)
	GetAsgardPubKeys() ([]PubKeyContractAddressPair, error)
	GetSolvencyMsg(height int64, chain common.Chain, pubKey common.PubKey, coins common.Coins) *stypes.MsgSolvency
	GetSWITCHName(name string) (stypes.SWITCHName, error)
	GetSwitchlyVersion() (semver.Version, error)
	IsCatchingUp() (bool, error)
	HasNetworkFee(chain common.Chain) (bool, error)
	GetNetworkFee(chain common.Chain) (transactionSize, transactionFeeRate uint64, err error)
	PostKeysignFailure(blame stypes.Blame, height int64, memo string, coins common.Coins, pubkey common.PubKey) (common.TxID, error)
	PostNetworkFee(height int64, chain common.Chain, transactionSize, transactionRate uint64) (common.TxID, error)
	RagnarokInProgress() (bool, error)
	WaitToCatchUp() error
	GetBlockHeight() (int64, error)
	GetLastObservedInHeight(chain common.Chain) (int64, error)
	GetLastSignedOutHeight(chain common.Chain) (int64, error)
	Broadcast(msgs ...sdk.Msg) (common.TxID, error)
	GetKeysign(blockHeight int64, pk string) (types.TxOut, error)
	GetNodeAccount(string) (*stypes.NodeAccount, error)
	GetNodeAccounts() ([]*stypes.NodeAccount, error)
	GetKeygenBlock(int64, string) (stypes.KeygenBlock, error)
}

// httpResponseCache used for caching HTTP responses for less frequent querying
type httpResponseCache struct {
	httpResponse        []byte
	httpResponseChecked time.Time
	httpResponseMu      *sync.Mutex
}

var (
	httpResponseCaches   = make(map[string]*httpResponseCache) // String-to-pointer map for quicker lookup
	httpResponseCachesMu = &sync.Mutex{}
)

// NewSwitchlyBridge create a new instance of SwitchlyBridge
func NewSwitchlyBridge(cfg config.BifrostClientConfiguration, m *metrics.Metrics, k *Keys) (SwitchlyBridge, error) {
	// main module logger
	logger := log.With().Str("module", "switchly_client").Logger()

	if len(cfg.ChainID) == 0 {
		return nil, errors.New("chain id is empty")
	}
	if len(cfg.ChainHost) == 0 {
		return nil, errors.New("chain host is empty")
	}

	httpClient := retryablehttp.NewClient()
	httpClient.Logger = nil

	return &switchlyBridge{
		logger:        logger,
		cfg:           cfg,
		keys:          k,
		errCounter:    m.GetCounterVec(metrics.SwitchlyClientError),
		httpClient:    httpClient,
		m:             m,
		broadcastLock: &sync.RWMutex{},
	}, nil
}

func MakeCodec() codec.ProtoCodecMarshaler {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	stypes.RegisterInterfaces(interfaceRegistry)
	return codec.NewProtoCodec(interfaceRegistry)
}

// GetContext return a valid context with all relevant values set
func (b *switchlyBridge) GetContext() client.Context {
	signerAddr, err := b.keys.GetSignerInfo().GetAddress()
	if err != nil {
		panic(err)
	}
	ctx := client.Context{}
	ctx = ctx.WithKeyring(b.keys.GetKeybase())
	ctx = ctx.WithChainID(string(b.cfg.ChainID))
	ctx = ctx.WithHomeDir(b.cfg.ChainHomeFolder)
	ctx = ctx.WithFromName(b.cfg.SignerName)
	ctx = ctx.WithFromAddress(signerAddr)
	ctx = ctx.WithBroadcastMode("sync")

	encodingConfig := app.MakeEncodingConfig()
	ctx = ctx.WithCodec(encodingConfig.Codec)
	ctx = ctx.WithInterfaceRegistry(encodingConfig.InterfaceRegistry)
	ctx = ctx.WithTxConfig(encodingConfig.TxConfig)
	ctx = ctx.WithLegacyAmino(encodingConfig.Amino)
	ctx = ctx.WithAccountRetriever(authtypes.AccountRetriever{})

	remote := b.cfg.ChainRPC
	if !strings.HasPrefix(b.cfg.ChainHost, "http") {
		remote = fmt.Sprintf("tcp://%s", remote)
	}
	ctx = ctx.WithNodeURI(remote)
	client, err := rpchttp.New(remote, "/websocket")
	if err != nil {
		panic(err)
	}
	ctx = ctx.WithClient(client)
	return ctx
}

func (b *switchlyBridge) getWithPath(path string) ([]byte, int, error) {
	return b.get(b.getThorChainURL(path))
}

// get handle all the low level http GET calls using retryablehttp.SwitchlyBridge
func (b *switchlyBridge) get(url string) ([]byte, int, error) {
	// To reduce querying time and chance of "429 Too Many Requests",
	// do not query the same endpoint more than once per block time.
	httpResponseCachesMu.Lock()
	respCachePointer := httpResponseCaches[url]
	if respCachePointer == nil {
		// Since this is the first time using this endpoint, prepare a Mutex for it.
		respCachePointer = &httpResponseCache{httpResponseMu: &sync.Mutex{}}
		httpResponseCaches[url] = respCachePointer
	}
	httpResponseCachesMu.Unlock()

	// So lengthy queries don't hold up short queries, use query-specific mutexes.
	respCachePointer.httpResponseMu.Lock()
	defer respCachePointer.httpResponseMu.Unlock()

	// When the same endpoint has been checked within the span of a single block, return the cached response.
	if time.Since(respCachePointer.httpResponseChecked) < constants.SwitchlyBlockTime && respCachePointer.httpResponse != nil {
		return respCachePointer.httpResponse, http.StatusOK, nil
	}

	resp, err := b.httpClient.Get(url)
	if err != nil {
		b.errCounter.WithLabelValues("fail_get_from_switchly", "").Inc()
		return nil, http.StatusNotFound, fmt.Errorf("failed to GET from switchly: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			b.logger.Error().Err(err).Msg("failed to close response body")
		}
	}()

	buf, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return buf, resp.StatusCode, errors.New("Status code: " + resp.Status + " returned")
	}
	if err != nil {
		b.errCounter.WithLabelValues("fail_read_switchly_resp", "").Inc()
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	// All being well with the response, save it to the cache.
	respCachePointer.httpResponse = buf
	respCachePointer.httpResponseChecked = time.Now()

	return buf, resp.StatusCode, nil
}

// getThorChainURL with the given path
func (b *switchlyBridge) getThorChainURL(path string) string {
	if strings.HasPrefix(b.cfg.ChainHost, "http") {
		return fmt.Sprintf("%s/%s", b.cfg.ChainHost, path)
	}

	uri := url.URL{
		Scheme: "http",
		Host:   b.cfg.ChainHost,
		Path:   path,
	}
	return uri.String()
}

// getAccountNumberAndSequenceNumber returns account and Sequence number required to post into switchly
func (b *switchlyBridge) getAccountNumberAndSequenceNumber() (uint64, uint64, error) {
	signerAddr, err := b.keys.GetSignerInfo().GetAddress()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get signer address: %w", err)
	}
	path := fmt.Sprintf("%s/%s", AuthAccountEndpoint, signerAddr)

	body, _, err := b.getWithPath(path)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get auth accounts: %w", err)
	}

	var resp types.AccountResp
	if err = json.Unmarshal(body, &resp); err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal account resp: %w", err)
	}
	acc := resp.Account

	return acc.AccountNumber, acc.Sequence, nil
}

// GetConfig return the configuration
func (b *switchlyBridge) GetConfig() config.BifrostClientConfiguration {
	return b.cfg
}

// PostKeysignFailure generate and  post a keysign fail tx to thorchan
func (b *switchlyBridge) PostKeysignFailure(blame stypes.Blame, height int64, memo string, coins common.Coins, pubkey common.PubKey) (common.TxID, error) {
	start := time.Now()
	defer func() {
		b.m.GetHistograms(metrics.SignToSwitchlyDuration).Observe(time.Since(start).Seconds())
	}()

	if blame.IsEmpty() {
		// MsgTssKeysignFail will fail validation if having no FailReason.
		blame.FailReason = "no fail reason available"
	}
	signerAddr, err := b.keys.GetSignerInfo().GetAddress()
	if err != nil {
		return common.BlankTxID, fmt.Errorf("failed to get signer address: %w", err)
	}
	msg, err := stypes.NewMsgTssKeysignFail(height, blame, memo, coins, signerAddr, pubkey)
	if err != nil {
		return common.BlankTxID, fmt.Errorf("fail to create keysign fail message: %w", err)
	}
	return b.Broadcast(msg)
}

// GetErrataMsg get errata tx from params
func (b *switchlyBridge) GetErrataMsg(txID common.TxID, chain common.Chain) sdk.Msg {
	signerAddr, err := b.keys.GetSignerInfo().GetAddress()
	if err != nil {
		panic(err)
	}
	return stypes.NewMsgErrataTx(txID, chain, signerAddr)
}

// GetSolvencyMsg create MsgSolvency from the given parameters
func (b *switchlyBridge) GetSolvencyMsg(height int64, chain common.Chain, pubKey common.PubKey, coins common.Coins) *stypes.MsgSolvency {
	// To prevent different MsgSolvency ID incompatibility between nodes with different coin-observation histories,
	// only report coins for which the amounts are not currently 0.
	coins = coins.NoneEmpty()
	signerAddr, err := b.keys.GetSignerInfo().GetAddress()
	if err != nil {
		panic(err)
	}
	msg, err := stypes.NewMsgSolvency(chain, pubKey, coins, height, signerAddr)
	if err != nil {
		b.logger.Err(err).Msg("fail to create MsgSolvency")
		return nil
	}
	return msg
}

// GetKeygenStdTx get keygen tx from params
func (b *switchlyBridge) GetKeygenStdTx(poolPubKey common.PubKey, secp256k1Signature, keysharesBackup []byte, blame stypes.Blame, inputPks common.PubKeys, keygenType stypes.KeygenType, chains common.Chains, height, keygenTime int64) (sdk.Msg, error) {
	signerAddr, err := b.keys.GetSignerInfo().GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get signer address: %w", err)
	}
	return stypes.NewMsgTssPool(inputPks.Strings(), poolPubKey, secp256k1Signature, keysharesBackup, keygenType, height, blame, chains.Strings(), signerAddr, keygenTime)
}

// GetInboundOutbound separate the txs into inbound and outbound
func (b *switchlyBridge) GetInboundOutbound(txIns common.ObservedTxs) (common.ObservedTxs, common.ObservedTxs, error) {
	if len(txIns) == 0 {
		return nil, nil, nil
	}
	inbound := common.ObservedTxs{}
	outbound := common.ObservedTxs{}

	// spilt our txs into inbound vs outbound txs
	for _, tx := range txIns {
		chain := common.EmptyChain
		if len(tx.Tx.Coins) > 0 {
			chain = tx.Tx.Coins[0].Asset.Chain
		}

		obAddr, err := tx.ObservedPubKey.GetAddress(chain)
		if err != nil {
			b.logger.Err(err).Msgf("fail to parse observed pool address: %s", tx.ObservedPubKey.String())
			continue
		}
		vaultToAddress := tx.Tx.ToAddress.Equals(obAddr)
		vaultFromAddress := tx.Tx.FromAddress.Equals(obAddr)
		var inInboundArray, inOutboundArray bool
		if vaultToAddress {
			inInboundArray = inbound.Contains(tx)
		}
		if vaultFromAddress {
			inOutboundArray = outbound.Contains(tx)
		}
		// for consolidate UTXO tx, both From & To address will be the asgard address
		// thus here we need to make sure that one add to inbound , the other add to outbound
		switch {
		case !vaultToAddress && !vaultFromAddress:
			// Neither ToAddress nor FromAddress matches obAddr, so drop it.
			b.logger.Error().Msgf("chain (%s) tx (%s) observedaddress (%s) does not match its toaddress (%s) or fromaddress (%s)", tx.Tx.Chain, tx.Tx.ID, obAddr, tx.Tx.ToAddress, tx.Tx.FromAddress)
		case vaultToAddress && !inInboundArray:
			inbound = append(inbound, tx)
		case vaultFromAddress && !inOutboundArray:
			outbound = append(outbound, tx)
		case inInboundArray && inOutboundArray:
			// It's already in both arrays, so drop it.
			b.logger.Error().Msgf("vault-to-vault chain (%s) tx (%s) is already in both inbound and outbound arrays", tx.Tx.Chain, tx.Tx.ID)
		case !vaultFromAddress && inInboundArray:
			// It's already in its only (inbound) array, so drop it.
			b.logger.Error().Msgf("observed tx in for chain (%s) tx (%s) is already in the inbound array", tx.Tx.Chain, tx.Tx.ID)
		case !vaultToAddress && inOutboundArray:
			// It's already in its only (outbound) array, so drop it.
			b.logger.Error().Msgf("observed tx out for chain (%s) tx (%s) is already in the outbound array", tx.Tx.Chain, tx.Tx.ID)
		default:
			// This should never happen; rather than dropping it, return an error.
			return nil, nil, fmt.Errorf("could not determine if chain (%s) tx (%s) was inbound or outbound", tx.Tx.Chain, tx.Tx.ID)
		}
	}

	return inbound, outbound, nil
}

// EnsureNodeWhitelistedWithTimeout check node is whitelisted with timeout retry
func (b *switchlyBridge) EnsureNodeWhitelistedWithTimeout() error {
	for {
		select {
		case <-time.After(time.Hour):
			return errors.New("Observer is not whitelisted yet")
		default:
			err := b.EnsureNodeWhitelisted()
			if err == nil {
				// node had been whitelisted
				return nil
			}
			b.logger.Error().Err(err).Msg("observer is not whitelisted , will retry a bit later")
			time.Sleep(time.Second * 5)
		}
	}
}

// EnsureNodeWhitelisted will call to switchly to check whether the observer had been whitelist or not
func (b *switchlyBridge) EnsureNodeWhitelisted() error {
	status, err := b.FetchNodeStatus()
	if err != nil {
		return fmt.Errorf("failed to get node status: %w", err)
	}
	if status == stypes.NodeStatus_Unknown {
		return fmt.Errorf("node account status %s , will not be able to forward transaction to switchly", status)
	}
	return nil
}

func (b *switchlyBridge) FetchActiveNodes() ([]common.PubKey, error) {
	na, err := b.GetNodeAccounts()
	if err != nil {
		return nil, fmt.Errorf("fail to get node accounts: %w", err)
	}
	active := make([]common.PubKey, 0)
	for _, item := range na {
		if item.Status == stypes.NodeStatus_Active {
			active = append(active, item.PubKeySet.Secp256k1)
		}
	}
	return active, nil
}

// FetchNodeStatus get current node status from switchly
func (b *switchlyBridge) FetchNodeStatus() (stypes.NodeStatus, error) {
	signerAddr, err := b.keys.GetSignerInfo().GetAddress()
	if err != nil {
		return stypes.NodeStatus_Unknown, fmt.Errorf("fail to get signer address: %w", err)
	}
	bepAddr := signerAddr.String()
	if len(bepAddr) == 0 {
		return stypes.NodeStatus_Unknown, errors.New("bep address is empty")
	}
	na, err := b.GetNodeAccount(bepAddr)
	if err != nil {
		return stypes.NodeStatus_Unknown, fmt.Errorf("failed to get node status: %w", err)
	}
	return na.Status, nil
}

// GetKeysignParty call into switchly to get the node accounts that should be join together to sign the message
func (b *switchlyBridge) GetKeysignParty(vaultPubKey common.PubKey) (common.PubKeys, error) {
	p := fmt.Sprintf(SignerMembershipEndpoint, vaultPubKey.String())
	result, _, err := b.getWithPath(p)
	if err != nil {
		return common.PubKeys{}, fmt.Errorf("fail to get key sign party from switchly: %w", err)
	}
	var keys common.PubKeys
	if err = json.Unmarshal(result, &keys); err != nil {
		return common.PubKeys{}, fmt.Errorf("fail to unmarshal result to pubkeys:%w", err)
	}
	return keys, nil
}

// IsCatchingUp returns bool for if switchly is catching up to the rest of the
// nodes. Returns yes, if it is, false if it is caught up.
func (b *switchlyBridge) IsCatchingUp() (bool, error) {
	uri := url.URL{
		Scheme: "http",
		Host:   b.cfg.ChainRPC,
		Path:   StatusEndpoint,
	}

	body, _, err := b.get(uri.String())
	if err != nil {
		return false, fmt.Errorf("failed to get status data: %w", err)
	}

	var resp struct {
		Result struct {
			SyncInfo struct {
				CatchingUp bool `json:"catching_up"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	if err = json.Unmarshal(body, &resp); err != nil {
		return false, fmt.Errorf("failed to unmarshal tendermint status: %w", err)
	}
	return resp.Result.SyncInfo.CatchingUp, nil
}

// HasNetworkFee checks whether the given chain has set a network fee - determined by
// whether the `outbound_tx_size` for the inbound address response is non-zero.
func (b *switchlyBridge) HasNetworkFee(chain common.Chain) (bool, error) {
	buf, s, err := b.getWithPath(InboundAddressesEndpoint)
	if err != nil {
		return false, fmt.Errorf("fail to get inbound addresses: %w", err)
	}
	if s != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", s)
	}

	var resp []openapi.InboundAddress
	if err = json.Unmarshal(buf, &resp); err != nil {
		return false, fmt.Errorf("fail to unmarshal inbound addresses: %w", err)
	}

	for _, addr := range resp {
		if addr.Chain != nil && *addr.Chain == chain.String() && addr.OutboundTxSize != nil {
			var size int64
			size, err = strconv.ParseInt(*addr.OutboundTxSize, 10, 64)
			if err != nil {
				return false, fmt.Errorf("fail to parse outbound_tx_size: %w", err)
			}
			return size > 0, nil
		}
	}

	return false, fmt.Errorf("no inbound address found for chain: %s", chain)
}

// GetNetworkFee get chain's network fee from SWITCHLYNode.
func (b *switchlyBridge) GetNetworkFee(chain common.Chain) (transactionSize, transactionFeeRate uint64, err error) {
	buf, s, err := b.getWithPath(InboundAddressesEndpoint)
	if err != nil {
		return 0, 0, fmt.Errorf("fail to get inbound addresses: %w", err)
	}
	if s != http.StatusOK {
		return 0, 0, fmt.Errorf("unexpected status code: %d", s)
	}
	var resp []openapi.InboundAddress
	if err = json.Unmarshal(buf, &resp); err != nil {
		return 0, 0, fmt.Errorf("fail to unmarshal to json: %w", err)
	}

	for _, addr := range resp {
		if addr.Chain != nil && *addr.Chain == chain.String() {
			// Default values if nil or unfound are 0.
			if addr.OutboundTxSize != nil {
				transactionSize, err = strconv.ParseUint(*addr.OutboundTxSize, 10, 64)
				if err != nil {
					return 0, 0, fmt.Errorf("fail to parse outbound_tx_size: %w", err)
				}
			}
			if addr.ObservedFeeRate != nil {
				transactionFeeRate, err = strconv.ParseUint(*addr.ObservedFeeRate, 10, 64)
				if err != nil {
					return 0, 0, fmt.Errorf("fail to parse observed_fee_rate: %w", err)
				}
			}
			// Having found the chain, do not continue through the remaining chains.
			break
		}
	}
	return
}

// WaitToCatchUp wait for switchly to catch up
func (b *switchlyBridge) WaitToCatchUp() error {
	for {
		yes, err := b.IsCatchingUp()
		if err != nil {
			return err
		}
		if !yes {
			break
		}
		b.logger.Info().Msg("switchly is not caught up... waiting...")
		time.Sleep(constants.SwitchlyBlockTime)
	}
	return nil
}

// GetAsgards retrieve all the asgard vaults from switchly
func (b *switchlyBridge) GetAsgards() (stypes.Vaults, error) {
	buf, s, err := b.getWithPath(AsgardVault)
	if err != nil {
		return nil, fmt.Errorf("fail to get asgard vaults: %w", err)
	}
	if s != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", s)
	}
	var vaults stypes.Vaults
	if err = json.Unmarshal(buf, &vaults); err != nil {
		return nil, fmt.Errorf("fail to unmarshal asgard vaults from json: %w", err)
	}
	return vaults, nil
}

// GetVault retrieves a specific vault from switchly.
func (b *switchlyBridge) GetVault(pubkey string) (stypes.Vault, error) {
	buf, s, err := b.getWithPath(fmt.Sprintf(VaultEndpoint, pubkey))
	if err != nil {
		return stypes.Vault{}, fmt.Errorf("fail to get vault: %w", err)
	}
	if s != http.StatusOK {
		return stypes.Vault{}, fmt.Errorf("unexpected status code %d", s)
	}
	var vault stypes.Vault
	if err = json.Unmarshal(buf, &vault); err != nil {
		return stypes.Vault{}, fmt.Errorf("fail to unmarshal vault from json: %w", err)
	}
	return vault, nil
}

func (b *switchlyBridge) getVaultPubkeys() ([]byte, error) {
	buf, s, err := b.getWithPath(PubKeysEndpoint)
	if err != nil {
		return nil, fmt.Errorf("fail to get asgard vaults: %w", err)
	}
	if s != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", s)
	}
	return buf, nil
}

// GetPubKeys retrieve vault pub keys and their relevant smart contracts
func (b *switchlyBridge) GetPubKeys() ([]PubKeyContractAddressPair, error) {
	buf, err := b.getVaultPubkeys()
	if err != nil {
		return nil, fmt.Errorf("fail to get vault pubkeys ,err: %w", err)
	}
	var result openapi.VaultPubkeysResponse
	if err = json.Unmarshal(buf, &result); err != nil {
		return nil, fmt.Errorf("fail to unmarshal pubkeys: %w", err)
	}
	var addressPairs []PubKeyContractAddressPair
	for _, v := range append(result.Asgard, result.Inactive...) {
		kp := PubKeyContractAddressPair{
			PubKey:    common.PubKey(v.PubKey),
			Contracts: make(map[common.Chain]common.Address),
		}
		for _, item := range v.Routers {
			kp.Contracts[common.Chain(*item.Chain)] = common.Address(*item.Router)
		}
		addressPairs = append(addressPairs, kp)
	}
	return addressPairs, nil
}

// GetAsgardPubKeys retrieve asgard vaults, and it's relevant smart contracts
func (b *switchlyBridge) GetAsgardPubKeys() ([]PubKeyContractAddressPair, error) {
	buf, err := b.getVaultPubkeys()
	if err != nil {
		return nil, fmt.Errorf("fail to get vault pubkeys ,err: %w", err)
	}
	var result openapi.VaultPubkeysResponse
	if err = json.Unmarshal(buf, &result); err != nil {
		return nil, fmt.Errorf("fail to unmarshal pubkeys: %w", err)
	}
	var addressPairs []PubKeyContractAddressPair
	for _, v := range append(result.Asgard, result.Inactive...) {
		kp := PubKeyContractAddressPair{
			PubKey:    common.PubKey(v.PubKey),
			Contracts: make(map[common.Chain]common.Address),
		}
		for _, item := range v.Routers {
			kp.Contracts[common.Chain(*item.Chain)] = common.Address(*item.Router)
		}
		addressPairs = append(addressPairs, kp)
	}
	return addressPairs, nil
}

// PostNetworkFee send network fee message to SWITCHLYNode
func (b *switchlyBridge) PostNetworkFee(height int64, chain common.Chain, transactionSize, transactionRate uint64) (common.TxID, error) {
	nodeStatus, err := b.FetchNodeStatus()
	if err != nil {
		return common.BlankTxID, fmt.Errorf("failed to get node status: %w", err)
	}

	if nodeStatus != stypes.NodeStatus_Active {
		return common.BlankTxID, nil
	}
	start := time.Now()
	defer func() {
		b.m.GetHistograms(metrics.SignToSwitchlyDuration).Observe(time.Since(start).Seconds())
	}()
	signerAddr, err := b.keys.GetSignerInfo().GetAddress()
	if err != nil {
		return common.BlankTxID, fmt.Errorf("fail to get signer address: %w", err)
	}
	msg := stypes.NewMsgNetworkFee(height, chain, transactionSize, transactionRate, signerAddr)
	return b.Broadcast(msg)
}

// GetConstants from switchlynode
func (b *switchlyBridge) GetConstants() (map[string]int64, error) {
	var result struct {
		Int64Values map[string]int64 `json:"int_64_values"`
	}
	buf, s, err := b.getWithPath(SwitchlyConstants)
	if err != nil {
		return nil, fmt.Errorf("fail to get constants: %w", err)
	}
	if s != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", s)
	}
	if err = json.Unmarshal(buf, &result); err != nil {
		return nil, fmt.Errorf("fail to unmarshal to json: %w", err)
	}
	return result.Int64Values, nil
}

// RagnarokInProgress is to query switchly to check whether ragnarok had been triggered
func (b *switchlyBridge) RagnarokInProgress() (bool, error) {
	buf, s, err := b.getWithPath(RagnarokEndpoint)
	if err != nil {
		return false, fmt.Errorf("fail to get ragnarok status: %w", err)
	}
	if s != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", s)
	}
	var ragnarok bool
	if err = json.Unmarshal(buf, &ragnarok); err != nil {
		return false, fmt.Errorf("fail to unmarshal ragnarok status: %w", err)
	}
	return ragnarok, nil
}

// GetSwitchlyVersion retrieve switchly version
func (b *switchlyBridge) GetSwitchlyVersion() (semver.Version, error) {
	buf, s, err := b.getWithPath(ChainVersionEndpoint)
	if err != nil {
		return semver.Version{}, fmt.Errorf("fail to get SWITCHLYChain version: %w", err)
	}
	if s != http.StatusOK {
		return semver.Version{}, fmt.Errorf("unexpected status code: %d", s)
	}
	var version openapi.VersionResponse
	if err = json.Unmarshal(buf, &version); err != nil {
		return semver.Version{}, fmt.Errorf("fail to unmarshal SWITCHLYChain version : %w", err)
	}
	return semver.MustParse(version.Current), nil
}

// GetMimir - get mimir settings
func (b *switchlyBridge) GetMimir(key string) (int64, error) {
	buf, s, err := b.getWithPath(MimirEndpoint + "/key/" + key)
	if err != nil {
		return 0, fmt.Errorf("fail to get mimir: %w", err)
	}
	if s != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", s)
	}
	var value int64
	if err = json.Unmarshal(buf, &value); err != nil {
		return 0, fmt.Errorf("fail to unmarshal mimir: %w", err)
	}
	return value, nil
}

// GetMimirWithRef is a helper function to more readably insert references (such as Asset MimirString or Chain) into Mimir key templates.
func (b *switchlyBridge) GetMimirWithRef(template, ref string) (int64, error) {
	// 'template' should be something like "Halt%sChain" (to halt an arbitrary specified chain)
	// or "Ragnarok-%s" (to halt the pool of an arbitrary specified Asset (MimirString used for Assets to join Chain and Symbol with a hyphen).
	key := fmt.Sprintf(template, ref)
	return b.GetMimir(key)
}

// PubKeyContractAddressPair is an entry to map pubkey and contract addresses
type PubKeyContractAddressPair struct {
	PubKey    common.PubKey
	Contracts map[common.Chain]common.Address
}

// GetContractAddress retrieve the contract address from asgard
func (b *switchlyBridge) GetContractAddress() ([]PubKeyContractAddressPair, error) {
	buf, s, err := b.getWithPath(InboundAddressesEndpoint)
	if err != nil {
		return nil, fmt.Errorf("fail to get inbound addresses: %w", err)
	}
	if s != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", s)
	}
	type address struct {
		Chain   common.Chain   `json:"chain"`
		PubKey  common.PubKey  `json:"pub_key"`
		Address common.Address `json:"address"`
		Router  common.Address `json:"router"`
		Halted  bool           `json:"halted"`
	}
	var resp []address
	if err = json.Unmarshal(buf, &resp); err != nil {
		return nil, fmt.Errorf("fail to unmarshal response: %w", err)
	}
	var result []PubKeyContractAddressPair
	for _, item := range resp {
		exist := false
		for _, pair := range result {
			if item.PubKey.Equals(pair.PubKey) {
				pair.Contracts[item.Chain] = item.Router
				exist = true
				break
			}
		}
		if !exist {
			pair := PubKeyContractAddressPair{
				PubKey:    item.PubKey,
				Contracts: map[common.Chain]common.Address{},
			}
			pair.Contracts[item.Chain] = item.Router
			result = append(result, pair)
		}
	}
	return result, nil
}

// GetPools get pools from SWITCHLYChain
func (b *switchlyBridge) GetPools() (stypes.Pools, error) {
	buf, s, err := b.getWithPath(PoolsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("fail to get pools addresses: %w", err)
	}
	if s != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", s)
	}
	var pools stypes.Pools
	if err = json.Unmarshal(buf, &pools); err != nil {
		return nil, fmt.Errorf("fail to unmarshal pools from json: %w", err)
	}
	return pools, nil
}

// GetSWITCHName get SWITCHName from SWITCHLYChain
func (b *switchlyBridge) GetSWITCHName(name string) (stypes.SWITCHName, error) {
	p := fmt.Sprintf(SWITCHNameEndpoint, name)
	buf, s, err := b.getWithPath(p)
	if err != nil {
		return stypes.SWITCHName{}, fmt.Errorf("fail to get SWITCHName: %w", err)
	}
	if s != http.StatusOK {
		return stypes.SWITCHName{}, fmt.Errorf("unexpected status code: %d", s)
	}
	var tn stypes.SWITCHName
	if err = json.Unmarshal(buf, &tn); err != nil {
		return stypes.SWITCHName{}, fmt.Errorf("fail to unmarshal SWITCHNames from json: %w", err)
	}
	return tn, nil
}
