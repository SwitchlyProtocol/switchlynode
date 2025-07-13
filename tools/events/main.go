package main

import (
	"os"
	"regexp"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/switchlyprotocol/switchlynode/v1/cmd"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
	"github.com/switchlyprotocol/switchlynode/v1/tools/events/pkg/config"
	"github.com/switchlyprotocol/switchlynode/v1/tools/events/pkg/util"
	"github.com/switchlyprotocol/switchlynode/v1/tools/thorscan"
)

////////////////////////////////////////////////////////////////////////////////////////
// Regexes
////////////////////////////////////////////////////////////////////////////////////////

var (
	reMemoMigration = regexp.MustCompile(`MIGRATE:(\d+)`)
	reMemoRagnarok  = regexp.MustCompile(`RAGNAROK:(\d+)`)
	reMemoRefund    = regexp.MustCompile(`REFUND:(.+)`)
)

////////////////////////////////////////////////////////////////////////////////////////
// InitNetwork
////////////////////////////////////////////////////////////////////////////////////////

func InitNetwork() {
	var bech32PrefixAccAddr string
	var bech32PrefixAccPub string
	var bech32PrefixValAddr string
	var bech32PrefixValPub string
	var bech32PrefixConsAddr string
	var bech32PrefixConsPub string

	switch config.Get().Network {
	case "mainnet":
		bech32PrefixAccAddr = "switch"
		bech32PrefixAccPub = "switchpub"
		bech32PrefixValAddr = "switchv"
		bech32PrefixValPub = "switchvpub"
		bech32PrefixConsAddr = "switchc"
		bech32PrefixConsPub = "switchcpub"

	case "stagenet":
		bech32PrefixAccAddr = "sswitch"
		bech32PrefixAccPub = "sswitchpub"
		bech32PrefixValAddr = "sswitchv"
		bech32PrefixValPub = "sswitchvpub"
		bech32PrefixConsAddr = "sswitchc"
		bech32PrefixConsPub = "sswitchcpub"

	case "mocknet":
		bech32PrefixAccAddr = "tswitch"
		bech32PrefixAccPub = "tswitchpub"
		bech32PrefixValAddr = "tswitchv"
		bech32PrefixValPub = "tswitchvpub"
		bech32PrefixConsAddr = "tswitchc"
		bech32PrefixConsPub = "tswitchcpub"

	default:
		log.Fatal().Str("network", config.Get().Network).Msg("unknown network")
	}

	// initialize the bech32 prefixes
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount(bech32PrefixAccAddr, bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(bech32PrefixValAddr, bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(bech32PrefixConsAddr, bech32PrefixConsPub)
	cfg.SetCoinType(cmd.SwitchlyProtocolCoinType)
	cfg.SetPurpose(cmd.SwitchlyProtocolCoinPurpose)
	cfg.Seal()
}

////////////////////////////////////////////////////////////////////////////////////////
// ScanBlock
////////////////////////////////////////////////////////////////////////////////////////

func ScanBlock(block *thorscan.BlockResponse) {
	ScanInfo(block)
	ScanActivity(block)
	ScanSecurity(block)
	ScanLoans(block)
}

////////////////////////////////////////////////////////////////////////////////////////
// Main
////////////////////////////////////////////////////////////////////////////////////////

func main() {
	// unix time and JSON logging in the cluster, otherwise make it pretty
	if _, err := os.Stat("/run/secrets/kubernetes.io"); err == nil {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	log.Logger = log.With().Caller().Logger()

	// initialize
	util.InitCache()
	InitNetwork()
	thorscan.APIEndpoint = config.Get().Endpoints.Thornode

	// prune local storage
	util.Prune("scheduled-outbound")
	util.Prune("seen-inactive-inbound")
	util.Prune("seen-large-unconfirmed-inbound")
	util.Prune("seen-large-streaming-swap")

	// load the last scanned height from storage
	height := -1
	err := util.Load("height", &height)
	if err != nil {
		log.Warn().Err(err).Msg("unable to load height")
	} else {
		log.Info().Int("height", height).Msg("loaded height")
		height++ // start from the next block
	}

	// override with config
	if config.Get().Scan.Start != 0 {
		height = config.Get().Scan.Start
		log.Info().Int("height", height).Msg("overriding start height")
	}

	// if in console mode set log level to error
	if config.Get().Console {
		log.Info().Msg("console mode enabled")
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}

	for block := range thorscan.Scan(height, config.Get().Scan.Stop) {
		// trail by one block to avoid race with downstream midgard use
		var blockTime time.Time
		blockTime, err = time.Parse(time.RFC3339, block.Header.Time)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to parse block time")
		}
		time.Sleep(time.Until(blockTime.Add(constants.ThorchainBlockTime)))

		ScanBlock(block)

		err = util.Store("height", block.Header.Height)
		if err != nil {
			log.Fatal().Err(err).Int64("height", block.Header.Height).Msg("unable to store height")
		}
		log.Info().Int64("height", block.Header.Height).Msg("scanned block")
	}
}
