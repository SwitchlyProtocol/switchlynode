package thornode

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.com/thorchain/thornode/v3/common"
	sdk "gitlab.com/thorchain/thornode/v3/common/cosmos"
	"gitlab.com/thorchain/thornode/v3/config"
	openapi "gitlab.com/thorchain/thornode/v3/openapi/gen"
)

////////////////////////////////////////////////////////////////////////////////////////
// Init
////////////////////////////////////////////////////////////////////////////////////////

var thornodeURL string

func init() {
	config.Init()
	thornodeURL = config.GetBifrost().Thorchain.ChainHost
	if !strings.HasPrefix(thornodeURL, "http") {
		thornodeURL = "http://" + thornodeURL
	}
}

func BaseURL() string {
	return thornodeURL
}

////////////////////////////////////////////////////////////////////////////////////////
// Exported
////////////////////////////////////////////////////////////////////////////////////////

func GetBalances(addr common.Address) (common.Coins, error) {
	url := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", thornodeURL, addr)
	var balances struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}
	err := Get(url, &balances)
	if err != nil {
		return nil, err
	}

	// convert to common.Coins
	coins := make(common.Coins, 0, len(balances.Balances))
	for _, balance := range balances.Balances {
		var amount uint64
		amount, err = strconv.ParseUint(balance.Amount, 10, 64)
		if err != nil {
			return nil, err
		}
		var asset common.Asset
		asset, err = common.NewAsset(strings.ToUpper(balance.Denom))
		if err != nil {
			return nil, err
		}
		coins = append(coins, common.NewCoin(asset, sdk.NewUint(amount)))
	}

	return coins, nil
}

func GetInboundAddress(chain common.Chain) (address common.Address, router *common.Address, err error) {
	url := fmt.Sprintf("%s/thorchain/inbound_addresses", thornodeURL)
	var inboundAddresses []openapi.InboundAddress
	err = Get(url, &inboundAddresses)
	if err != nil {
		return "", nil, err
	}

	// find address for chain
	for _, inboundAddress := range inboundAddresses {
		if *inboundAddress.Chain == string(chain) {
			if inboundAddress.Router != nil {
				router = new(common.Address)
				*router = common.Address(*inboundAddress.Router)
			}
			return common.Address(*inboundAddress.Address), router, nil
		}
	}

	return "", nil, fmt.Errorf("no inbound address found for chain %s", chain)
}

func GetRouterAddress(chain common.Chain) (common.Address, error) {
	url := fmt.Sprintf("%s/thorchain/inbound_addresses", thornodeURL)
	var inboundAddresses []openapi.InboundAddress
	err := Get(url, &inboundAddresses)
	if err != nil {
		return "", err
	}

	// find address for chain
	for _, inboundAddress := range inboundAddresses {
		if *inboundAddress.Chain == string(chain) {
			return common.Address(*inboundAddress.Router), nil
		}
	}

	return "", fmt.Errorf("no inbound address found for chain %s", chain)
}

func GetLiquidityProviders(asset common.Asset) ([]openapi.LiquidityProvider, error) {
	url := fmt.Sprintf("%s/thorchain/pool/%s/liquidity_providers", thornodeURL, asset.String())
	var liquidityProviders []openapi.LiquidityProvider
	err := Get(url, &liquidityProviders)
	return liquidityProviders, err
}

func GetPools() ([]openapi.Pool, error) {
	url := fmt.Sprintf("%s/thorchain/pools", thornodeURL)
	var pools []openapi.Pool
	err := Get(url, &pools)
	return pools, err
}

func GetVaults() ([]openapi.Vault, error) {
	url := fmt.Sprintf("%s/thorchain/vaults/asgard", thornodeURL)
	var vaults []openapi.Vault
	err := Get(url, &vaults)
	return vaults, err
}

func GetNetwork() (openapi.NetworkResponse, error) {
	url := fmt.Sprintf("%s/thorchain/network", thornodeURL)
	var network openapi.NetworkResponse
	err := Get(url, &network)
	return network, err
}

func GetNodes() ([]openapi.Node, error) {
	url := fmt.Sprintf("%s/thorchain/nodes", thornodeURL)
	var nodes []openapi.Node
	err := Get(url, &nodes)
	return nodes, err
}

func GetPool(asset common.Asset) (openapi.Pool, error) {
	url := fmt.Sprintf("%s/thorchain/pool/%s", thornodeURL, asset.String())
	var pool openapi.Pool
	err := Get(url, &pool)
	return pool, err
}

func GetSwapQuote(from, to common.Asset, amount sdk.Uint) (openapi.QuoteSwapResponse, error) {
	baseURL := fmt.Sprintf("%s/thorchain/quote/swap", thornodeURL)
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return openapi.QuoteSwapResponse{}, err
	}
	params := url.Values{}
	params.Add("from_asset", from.String())
	params.Add("to_asset", to.String())
	params.Add("amount", amount.String())
	parsedURL.RawQuery = params.Encode()
	url := parsedURL.String()

	var quote openapi.QuoteSwapResponse
	err = Get(url, &quote)
	return quote, err
}

func GetTxStages(txid string) (openapi.TxStagesResponse, error) {
	url := fmt.Sprintf("%s/thorchain/tx/stages/%s", thornodeURL, txid)
	var stages openapi.TxStagesResponse
	err := Get(url, &stages)
	return stages, err
}

func GetTxDetails(txid string) (openapi.TxDetailsResponse, error) {
	url := fmt.Sprintf("%s/thorchain/tx/details/%s", thornodeURL, txid)
	var details openapi.TxDetailsResponse
	err := Get(url, &details)
	return details, err
}

func GetMimirs() (map[string]int64, error) {
	url := fmt.Sprintf("%s/thorchain/mimir", thornodeURL)
	var mimirs map[string]int64
	err := Get(url, &mimirs)
	return mimirs, err
}

func GetBlock(height int64) (openapi.BlockResponse, error) {
	url := fmt.Sprintf("%s/thorchain/block", thornodeURL)
	if height > 0 {
		url = fmt.Sprintf("%s?height=%d", url, height)
	}
	var block openapi.BlockResponse
	err := Get(url, &block)
	return block, err
}

func Get(url string, target interface{}) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("(%s) HTTP: %d => %s", url, resp.StatusCode, body)
	}

	// extract error if the request failed
	type ErrorResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	errResp := ErrorResponse{}
	err = json.Unmarshal(buf, &errResp)
	if err == nil && errResp.Code != 0 && errResp.Message != "" {
		return fmt.Errorf("code: %d, message: %s", errResp.Code, errResp.Message)
	}

	// decode response
	return json.Unmarshal(buf, target)
}

////////////////////////////////////////////////////////////////////////////////////////
// Internal
////////////////////////////////////////////////////////////////////////////////////////

var httpClient = &http.Client{
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
	},
	Timeout: 5 * time.Second,
}
