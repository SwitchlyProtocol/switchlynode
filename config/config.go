package config

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	tmhttp "github.com/cometbft/cometbft/rpc/client/http"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/switchlyprotocol/switchlynode/v1/common"
)

// -------------------------------------------------------------------------------------
// Config
// -------------------------------------------------------------------------------------

const (
	MaxRetries   = 3
	RetryBackoff = time.Second * 5
)

var (
	//go:embed default.yaml
	defaultConfig []byte

	//go:embed *.tmpl
	templates embed.FS

	// config is the global configuration, it should never be returned by reference.
	config Config

	httpClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
)

type Config struct {
	Switchly Switchly `mapstructure:"switchly"`
	Bifrost  Bifrost  `mapstructure:"bifrost"`
}

// GetSwitchly returns the global switchly configuration.
func GetSwitchly() Switchly {
	return config.Switchly
}

// GetBifrost returns the global switchlynode configuration.
func GetBifrost() Bifrost {
	return config.Bifrost
}

// -------------------------------------------------------------------------------------
// Init
// -------------------------------------------------------------------------------------

// Init should be called at the beginning of execution to load base configuration and
// generate dependent configuration files. The defaults for the config package will be
// loaded from values defined in defaults.yaml in this package, then overridden the
// corresponding environment variables.
func Init() {
	assert := func(err error) {
		if err != nil {
			log.Fatal().Err(err).Msg("failed to bind env")
		}
	}

	// TODO: The following can be cleaned once all deployments are updated to use
	// explicit keys for the new configuration package. In the meantime we will preserve
	// mappings from historical environment for backwards compatibility.
	assert(viper.BindEnv("bifrost.switchly.signer_name", "SIGNER_NAME"))
	assert(viper.BindEnv(
		"bifrost.chains.btc.block_scanner.block_height_discover_back_off",
		"BLOCK_SCANNER_BACKOFF",
	))
	assert(viper.BindEnv(
		"bifrost.chains.doge.block_scanner.block_height_discover_back_off",
		"BLOCK_SCANNER_BACKOFF",
	))
	assert(viper.BindEnv(
		"bifrost.chains.ltc.block_scanner.block_height_discover_back_off",
		"BLOCK_SCANNER_BACKOFF",
	))
	assert(viper.BindEnv(
		"bifrost.chains.bch.block_scanner.block_height_discover_back_off",
		"BLOCK_SCANNER_BACKOFF",
	))
	assert(viper.BindEnv(
		"bifrost.chains.eth.block_scanner.block_height_discover_back_off",
		"BLOCK_SCANNER_BACKOFF",
	))
	assert(viper.BindEnv(
		"bifrost.signer.block_scanner.block_height_discover_back_off",
		"SWITCHLY_BLOCK_TIME",
	))
	assert(viper.BindEnv(
		"switchly.tendermint.consensus.timeout_propose_delta",
		"SWITCHLY_BLOCK_TIME",
	))
	assert(viper.BindEnv(
		"switchly.tendermint.consensus.timeout_prevote_delta",
		"SWITCHLY_BLOCK_TIME",
	))
	assert(viper.BindEnv(
		"switchly.tendermint.consensus.timeout_precommit_delta",
		"SWITCHLY_BLOCK_TIME",
	))
	assert(viper.BindEnv(
		"switchly.tendermint.consensus.timeout_commit",
		"SWITCHLY_BLOCK_TIME",
	))
	assert(viper.BindEnv("bifrost.tss.bootstrap_peers", "PEER"))
	assert(viper.BindEnv("bifrost.tss.external_ip", "EXTERNAL_IP"))
	assert(viper.BindEnv("bifrost.switchly.chain_id", "CHAIN_ID"))
	assert(viper.BindEnv("bifrost.switchly.chain_host", "CHAIN_API"))
	assert(viper.BindEnv("bifrost.switchly.chain_host", "BIFROST_SWITCHLY_CHAIN_HOST"))
	assert(viper.BindEnv(
		"bifrost.switchly.chain_rpc",
		"CHAIN_RPC",
	))
	assert(viper.BindEnv(
		"bifrost.switchly.chain_rpc",
		"BIFROST_SWITCHLY_CHAIN_RPC",
	))
	assert(viper.BindEnv(
		"bifrost.switchly.chain_ebifrost",
		"BIFROST_SWITCHLY_CHAIN_EBIFROST",
	))
	assert(viper.BindEnv(
		"bifrost.chains.BTC.rpc_host",
		"BTC_HOST",
	))
	assert(viper.BindEnv(
		"bifrost.chains.BTC.block_scanner.start_block_height",
		"BTC_START_BLOCK_HEIGHT",
	))
	assert(viper.BindEnv(
		"bifrost.chains.ETH.rpc_host",
		"ETH_HOST",
	))
	assert(viper.BindEnv(
		"bifrost.chains.ETH.block_scanner.start_block_height",
		"ETH_START_BLOCK_HEIGHT",
	))
	assert(viper.BindEnv(
		"bifrost.chains.AVAX.rpc_host",
		"AVAX_HOST",
	))
	assert(viper.BindEnv(
		"bifrost.chains.AVAX.block_scanner.start_block_height",
		"AVAX_START_BLOCK_HEIGHT",
	))
	assert(viper.BindEnv(
		"bifrost.chains.DOGE.rpc_host",
		"DOGE_HOST",
	))
	assert(viper.BindEnv(
		"bifrost.chains.DOGE.block_scanner.start_block_height",
		"DOGE_START_BLOCK_HEIGHT",
	))
	assert(viper.BindEnv(
		"bifrost.chains.GAIA.rpc_host",
		"GAIA_HOST",
	))
	assert(viper.BindEnv(
		"bifrost.chains.GAIA.block_scanner.start_block_height",
		"GAIA_START_BLOCK_HEIGHT",
	))
	assert(viper.BindEnv(
		"bifrost.chains.LTC.rpc_host",
		"LTC_HOST",
	))
	assert(viper.BindEnv(
		"bifrost.chains.LTC.block_scanner.start_block_height",
		"LTC_START_BLOCK_HEIGHT",
	))
	assert(viper.BindEnv(
		"bifrost.chains.BCH.rpc_host",
		"BCH_HOST",
	))
	assert(viper.BindEnv(
		"bifrost.chains.BCH.block_scanner.start_block_height",
		"BCH_START_BLOCK_HEIGHT",
	))
	assert(viper.BindEnv(
		"bifrost.chains.XLM.rpc_host",
		"XLM_HOST",
	))
	assert(viper.BindEnv(
		"bifrost.chains.GAIA.cosmos_grpc_host",
		"GAIA_GRPC_HOST",
	))
	assert(viper.BindEnv(
		"bifrost.chains.GAIA.block_scanner.cosmos_grpc_host",
		"GAIA_GRPC_HOST",
	))
	assert(viper.BindEnv(
		"bifrost.chains.GAIA.cosmos_grpc_tls",
		"GAIA_GRPC_TLS",
	))
	assert(viper.BindEnv(
		"bifrost.chains.GAIA.block_scanner.cosmos_grpc_tls",
		"GAIA_GRPC_TLS",
	))
	assert(viper.BindEnv("bifrost.chains.GAIA.disabled", "GAIA_DISABLED"))
	assert(viper.BindEnv("bifrost.chains.DOGE.disabled", "DOGE_DISABLED"))
	assert(viper.BindEnv("bifrost.chains.LTC.disabled", "LTC_DISABLED"))
	assert(viper.BindEnv("bifrost.chains.AVAX.disabled", "AVAX_DISABLED"))
	assert(viper.BindEnv("bifrost.chains.AVAX.block_scanner.gas_cache_size", "AVAX_GAS_CACHE_SIZE"))
	assert(viper.BindEnv("switchly.cosmos.halt_height", "HARDFORK_BLOCK_HEIGHT"))

	// always override from environment
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)

	// load defaults
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(bytes.NewBuffer(defaultConfig)); err != nil {
		log.Fatal().Err(err).Msg("failed to read default config")
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatal().Err(err).Msg("failed to unmarshal config")
	}
	// set back to toml for cometbft config
	viper.SetConfigType("toml")

	// dynamically set rpc listen address
	if config.Switchly.Tendermint.RPC.ListenAddress == "" {
		config.Switchly.Tendermint.RPC.ListenAddress = fmt.Sprintf("tcp://0.0.0.0:%d", rpcPort)
	}
	if config.Switchly.Tendermint.P2P.ListenAddress == "" {
		config.Switchly.Tendermint.P2P.ListenAddress = fmt.Sprintf("tcp://0.0.0.0:%d", p2pPort)
	}
	if config.Switchly.Cosmos.EBifrost.Address == "" {
		config.Switchly.Cosmos.EBifrost.Address = fmt.Sprintf("0.0.0.0:%d", ebifrostPort)
	}
}

func InitBifrost() {
	for _, chain := range config.Bifrost.GetChains() {
		// validate chain configurations
		if err := chain.ChainID.Valid(); err != nil {
			log.Fatal().Err(err).
				Stringer("chain_id", chain.ChainID).
				Msg("chain failed validation")
		}
		if err := chain.BlockScanner.ChainID.Valid(); err != nil {
			log.Fatal().Err(err).
				Stringer("chain_id", chain.BlockScanner.ChainID).
				Msg("chain failed validation")
		}
	}

	// create observer db paths
	for _, chain := range config.Bifrost.GetChains() {
		err := os.MkdirAll(chain.BlockScanner.DBPath, os.ModePerm)
		if err != nil {
			log.Fatal().Err(err).Str("path", chain.BlockScanner.DBPath).
				Msg("failed to create observer db directory")
		}
	}

	// create signer db path
	err := os.MkdirAll(config.Bifrost.Signer.SignerDbPath, os.ModePerm)
	if err != nil {
		log.Fatal().Err(err).Str("path", config.Bifrost.Signer.SignerDbPath).
			Msg("failed to create signer db directory")
	}

	// set signer password explicitly from environment variable
	config.Bifrost.Switchly.SignerPasswd = os.Getenv("SIGNER_PASSWD")

	// set bootstrap peers from seeds endpoint if unset
	if len(config.Bifrost.TSS.BootstrapPeers) == 0 {
		config.Bifrost.TSS.BootstrapPeers = resolveAddrs(getSeedAddrs())
	}
}

func InitSwitchly(ctx context.Context) {
	// Environment variables prefixed with `SWITCHLY` will be read by viper in cosmos-sdk
	// initialization and overwrite configuration we apply in this package.
	for _, env := range os.Environ() {
		envKey := strings.Split(env, "=")[0]
		if strings.HasPrefix(envKey, "SWITCHLY_") {
			log.Warn().Msgf("environment variable %s could overwrite config", env)
		}
	}

	// if auto statesync enable, find latest snapshot height and hash that should exist
	if config.Switchly.AutoStateSync.Enabled {
		switchlyAutoStateSync(ctx)
	}

	// dynamically set seeds
	seedAddrs, tmSeeds := switchlySeeds()
	config.Switchly.Tendermint.P2P.Seeds = strings.Join(tmSeeds, ",")

	// set the Tendermint external address
	if os.Getenv("EXTERNAL_IP") != "" {
		config.Switchly.Tendermint.P2P.ExternalAddress = fmt.Sprintf("%s:%d", os.Getenv("EXTERNAL_IP"), p2pPort)
	}

	// set paths
	home := os.ExpandEnv("$HOME/.switchlynode")
	tendermintPath := filepath.Join(home, "config", "config.toml")
	cosmosPath := filepath.Join(home, "config", "app.toml")

	// template tendermint config into place
	t := template.Must(template.ParseFS(templates, "*.tmpl"))
	tendermintFile, err := os.OpenFile(tendermintPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open config.toml")
	}
	err = t.ExecuteTemplate(tendermintFile, "config.toml.tmpl", config.Switchly.Tendermint)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to render config.toml")
	}

	// template cosmos config into place
	cosmosFile, err := os.OpenFile(cosmosPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open app.toml")
	}
	err = t.ExecuteTemplate(cosmosFile, "app.toml.tmpl", config.Switchly.Cosmos)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to render app.toml")
	}

	// fetch genesis
	if len(seedAddrs) > 0 {
		// Genesis fetching is handled elsewhere
		log.Info().Msg("seeds available for genesis fetch")
	} else {
		log.Warn().Msg("no seeds, skipping genesis fetch")
	}
}

// -------------------------------------------------------------------------------------
// Switchly
// -------------------------------------------------------------------------------------

func switchlySeeds() (seedAddrs, tmSeeds []string) {
	// use environment variable if set
	seeds := os.Getenv("SEEDS")
	if seeds != "" {
		for _, seed := range strings.Split(seeds, ",") {
			seedAddrs = append(seedAddrs, strings.TrimSpace(seed))
		}
	} else {
		seedAddrs = getSeedAddrs()
	}

	// fetch p2p node ids from seeds
	var mu sync.Mutex
	var wg sync.WaitGroup
	for retry := 0; retry < MaxRetries; retry++ {
		for _, seed := range seedAddrs {
			wg.Add(1)
			go func(seed string) {
				defer wg.Done()

				// parse seed address
				seedIP := seed
				if strings.Contains(seed, ":") {
					seedIP = strings.Split(seed, ":")[0]
				}

				// fetch node status
				url := fmt.Sprintf("http://%s:%d/status", seedIP, rpcPort)
				res, err := http.Get(url)
				if err != nil {
					return
				}
				defer res.Body.Close()

				// decode response
				type status struct {
					Result struct {
						NodeInfo struct {
							ID      string `json:"id"`
							Network string `json:"network"`
						} `json:"node_info"`
					} `json:"result"`
				}
				var s status
				dec := json.NewDecoder(res.Body)
				err = dec.Decode(&s)
				if err != nil {
					log.Error().Err(err).Msg("failed to decode node status")
					return
				}

				// skip if the node is not on the same network
				if s.Result.NodeInfo.Network != os.Getenv("CHAIN_ID") {
					log.Error().
						Str("network", s.Result.NodeInfo.Network).
						Str("expected", os.Getenv("CHAIN_ID")).
						Msg("node is not on the same network")
					return
				}

				// update seeds
				mu.Lock()
				tmSeeds = append(tmSeeds, fmt.Sprintf("%s@%s:%d", s.Result.NodeInfo.ID, seedIP, p2pPort))
				mu.Unlock()
			}(seed)
		}
		wg.Wait()

		// retry a few times if we have no seeds
		if len(tmSeeds) > 0 {
			break
		}
		log.Info().Msg("retrying to fetch seeds...")
		time.Sleep(RetryBackoff)
	}

	log.Info().Msgf("found %d p2p seeds", len(tmSeeds))
	return
}

func switchlyAutoStateSync(ctx context.Context) {
	// if we already have a state assume we have a snapshot and skip
	dataDir := os.ExpandEnv("$HOME/.switchlynode/data/state.db")
	if _, err := os.Stat(dataDir); err == nil {
		log.Info().Msg("data directory detected, skipping auto statesync configuration")
		return
	}

	for _, host := range strings.Split(config.Switchly.Tendermint.StateSync.RPCServers, ",") {
		log.Info().Msgf("auto statesync enabled, determining trust height via %s", host)

		client, err := tmhttp.New(host, "")
		if err != nil {
			log.Err(err).Str("host", host).Msg("failed to create tendermint client")
			continue
		}

		// get the height of the expected snapshot
		status, err := client.Status(ctx)
		if err != nil {
			log.Err(err).Str("host", host).Msg("failed to get status")
			continue
		}
		height := status.SyncInfo.LatestBlockHeight - config.Switchly.AutoStateSync.BlockBuffer

		// get the hash of the trust block
		block, err := client.Block(ctx, &height)
		if err != nil {
			log.Err(err).Str("host", host).Int64("height", height).Msg("failed to get block")
			continue
		}
		hash := block.BlockID.Hash.String()

		// set the trusted hash and height in tendermint
		log.Info().Int64("height", height).Str("hash", hash).Msg("setting automatic statesync trust")
		config.Switchly.Tendermint.StateSync.Enable = true
		config.Switchly.Tendermint.StateSync.TrustHeight = height
		config.Switchly.Tendermint.StateSync.TrustHash = hash

		// set the persistent peers in tendermint to the known auto statesync peers
		config.Switchly.Tendermint.P2P.PersistentPeers = strings.Join(config.Switchly.AutoStateSync.Peers, ",")

		// success
		return
	}

	log.Fatal().Msg("failed to determine statesync trust height from any rpc host")
}

type Switchly struct {
	// NodeRelayURL is the URL of the node relay service.
	NodeRelayURL string `mapstructure:"node_relay_url"`

	// VaultPubkeysCutoffBlocks is the max age in blocks for inactive vaults to be
	// included in the vaults pubkeys response. Vaults older than this age will not be
	// observed by bifrost.
	VaultPubkeysCutoffBlocks int64 `mapstructure:"vault_pubkeys_cutoff_blocks"`

	// SeedNodesEndpoint is the full URL to a /switchly/nodes endpoint for finding active
	// validators to seed genesis and peers.
	SeedNodesEndpoint string `mapstructure:"seed_nodes_endpoint"`

	// StagenetAdminAddresses is only leveraged in stagenet builds to allow for running
	// independent stagenet networks with their own admin addresses. This must remain
	// constant in the configuration for the lifetime of the stagenet instance to avoid
	// consensus failure on sync from genesis.
	StagenetAdminAddresses []string `mapstructure:"stagenet_admin_addresses"`

	// Telemetry contains SwitchlyNode-specific telemetry configuration.
	Telemetry struct {
		// SlashPoints enables slash point telemetry. This creates a file in the node home
		// directory with JSON events for all slash increments and decrements. This feature
		// should not be enabled on production nodes.
		SlashPoints bool `mapstructure:"slash_points"`
	} `mapstructure:"telemetry"`

	// LogFilter will drop logs matching the modules and messages when not in debug level.
	LogFilter struct {
		// Modules is a list of modules to filter.
		Modules []string `mapstructure:"modules"`

		// Messages is a list of messages to filter.
		Messages []string `mapstructure:"messages"`
	} `mapstructure:"log_filter"`

	AutoStateSync struct {
		Enabled bool `mapstructure:"enabled"`

		// BlockBuffer is the number of blocks in the past we will automatically reference
		// for the trust state from one of the configured RPC endpoints.
		BlockBuffer int64 `mapstructure:"block_buffer"`

		// Peers will be used to template the persistent peers in the Tendermint P2P config
		// on the first launch. These peers are static and typically provided by benevolent
		// community members, since the statesync snapshot creation is very expensive and
		// cannot be enabled on nodes unless they are willing to fall behind for a few hours
		// while the snapshots create. Once the initial snapshot is recovered, subsequent
		// restarts will unset the fixed persistent peers to free up peer slots on nodes
		// that are known statesync providers.
		Peers []string `mapstructure:"peers"`
	} `mapstructure:"auto_state_sync"`

	// Cosmos contains values used in templating the Cosmos app.toml.
	Cosmos struct {
		Pruning         string `mapstructure:"pruning"`
		HaltHeight      int64  `mapstructure:"halt_height"`
		MinRetainBlocks int64  `mapstructure:"min_retain_blocks"`

		Telemetry struct {
			Enabled                 bool  `mapstructure:"enabled"`
			PrometheusRetentionTime int64 `mapstructure:"prometheus_retention_time"`
		} `mapstructure:"telemetry"`

		API struct {
			Enable            bool   `mapstructure:"enable"`
			EnabledUnsafeCORS bool   `mapstructure:"enabled_unsafe_cors"`
			EnabledSwagger    bool   `mapstructure:"enabled_swagger"`
			Address           string `mapstructure:"address"`
		} `mapstructure:"api"`

		GRPC struct {
			Enable  bool   `mapstructure:"enable"`
			Address string `mapstructure:"address"`
		} `mapstructure:"grpc"`

		EBifrost struct {
			Enable       bool          `mapstructure:"enable"`
			Address      string        `mapstructure:"address"`
			CacheItemTTL time.Duration `mapstructure:"cache_item_ttl"`
		} `mapstructure:"ebifrost"`

		StateSync struct {
			SnapshotInterval   int64 `mapstructure:"snapshot_interval"`
			SnapshotKeepRecent int64 `mapstructure:"snapshot_keep_recent"`
		} `mapstructure:"state_sync"`
	} `mapstructure:"cosmos"`

	// Tendermint contains values used in templating the Tendermint config.toml.
	Tendermint struct {
		Consensus struct {
			TimeoutProposeDelta   time.Duration `mapstructure:"timeout_propose_delta"`
			TimeoutPrevoteDelta   time.Duration `mapstructure:"timeout_prevote_delta"`
			TimeoutPrecommitDelta time.Duration `mapstructure:"timeout_precommit_delta"`
			TimeoutCommit         time.Duration `mapstructure:"timeout_commit"`
		} `mapstructure:"consensus"`

		Log struct {
			Level  string `mapstructure:"level"`
			Format string `mapstructure:"format"`
		} `mapstructure:"log"`

		RPC struct {
			ListenAddress                        string `mapstructure:"listen_address"`
			CORSAllowedOrigin                    string `mapstructure:"cors_allowed_origin"`
			ExperimentalSubscriptionBufferSize   int64  `mapstructure:"experimental_subscription_buffer_size"`
			ExperimentalWebsocketWriteBufferSize int64  `mapstructure:"experimental_websocket_write_buffer_size"`
		} `mapstructure:"rpc"`

		P2P struct {
			ExternalAddress     string `mapstructure:"external_address"`
			ListenAddress       string `mapstructure:"listen_address"`
			PersistentPeers     string `mapstructure:"persistent_peers"`
			AddrBookStrict      bool   `mapstructure:"addr_book_strict"`
			MaxNumInboundPeers  int64  `mapstructure:"max_num_inbound_peers"`
			MaxNumOutboundPeers int64  `mapstructure:"max_num_outbound_peers"`
			AllowDuplicateIP    bool   `mapstructure:"allow_duplicate_ip"`
			Seeds               string `mapstructure:"seeds"`
		} `mapstructure:"p2p"`

		StateSync struct {
			Enable      bool   `mapstructure:"enable"`
			RPCServers  string `mapstructure:"rpc_servers"`
			TrustHeight int64  `mapstructure:"trust_height"`
			TrustHash   string `mapstructure:"trust_hash"`
			TrustPeriod string `mapstructure:"trust_period"`
		} `mapstructure:"state_sync"`

		Instrumentation struct {
			Prometheus bool `mapstructure:"prometheus"`
		} `mapstructure:"instrumentation"`
	} `mapstructure:"tendermint"`
}

// -------------------------------------------------------------------------------------
// Bifrost
// -------------------------------------------------------------------------------------

type Bifrost struct {
	Signer            BifrostSignerConfiguration     `mapstructure:"signer"`
	Switchly          BifrostClientConfiguration     `mapstructure:"switchly"`
	AttestationGossip BifrostAttestationGossipConfig `mapstructure:"attestation_gossip"`
	Metrics           BifrostMetricsConfiguration    `mapstructure:"metrics"`
	Chains            struct {
		AVAX BifrostChainConfiguration `mapstructure:"avax"`
		BCH  BifrostChainConfiguration `mapstructure:"bch"`
		BSC  BifrostChainConfiguration `mapstructure:"bsc"`
		BTC  BifrostChainConfiguration `mapstructure:"btc"`
		DOGE BifrostChainConfiguration `mapstructure:"doge"`
		ETH  BifrostChainConfiguration `mapstructure:"eth"`
		GAIA BifrostChainConfiguration `mapstructure:"gaia"`
		LTC  BifrostChainConfiguration `mapstructure:"ltc"`
		BASE BifrostChainConfiguration `mapstructure:"base"`
		XRP  BifrostChainConfiguration `mapstructure:"xrp"`
		XLM  BifrostChainConfiguration `mapstructure:"xlm"`
	} `mapstructure:"chains"`
	TSS             BifrostTSSConfiguration `mapstructure:"tss"`
	ObserverLevelDB LevelDBOptions          `mapstructure:"observer_leveldb"`
	ObserverWorkers int                     `mapstructure:"observer_workers"`
}

func (b Bifrost) GetChains() map[common.Chain]BifrostChainConfiguration {
	return map[common.Chain]BifrostChainConfiguration{
		common.AVAXChain:    b.Chains.AVAX,
		common.BCHChain:     b.Chains.BCH,
		common.BSCChain:     b.Chains.BSC,
		common.BTCChain:     b.Chains.BTC,
		common.DOGEChain:    b.Chains.DOGE,
		common.ETHChain:     b.Chains.ETH,
		common.GAIAChain:    b.Chains.GAIA,
		common.LTCChain:     b.Chains.LTC,
		common.BASEChain:    b.Chains.BASE,
		common.XRPChain:     b.Chains.XRP,
		common.StellarChain: b.Chains.XLM,
	}
}

// LevelDBOptions are a superset of the options passed to the LevelDB constructor.
type LevelDBOptions struct {
	// FilterBitsPerKey is the number of bits per key for the bloom filter.
	FilterBitsPerKey int `mapstructure:"filter_bits_per_key"`

	// CompactionTableSizeMultiplier is the multiplier for compaction (table size is 2Mb).
	CompactionTableSizeMultiplier float64 `mapstructure:"compaction_table_size_multiplier"`

	// WriteBuffer is the size of the write buffer in bytes. LevelDB default is 4Mb.
	WriteBuffer int `mapstructure:"write_buffer"`

	// BlockCacheCapacity is the size of the block cache in bytes. LevelDB default is 8Mb.
	BlockCacheCapacity int `mapstructure:"block_cache_capacity"`

	// CompactOnInit will trigger a full compaction at init.
	CompactOnInit bool `mapstructure:"compact_on_init"`
}

// Options returns the corresponding LevelDB options for the constructor.
func (b LevelDBOptions) Options() *opt.Options {
	o := &opt.Options{}

	// only override zero values if set
	if b.CompactionTableSizeMultiplier > 0 {
		o.CompactionTableSizeMultiplier = b.CompactionTableSizeMultiplier
	}
	if b.WriteBuffer > 0 {
		o.WriteBuffer = b.WriteBuffer
	}
	if b.BlockCacheCapacity > 0 {
		o.BlockCacheCapacity = b.BlockCacheCapacity
	}
	if b.FilterBitsPerKey > 0 {
		o.Filter = filter.NewBloomFilter(b.FilterBitsPerKey)
	}

	return o
}

type BifrostSignerConfiguration struct {
	BackupKeyshares bool                             `mapstructure:"backup_keyshares"`
	SignerDbPath    string                           `mapstructure:"signer_db_path"`
	BlockScanner    BifrostBlockScannerConfiguration `mapstructure:"block_scanner"`
	RetryInterval   time.Duration                    `mapstructure:"retry_interval"`

	// RescheduleBufferBlocks is the number of blocks before reschedule we will stop
	// attempting to sign and broadcast (for outbounds not in round 7 retry).
	RescheduleBufferBlocks int64          `mapstructure:"reschedule_buffer_blocks"`
	LevelDB                LevelDBOptions `mapstructure:"leveldb"`

	// AutoObserve will automatically submit the observation for outbound transactions once
	// they are signed - regardless of broadcast success.
	AutoObserve bool `mapstructure:"auto_observe"`

	// -------------------- tss timeouts --------------------

	KeygenTimeout   time.Duration `mapstructure:"keygen_timeout"`
	KeysignTimeout  time.Duration `mapstructure:"keysign_timeout"`
	PartyTimeout    time.Duration `mapstructure:"party_timeout"`
	PreParamTimeout time.Duration `mapstructure:"pre_param_timeout"`
}

type BifrostAttestationGossipConfig struct {
	// how often to prune old observed txs and check if late attestations should be sent.
	// should be less than lateObserveTimeout and minTimeBetweenAttestations by at least a factor of 2.
	ObserveReconcileInterval time.Duration `mapstructure:"observe_reconcile_interval"`

	// validators can get credit for observing a tx for up to this amount of time after it is committed, after which it count against a slash penalty.
	LateObserveTimeout time.Duration `mapstructure:"late_observe_timeout"`

	// Prune observed tx attestations after this amount of time, even if they are not yet committed.
	// Gives some time for longer chain halts.
	// If chain halts for longer than this, validators will need to restart their bifrosts to re-share their attestations.
	NonQuorumTimeout time.Duration `mapstructure:"non_quorum_timeout"`

	// minTimeBetweenAttestations is the minimum time between sending batches of attestations for a quorum tx to switchlynode.
	MinTimeBetweenAttestations time.Duration `mapstructure:"min_time_between_attestations"`

	// how many random peers to ask for their attestation state on startup.
	AskPeers int `mapstructure:"ask_peers"`

	// delay before asking peers for their attestation state on startup.
	AskPeersDelay time.Duration `mapstructure:"ask_peers_delay"`

	// how many attestations to batch together before sending to switchlynode.
	MaxBatchSize int64 `mapstructure:"max_batch_size"`

	// how often to send batches of attestations to switchlynode.
	BatchInterval time.Duration `mapstructure:"batch_interval"`

	// how long to wait when sending a single attestation to a peer before giving up.
	PeerTimeout time.Duration `mapstructure:"peer_timeout"`

	// maximum concurrent sends to a single peer
	PeerConcurrentSends int `mapstructure:"peer_concurrent_sends"`

	// maximum concurrent receives from a single peer
	PeerConcurrentReceives int `mapstructure:"peer_concurrent_receives"`
}

type BifrostChainConfiguration struct {
	ChainID             common.Chain                     `mapstructure:"chain_id"`
	ChainHost           string                           `mapstructure:"chain_host"`
	ChainNetwork        string                           `mapstructure:"chain_network"`
	UserName            string                           `mapstructure:"username"`
	Password            string                           `mapstructure:"password"`
	RPCHost             string                           `mapstructure:"rpc_host"`
	CosmosGRPCHost      string                           `mapstructure:"cosmos_grpc_host"`
	CosmosGRPCTLS       bool                             `mapstructure:"cosmos_grpc_tls"`
	HTTPostMode         bool                             `mapstructure:"http_post_mode"` // Bitcoin core only supports HTTP POST mode
	DisableTLS          bool                             `mapstructure:"disable_tls"`    // Bitcoin core does not provide TLS by default
	OptToRetire         bool                             `mapstructure:"opt_to_retire"`  // don't emit support for this chain during keygen process
	ParallelMempoolScan int                              `mapstructure:"parallel_mempool_scan"`
	Disabled            bool                             `mapstructure:"disabled"`
	SolvencyBlocks      int64                            `mapstructure:"solvency_blocks"`
	BlockScanner        BifrostBlockScannerConfiguration `mapstructure:"block_scanner"`

	// MemPoolTxIDCacheSize is the number of transaction ids to cache in memory. This
	// prevents read on LevelDB which may hit disk for every transaction in the mempool in
	// a loop - which causes concern in recent times with growing Bitcoin mempool.
	MempoolTxIDCacheSize int `mapstructure:"mempool_tx_id_cache_size"`

	// ScannerLevelDB is the LevelDB configuration for the block scanner.
	ScannerLevelDB LevelDBOptions `mapstructure:"scanner_leveldb"`

	// MinConfirmations is the minimum number of confirmations to require before an
	// observed inbound transaction is considered valid.
	MinConfirmations uint64 `mapstructure:"min_confirmations"`

	// MaxRPCRetries is the maximum number of retries for RPC requests.
	MaxRPCRetries int `mapstructure:"max_rpc_retries"`

	// MaxPendingNonces is the maximum number of pending nonces to allow before aborting
	// new signing attempts.
	MaxPendingNonces uint64 `mapstructure:"max_pending_nonces"`

	// AuthorizationBearer can be set to configure the RPC client with an API token that
	// will be provided to the backend in an Authorization header.
	AuthorizationBearer string `mapstructure:"authorization_bearer"`

	// EVM contains EVM chain specific configuration.
	EVM struct {
		// MaxGasTipPercentage is the percentage of the max fee to set for the max tip cap on
		// dynamic fee EVM transactions.
		MaxGasTipPercentage int `mapstructure:"max_gas_tip_percentage"`

		// TokenMaxGasMultiplier is a multiplier applied to max gas for outbounds which are
		// not the gas asset. This compensates for variance in gas units when contracts for
		// pool assets use more than the configured MaxGasLimit gas units in transferOut.
		TokenMaxGasMultiplier int64 `mapstructure:"token_max_gas_multiplier"`

		// AggregatorMaxGasMultiplier is a multiplier applied to max gas for outbounds which
		// swap out via an aggregator contract. This compensates for variance in gas units when
		// aggregator swaps outs use more than the configured MaxGasLimit gas units.
		AggregatorMaxGasMultiplier int64 `mapstructure:"aggregator_max_gas_multiplier"`

		// ExtraL1GasFee is an extra 1e8 amount deducted from the scheduled max gas to
		// account for the Ethereum gas fee on Ethereum L2s.
		ExtraL1GasFee int64 `mapstructure:"extra_l1_gas_fee"`
	} `mapstructure:"evm"`

	// UTXO contains UTXO chain specific configuration.
	UTXO struct {
		// BlockCacheCount is the number of blocks to cache in storage.
		BlockCacheCount uint64 `mapstructure:"block_cache_count"`

		// TransactionBatchSize is the number of transactions to batch in a single request.
		// This is used as the limit for one iteration of the mempool check, and for fanout
		// in fetching block transactions for chains that do not yet support verbosity level
		// 2 on getblock (dogecoin).
		TransactionBatchSize int `mapstructure:"transaction_batch_size"`

		// MaxMempoolBatches is the maximum number of batches to fetch from the mempool in
		// a single scanning pass.
		MaxMempoolBatches int `mapstructure:"max_mempool_batches"`

		// NOTE: The following fields must be consistent across all validators. Otherwise,
		// nodes can fail to sign outbounds from asgard since they may build different
		// transactions. They may also be slashed for reporting different fee and solvency.

		// EstimatedAverageTxSize is the estimated average transaction size in bytes.
		EstimatedAverageTxSize uint64 `mapstructure:"estimated_average_tx_size"`

		// DefaultMinRelayFee is the default minimum relay fee in sats.
		DefaultMinRelayFeeSats uint64 `mapstructure:"default_min_relay_fee_sats"`

		// DefaultSatsPerVByte is the default fee rate in sats per vbyte. It is only used as
		// a fallback when local fee information is unavailable.
		DefaultSatsPerVByte int64 `mapstructure:"default_sats_per_vbyte"`

		// MaxSatsPerVByte is the maximum fee rate in sats per vbyte. It is used to cap the
		// fee rate, since overpaid fees may be rejected by the chain daemon.
		MaxSatsPerVByte int64 `mapstructure:"max_sats_per_vbyte"`

		// MinSatsPerVByte is the minimum fee rate in sats per vbyte. It is used floor the
		// fee rate in order to prevent inbound transactions from being stuck in the
		// mempool.
		MinSatsPerVByte int64 `mapstructure:"min_sats_per_vbyte"`

		// MinUTXOConfirmations is the minimum number of confirmations required for a UTXO to
		// be considered for spending from an asgard vault.
		MinUTXOConfirmations int64 `mapstructure:"min_utxo_confirmations"`

		// MaxUTXOsToSpend is the maximum number of UTXOs to spend in a single transaction.
		// This is overridden at runtime by the `MaxUTXOsToSpend` mimir value.
		MaxUTXOsToSpend int64 `mapstructure:"max_utxos_to_spend"`
	} `mapstructure:"utxo"`
}

func (b *BifrostChainConfiguration) Validate() {
	if b.RPCHost == "" {
		log.Fatal().Str("chain", b.ChainID.String()).Msg("rpc host is required")
	}
}

type BifrostBlockScannerConfiguration struct {
	StartBlockHeight           int64         `mapstructure:"start_block_height"`
	BlockScanProcessors        int           `mapstructure:"block_scan_processors"`
	HTTPRequestTimeout         time.Duration `mapstructure:"http_request_timeout"`
	HTTPRequestReadTimeout     time.Duration `mapstructure:"http_request_read_timeout"`
	HTTPRequestWriteTimeout    time.Duration `mapstructure:"http_request_write_timeout"`
	MaxHTTPRequestRetry        int           `mapstructure:"max_http_request_retry"`
	BlockHeightDiscoverBackoff time.Duration `mapstructure:"block_height_discover_back_off"`
	BlockRetryInterval         time.Duration `mapstructure:"block_retry_interval"`
	EnforceBlockHeight         bool          `mapstructure:"enforce_block_height"`
	DBPath                     string        `mapstructure:"db_path"`
	ChainID                    common.Chain  `mapstructure:"chain_id"`

	// ScanBlocks indicates whether mempool transactions should be scanned.
	ScanMemPool bool `mapstructure:"scan_mempool"`

	// The following configuration values apply only to a subset of chains.

	// CosmosGRPCHost is the <host>:<port> of the gRPC endpoint of the Cosmos SDK chain.
	CosmosGRPCHost string `mapstructure:"cosmos_grpc_host"`

	// CosmosGRPCTLS is a boolean value indicating whether the gRPC host is using TLS.
	CosmosGRPCTLS bool `mapstructure:"cosmos_grpc_tls"`

	// GasCacheBlocks is the number of blocks worth of gas price data cached to determine
	// the gas price reported to Switchly.
	GasCacheBlocks int `mapstructure:"gas_cache_blocks"`

	// Concurrency is the number of goroutines used for RPC requests on data within a
	// block - e.g. transactions, receipts, logs, etc. Blocks are processed sequentially.
	Concurrency int64 `mapstructure:"concurrency"`

	// FixedGasRate will force the scanner to only report the defined gas rate.
	FixedGasRate int64 `mapstructure:"fixed_gas_rate"`

	// GasPriceResolution is the resolution of price per gas unit in the base asset of the
	// chain (wei, tavax, uatom, satoshi, etc) and is transitively the floor price. The
	// gas price will be rounded up to the nearest multiple of this value.
	GasPriceResolution int64 `mapstructure:"gas_price_resolution"`

	// ObservationFlexibilityBlocks is the number of blocks behind the current tip we will
	// submit network fee and solvency observations.
	ObservationFlexibilityBlocks int64 `mapstructure:"observation_flexibility_blocks"`

	// MaxGasLimit is the maximum gas allowed for non-aggregator outbounds. This is used
	// as the limit in the estimate gas call, and the estimate gas (lower) is used in the
	// final outbound.
	MaxGasLimit uint64 `mapstructure:"max_gas_limit"`

	// MaxContractTxLogs is the maximum logs allowed for an inbound EVM tx. This is used to prevent
	// bifrost being bogged down during smart contract log parsing down by large txs.
	MaxContractTxLogs int `mapstructure:"max_contract_tx_logs"`

	// WhitelistTokens is the set of whitelisted token addresses. Inbounds for all other
	// tokens are ignored.
	WhitelistTokens []string `mapstructure:"whitelist_tokens"`

	WhitelistCosmosAssets []WhitelistCosmosAsset `mapstructure:"whitelist_cosmos_assets"`

	// MaxResumeBlockLag is the max duration to lag behind the latest current consensus
	// inbound height upon startup. If there is a local scanner position we will start
	// from that height up to this threshold. The local scanner height is compared to the
	// height from the lastblock response, which contains the height of the latest
	// consensus inbound. This is necessary to avoid a race, as there could be a consensus
	// inbound after an outbound which has not reached consensus, causing double spend.
	MaxResumeBlockLag time.Duration `mapstructure:"max_resume_block_lag"`

	// MaxHealthyLag is the max duration to lag behind the latest block before the scanner
	// is considered unhealthy.
	MaxHealthyLag time.Duration `mapstructure:"max_healthy_lag"`

	// TransactionBatchSize is the number of transactions to batch in a single request.
	// This is used as the limit for one iteration of mempool checks and fanout in
	// fetching block transactions.
	//
	// TODO: This is redundant with the UTXO config, but needs to be on this object for
	// EVM chains - use common config when we refactor to consolidate the config object.
	TransactionBatchSize int `mapstructure:"transaction_batch_size"`

	// MaxReorgRescanBlocks is the maximum number of blocks to rescan during a reorg.
	MaxReorgRescanBlocks int64 `mapstructure:"max_reorg_rescan_blocks"`
}

type BifrostClientConfiguration struct {
	ChainID         common.Chain `mapstructure:"chain_id" `
	ChainHost       string       `mapstructure:"chain_host"`
	ChainRPC        string       `mapstructure:"chain_rpc"`
	ChainEBifrost   string       `mapstructure:"chain_ebifrost"`
	ChainHomeFolder string       `mapstructure:"chain_home_folder"`
	SignerName      string       `mapstructure:"signer_name"`
	SignerPasswd    string
}

type BifrostMetricsConfiguration struct {
	Enabled      bool           `mapstructure:"enabled"`
	PprofEnabled bool           `mapstructure:"pprof_enabled"`
	ListenPort   int            `mapstructure:"listen_port"`
	ReadTimeout  time.Duration  `mapstructure:"read_timeout"`
	WriteTimeout time.Duration  `mapstructure:"write_timeout"`
	Chains       []common.Chain `mapstructure:"chains"`
}

type BifrostTSSConfiguration struct {
	BootstrapPeers               []string `mapstructure:"bootstrap_peers"`
	Rendezvous                   string   `mapstructure:"rendezvous"`
	P2PPort                      int      `mapstructure:"p2p_port"`
	InfoAddress                  string   `mapstructure:"info_address"`
	ExternalIP                   string   `mapstructure:"external_ip"`
	MaxKeyshareRecoverScanBlocks int64    `mapstructure:"max_keyshare_recover_scan_blocks"`
}

func (c BifrostTSSConfiguration) GetP2PPort() int {
	return c.P2PPort
}

func (c BifrostTSSConfiguration) GetRendezvous() string {
	return c.Rendezvous
}

func (c BifrostTSSConfiguration) GetExternalIP() string {
	return c.ExternalIP
}

type WhitelistCosmosAsset struct {
	Denom          string `mapstructure:"denom"`
	Decimals       int    `mapstructure:"decimals"`
	SwitchlySymbol string `mapstructure:"symbol"`
}

// GetBootstrapPeers return the internal bootstrap peers in a slice of maddr.Multiaddr
func (c BifrostTSSConfiguration) GetBootstrapPeers() ([]maddr.Multiaddr, error) {
	var addrs []maddr.Multiaddr

	for _, ip := range resolveAddrs(c.BootstrapPeers) {
		if len(ip) == 0 {
			continue
		}

		// fetch the p2pid
		res, err := httpClient.Get(fmt.Sprintf("http://%s:6040/p2pid", ip))
		if err != nil {
			log.Error().Err(err).Msg("failed to get p2p id")
			continue
		}

		// skip peers with a bad response status
		if res.StatusCode != http.StatusOK {
			log.Warn().Msgf("failed to get p2p id, status code: %d", res.StatusCode)
			continue
		}

		// read the response
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Error().Err(err).Msg("failed to read p2p id response")
			continue
		}
		res.Body.Close()

		// format the multiaddr
		peerMultiAddr := fmt.Sprintf("/ip4/%s/tcp/5040/ipfs/%s", ip, string(body))

		addr, err := maddr.NewMultiaddr(peerMultiAddr)
		if err != nil {
			log.Error().Err(err).Str("addr", peerMultiAddr).Msg("failed to parse multiaddr")
			continue
		}
		addrs = append(addrs, addr)
	}

	if len(addrs) == 0 {
		log.Error().Msg("no bootstrap peers found")
		assertBifrostHasSeeds()
	} else {
		log.Info().Interface("peers", addrs).Msg("bootstrap peers")
	}
	return addrs, nil
}

// -------------------------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------------------------

func resolveAddrs(addrs []string) []string {
	resolvedAddrs := []string{}
	for _, addr := range addrs {
		if net.ParseIP(addr) == nil {
			ips, err := net.LookupHost(addr)
			if err != nil {
				log.Warn().Err(err).Msg("failed to resolve address")
			} else {
				resolvedAddrs = append(resolvedAddrs, ips[0]) // just take the first
			}
		} else {
			resolvedAddrs = append(resolvedAddrs, addr)
		}
	}

	return resolvedAddrs
}
