package evm

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	cKeys "github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	ethclient "github.com/ethereum/go-ethereum/ethclient"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/blockscanner"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/pkg/chainclients/shared/evm"
	evmtypes "github.com/switchlyprotocol/switchlynode/v1/bifrost/pkg/chainclients/shared/evm/types"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/pubkeymanager"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient"
	stypes "github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient/types"
	"github.com/switchlyprotocol/switchlynode/v1/cmd"
	thorcommon "github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/config"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
	. "gopkg.in/check.v1"
)

const TestGasPriceResolution = 50_000_000_000

const Mainnet = 1

var (
	//go:embed test/deposit_evm_transaction.json
	depositEVMTx []byte
	//go:embed test/deposit_evm_receipt.json
	depositEVMReceipt []byte
	//go:embed test/transfer_out_transaction.json
	transferOutTx []byte
	//go:embed test/transfer_out_receipt.json
	transferOutReceipt []byte
	//go:embed test/deposit_tkn_transaction.json
	depositTknTx []byte
	//go:embed test/deposit_tkn_receipt.json
	depositTknReceipt []byte
	//go:embed test/block_by_number.json
	blockByNumberResp []byte
)

func CreateBlock(height int) (*etypes.Header, error) {
	strHeight := fmt.Sprintf("%x", height)
	blockJson := `{
		"parentHash":"0x8b535592eb3192017a527bbf8e3596da86b3abea51d6257898b2ced9d3a83826",
		"difficulty": "0x31962a3fc82b",
		"extraData": "0x4477617266506f6f6c",
		"gasLimit": "0x47c3d8",
		"gasUsed": "0x0",
		"hash": "0x78bfef68fccd4507f9f4804ba5c65eb2f928ea45b3383ade88aaa720f1209cba",
		"logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		"miner": "0x2a65aca4d5fc5b5c859090a6c34d164135398226",
		"nonce": "0xa5e8fb780cc2cd5e",
		"number": "0x` + strHeight + `",
		"receiptsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
		"sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
		"size": "0x20e",
		"stateRoot": "0xdc6ed0a382e50edfedb6bd296892690eb97eb3fc88fd55088d5ea753c48253dc",
		"timestamp": "0x579f4981",
		"totalDifficulty": "0x25cff06a0d96f4bee",
		"transactionsRoot": "0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b"
	}`
	var header *etypes.Header
	if err := json.Unmarshal([]byte(blockJson), &header); err != nil {
		return nil, err
	}
	return header, nil
}

type BlockScannerTestSuite struct {
	m      *metrics.Metrics
	bridge thorclient.ThorchainBridge
	keys   *thorclient.Keys
}

var _ = Suite(&BlockScannerTestSuite{})

func (s *BlockScannerTestSuite) SetUpSuite(c *C) {
	thorchain.SetupConfigForTest()
	s.m = GetMetricForTest(c)
	c.Assert(s.m, NotNil)
	cfg := config.BifrostClientConfiguration{
		ChainID:         "thorchain",
		ChainHost:       "localhost",
		SignerName:      "bob",
		SignerPasswd:    "password",
		ChainHomeFolder: "",
	}

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	kb := cKeys.NewInMemory(cdc)
	_, _, err := kb.NewMnemonic(cfg.SignerName, cKeys.English, cmd.SwitchlyHDPath, cfg.SignerPasswd, hd.Secp256k1)
	c.Assert(err, IsNil)
	thorKeys := thorclient.NewKeysWithKeybase(kb, cfg.SignerName, cfg.SignerPasswd)
	c.Assert(err, IsNil)
	s.keys = thorKeys
	s.bridge, err = thorclient.NewThorchainBridge(cfg, s.m, thorKeys)
	c.Assert(err, IsNil)
}

func getConfigForTest() config.BifrostBlockScannerConfiguration {
	return config.BifrostBlockScannerConfiguration{
		ChainID:                    thorcommon.AVAXChain,
		StartBlockHeight:           1, // avoids querying thorchain for block height
		BlockScanProcessors:        1,
		HTTPRequestTimeout:         time.Second,
		HTTPRequestReadTimeout:     time.Second * 30,
		HTTPRequestWriteTimeout:    time.Second * 30,
		MaxHTTPRequestRetry:        3,
		BlockHeightDiscoverBackoff: time.Second,
		BlockRetryInterval:         time.Second,
		GasCacheBlocks:             100,
		Concurrency:                1,
		GasPriceResolution:         TestGasPriceResolution, // 50 navax
		TransactionBatchSize:       500,
	}
}

func (s *BlockScannerTestSuite) TestNewBlockScanner(c *C) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		c.Assert(err, IsNil)
		type RPCRequest struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      interface{}     `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		var rpcRequest RPCRequest
		err = json.Unmarshal(body, &rpcRequest)
		c.Assert(err, IsNil)
		if rpcRequest.Method == "eth_chainId" {
			_, err = rw.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x539"}`))
			c.Assert(err, IsNil)
		}
		if rpcRequest.Method == "eth_gasPrice" {
			_, err = rw.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
			c.Assert(err, IsNil)
		}
	}))
	storage, err := blockscanner.NewBlockScannerStorage("", config.LevelDBOptions{})
	c.Assert(err, IsNil)
	ethClient, err := ethclient.Dial(server.URL)
	c.Assert(err, IsNil)
	rpcClient, err := evm.NewEthRPC(ethClient, time.Second, "AVAX")
	c.Assert(err, IsNil)
	pubKeyManager, err := pubkeymanager.NewPubKeyManager(s.bridge, s.m)
	c.Assert(err, IsNil)
	solvencyReporter := func(height int64) error {
		return nil
	}
	bs, err := NewEVMScanner(getConfigForTest(), nil, big.NewInt(int64(Mainnet)), ethClient, rpcClient, s.bridge, s.m, pubKeyManager, solvencyReporter, nil)
	c.Assert(err, NotNil)
	c.Assert(bs, IsNil)

	bs, err = NewEVMScanner(getConfigForTest(), storage, big.NewInt(int64(Mainnet)), ethClient, rpcClient, s.bridge, nil, pubKeyManager, solvencyReporter, nil)
	c.Assert(err, NotNil)
	c.Assert(bs, IsNil)

	bs, err = NewEVMScanner(getConfigForTest(), storage, big.NewInt(int64(Mainnet)), nil, rpcClient, s.bridge, s.m, pubKeyManager, solvencyReporter, nil)
	c.Assert(err, NotNil)
	c.Assert(bs, IsNil)

	bs, err = NewEVMScanner(getConfigForTest(), storage, big.NewInt(int64(Mainnet)), ethClient, rpcClient, s.bridge, s.m, nil, solvencyReporter, nil)
	c.Assert(err, NotNil)
	c.Assert(bs, IsNil)

	bs, err = NewEVMScanner(getConfigForTest(), storage, big.NewInt(int64(Mainnet)), ethClient, rpcClient, s.bridge, s.m, pubKeyManager, solvencyReporter, nil)
	c.Assert(err, IsNil)
	c.Assert(bs, NotNil)
}

func (s *BlockScannerTestSuite) TestProcessBlock(c *C) {
	rpcResults := map[string]string{
		"eth_chainId":  `"0xa868"`,
		"eth_gasPrice": `"0x5d21dba00"`,
		"eth_call":     `"0x52554e45"`,
	}

	// extract result from embedded json
	var resp struct {
		Result json.RawMessage `json:"result"`
	}
	err := json.Unmarshal(depositEVMReceipt, &resp)
	c.Assert(err, IsNil)
	rpcResults["eth_getTransactionReceipt"] = string(resp.Result)
	err = json.Unmarshal(blockByNumberResp, &resp)
	c.Assert(err, IsNil)
	rpcResults["eth_getBlockByNumber"] = string(resp.Result)

	handleRPC := func(body []byte, rw http.ResponseWriter) {
		r := struct {
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
		}{}

		err := json.Unmarshal(body, &r)
		c.Assert(err, IsNil)

		rw.Header().Set("Content-Type", "application/json")
		result := map[string]json.RawMessage{
			"result": json.RawMessage(rpcResults[r.Method]),
		}

		err = json.NewEncoder(rw).Encode(result)
		c.Assert(err, IsNil)
	}
	handleBatchRPC := func(body []byte, rw http.ResponseWriter) {
		r := []struct {
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
			ID     int           `json:"id"`
		}{}

		err := json.Unmarshal(body, &r)
		c.Assert(err, IsNil)

		rw.Header().Set("Content-Type", "application/json")
		result := make([]map[string]json.RawMessage, len(r))
		for i, v := range r {
			result[i] = map[string]json.RawMessage{
				"result": json.RawMessage(rpcResults[v.Method]),
				"id":     json.RawMessage(strconv.Itoa(v.ID)),
			}
		}

		err = json.NewEncoder(rw).Encode(result)
		c.Assert(err, IsNil)
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch {
		case req.RequestURI == thorclient.ChainVersionEndpoint:
			_, err = rw.Write([]byte(`{"current":"` + types.GetCurrentVersion().String() + `"}`))
			c.Assert(err, IsNil)
		case req.RequestURI == thorclient.PubKeysEndpoint:
			httpTestHandler(c, rw, "../../../../test/fixtures/endpoints/vaults/pubKeys.json")
		case req.RequestURI == thorclient.InboundAddressesEndpoint:
			httpTestHandler(c, rw, "../../../../test/fixtures/endpoints/inbound_addresses/inbound_addresses.json")
		case req.RequestURI == thorclient.AsgardVault:
			httpTestHandler(c, rw, "../../../../test/fixtures/endpoints/vaults/asgard.json")
		case strings.HasPrefix(req.RequestURI, thorclient.NodeAccountEndpoint):
			httpTestHandler(c, rw, "../../../../test/fixtures/endpoints/nodeaccount/template.json")
		case strings.HasPrefix(req.RequestURI, thorclient.LastBlockEndpoint):
			httpTestHandler(c, rw, "../../../../test/fixtures/endpoints/lastblock/eth.json")
		case strings.HasPrefix(req.RequestURI, thorclient.AuthAccountEndpoint):
			httpTestHandler(c, rw, "../../../../test/fixtures/endpoints/auth/accounts/template.json")
		default:
			// return -1 for all unset mimirs
			if strings.HasPrefix(req.RequestURI, thorclient.MimirEndpoint+"/key") {
				_, err = rw.Write([]byte(`-1`))
				c.Assert(err, IsNil)
				return
			}

			body, err := io.ReadAll(req.Body)
			c.Assert(err, IsNil)
			defer func() {
				c.Assert(req.Body.Close(), IsNil)
			}()
			if body[0] == '[' {
				handleBatchRPC(body, rw)
			} else {
				handleRPC(body, rw)
			}
		}
	}))
	ethClient, err := ethclient.Dial(server.URL)
	c.Assert(err, IsNil)
	c.Assert(ethClient, NotNil)
	rpcClient, err := evm.NewEthRPC(ethClient, time.Second, "AVAX")
	c.Assert(err, IsNil)
	storage, err := blockscanner.NewBlockScannerStorage("", config.LevelDBOptions{})
	c.Assert(err, IsNil)
	u, err := url.Parse(server.URL)
	c.Assert(err, IsNil)
	bridge, err := thorclient.NewThorchainBridge(config.BifrostClientConfiguration{
		ChainID:         "thorchain",
		ChainHost:       u.Host,
		SignerName:      "bob",
		SignerPasswd:    "password",
		ChainHomeFolder: "",
	}, s.m, s.keys)
	c.Assert(err, IsNil)
	pubKeyMgr, err := pubkeymanager.NewPubKeyManager(bridge, s.m)
	c.Assert(err, IsNil)
	c.Assert(pubKeyMgr.Start(), IsNil)
	defer func() {
		c.Assert(pubKeyMgr.Stop(), IsNil)
	}()

	config := getConfigForTest()
	bs, err := NewEVMScanner(config, storage, big.NewInt(43112), ethClient, rpcClient, bridge, s.m, pubKeyMgr, func(height int64) error {
		return nil
	}, nil)
	c.Assert(err, IsNil)
	c.Assert(bs, NotNil)
	bs.whitelistContracts = append(bs.whitelistContracts, "0x40bcd4dB8889a8Bf0b1391d0c819dcd9627f9d0a")
	txIn, err := bs.FetchTxs(int64(1), int64(1))
	c.Assert(err, IsNil)
	c.Check(len(txIn.TxArray), Equals, 1)
}

func httpTestHandler(c *C, rw http.ResponseWriter, fixture string) {
	var content []byte
	var err error

	switch fixture {
	case "500":
		rw.WriteHeader(http.StatusInternalServerError)
	default:
		content, err = os.ReadFile(fixture)
		if err != nil {
			c.Fatal(err)
		}
	}

	rw.Header().Set("Content-Type", "application/json")
	if _, err = rw.Write(content); err != nil {
		c.Fatal(err)
	}
}

func (s *BlockScannerTestSuite) TestGetTxInItem(c *C) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch {
		case req.RequestURI == thorclient.ChainVersionEndpoint:
			_, err := rw.Write([]byte(`{"current":"` + types.GetCurrentVersion().String() + `"}`))
			c.Assert(err, IsNil)
		case req.RequestURI == thorclient.PubKeysEndpoint:
			httpTestHandler(c, rw, "../../../../test/fixtures/endpoints/vaults/pubKeys.json")
		case req.RequestURI == thorclient.InboundAddressesEndpoint:
			httpTestHandler(c, rw, "../../../../test/fixtures/endpoints/inbound_addresses/inbound_addresses.json")
		case req.RequestURI == thorclient.AsgardVault:
			httpTestHandler(c, rw, "../../../../test/fixtures/endpoints/vaults/asgard.json")
		case strings.HasPrefix(req.RequestURI, thorclient.NodeAccountEndpoint):
			httpTestHandler(c, rw, "../../../../test/fixtures/endpoints/nodeaccount/template.json")
		default:
			// return -1 for all unset mimirs
			if strings.HasPrefix(req.RequestURI, thorclient.MimirEndpoint+"/key") {
				_, err := rw.Write([]byte(`-1`))
				c.Assert(err, IsNil)
				return
			}

			body, err := io.ReadAll(req.Body)
			c.Assert(err, IsNil)
			type RPCRequest struct {
				JSONRPC string          `json:"jsonrpc"`
				ID      interface{}     `json:"id"`
				Method  string          `json:"method"`
				Params  json.RawMessage `json:"params"`
			}
			var rpcRequest RPCRequest
			err = json.Unmarshal(body, &rpcRequest)
			if err != nil {
				return
			}
			if rpcRequest.Method == "eth_chainId" {
				_, err = rw.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0xa868"}`))
				c.Assert(err, IsNil)
			}
			if rpcRequest.Method == "eth_gasPrice" {
				_, err = rw.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
				c.Assert(err, IsNil)
			}
			if rpcRequest.Method == "eth_call" {
				c.Log()
				if string(rpcRequest.Params) == `[{"data":"0x95d89b41", "to":"0x333c3310824b7c685133F2BeDb2CA4b8b4DF633d"},"latest"]` {
					_, err = rw.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x544B4E"}`))
					c.Assert(err, IsNil)
					return
				} else if string(rpcRequest.Params) == `[{"data":"0x313ce567","from":"0x0000000000000000000000000000000000000000","to":"0x333c3310824b7c685133F2BeDb2CA4b8b4DF633d"},"latest"]` {
					_, err = rw.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000012"}`))
					c.Assert(err, IsNil)
					return
				}
				_, err = rw.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x544B4E"}`))
				c.Assert(err, IsNil)
			}
			if rpcRequest.Method == "eth_getTransactionReceipt" {
				switch string(rpcRequest.Params) {
				case `["0xc5df10917683a31c361218577d5e13ee9d7e29f8b92415f337a318942bd2c875"]`:
					_, err = rw.Write(depositEVMReceipt)
					c.Assert(err, IsNil)
					return
				case `["0x08053d250f3897e1e27b29dc97bb71a7f99809a5dfd052117ea335c2ee0f55e5"]`:
					_, err = rw.Write(depositTknReceipt)
					c.Assert(err, IsNil)
					return
				case `["0x1f451e1361a1374d135d3da413391cd0d0510e106488b681bed888f3e141bb04"]`:
					_, err = rw.Write(transferOutReceipt)
					c.Assert(err, IsNil)
					return
				}
				_, err = rw.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{
				"transactionHash":"0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b",
				"transactionIndex":"0x0",
				"blockNumber":"0x1",
				"blockHash":"0x78bfef68fccd4507f9f4804ba5c65eb2f928ea45b3383ade88aaa720f1209cba",
				"cumulativeGasUsed":"0xc350",
				"gasUsed":"0x4dc",
				"logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				"logs":[],
				"status":"0x1"
			}}`))
				c.Assert(err, IsNil)
			}
		}
	}))
	ethClient, err := ethclient.Dial(server.URL)
	c.Assert(err, IsNil)
	c.Assert(ethClient, NotNil)
	rpcClient, err := evm.NewEthRPC(ethClient, time.Second, "AVAX")
	c.Assert(err, IsNil)
	storage, err := blockscanner.NewBlockScannerStorage("", config.LevelDBOptions{})
	c.Assert(err, IsNil)
	c.Assert(storage, NotNil)
	u, err := url.Parse(server.URL)
	c.Assert(err, IsNil)

	cfg := config.BifrostClientConfiguration{
		ChainID:         "thorchain",
		ChainHost:       u.Host,
		SignerName:      "bob",
		SignerPasswd:    "password",
		ChainHomeFolder: "",
	}
	bridge, err := thorclient.NewThorchainBridge(cfg, s.m, s.keys)
	c.Assert(err, IsNil)
	c.Assert(bridge, NotNil)
	pkeyMgr, err := pubkeymanager.NewPubKeyManager(bridge, s.m)
	c.Assert(pkeyMgr.Start(), IsNil)
	defer func() {
		c.Assert(pkeyMgr.Stop(), IsNil)
	}()
	c.Assert(err, IsNil)
	config := getConfigForTest()
	bs, err := NewEVMScanner(config, storage, big.NewInt(int64(Mainnet)), ethClient, rpcClient, bridge, s.m, pkeyMgr, func(height int64) error {
		return nil
	}, nil)
	c.Assert(err, IsNil)
	c.Assert(bs, NotNil)

	// send directly to AVAX address
	encodedTx := `{
		"blockHash":"0x1d59ff54b1eb26b013ce3cb5fc9dab3705b415a67127a003c3e61eb445bb8df2",
		"blockNumber":"0x5daf3b",
		"from":"0xa7d9ddbe1f17865597fbd27ec712455208b6b76d",
		"gas":"0xc350",
		"gasPrice":"0x4a817c800",
		"hash":"0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b",
		"input":"0x68656c6c6f21",
		"nonce":"0x15",
		"to":"0xf02c1c8e6114b1dbe8937a39260b5b0a374432bb",
		"transactionIndex":"0x41",
		"value":"0xf3dbb76162000",
		"v":"0x25",
		"r":"0x1b5e176d927f8e9ab405058b2d2457392da3e20f328b16ddabcebc33eaac5fea",
		"s":"0x4ba69724e8f69de52f0125ad8b3c5c2cef33019bac3249e2c0a2192766d1721c"
	}`
	tx := etypes.NewTransaction(0, common.HexToAddress(evm.NativeTokenAddr), nil, 0, nil, nil)
	err = tx.UnmarshalJSON([]byte(encodedTx))
	c.Assert(err, IsNil)

	txInItem, err := bs.getTxInItem(tx)
	c.Assert(err, IsNil)
	c.Assert(txInItem, NotNil)
	c.Check(txInItem.Sender, Equals, "0xa7d9ddbe1f17865597fbd27ec712455208b6b76d")
	c.Check(txInItem.To, Equals, "0xf02c1c8e6114b1dbe8937a39260b5b0a374432bb")
	c.Check(len(txInItem.Coins), Equals, 1)

	c.Check(txInItem.Coins[0].Asset.String(), Equals, "AVAX.AVAX")
	c.Check(
		txInItem.Coins[0].Amount.Equal(cosmos.NewUint(429000)),
		Equals,
		true,
	)
	c.Check(
		txInItem.Gas[0].Amount.Equal(cosmos.NewUint(2488)), // from GasUsed rather than gas limit
		Equals,
		true,
	)

	bs, err = NewEVMScanner(config, storage, big.NewInt(43112), ethClient, rpcClient, bridge, s.m, pkeyMgr, func(height int64) error {
		return nil
	}, nil)
	c.Assert(err, IsNil)
	c.Assert(bs, NotNil)
	tx = etypes.NewTransaction(0, common.HexToAddress(evm.NativeTokenAddr), nil, 0, nil, nil)
	c.Assert(tx.UnmarshalJSON(depositEVMTx), IsNil)
	txInItem, err = bs.getTxInItem(tx)
	c.Assert(err, IsNil)
	c.Assert(txInItem, NotNil)
	c.Assert(txInItem.Sender, Equals, "0x970e8128ab834e8eac17ab8e3812f010678cf791")
	c.Assert(txInItem.To, Equals, "0x6F2744B3a9eba0C5929AAdc9e81183a48B9221FC")
	c.Assert(txInItem.Memo, Equals, "ADD:AVAX.AVAX:tswitch1k7wjdejuf2s9qltyjtvnx2jdn32y7txxgxddvk")
	c.Assert(txInItem.Tx, Equals, "c5df10917683a31c361218577d5e13ee9d7e29f8b92415f337a318942bd2c875")
	c.Assert(txInItem.Coins[0].Asset.String(), Equals, "AVAX.AVAX")
	c.Assert(txInItem.Coins[0].Amount.Uint64(), Equals, cosmos.NewUint(200000000).Uint64())

	config.WhitelistTokens = append(config.WhitelistTokens, "0x333c3310824b7c685133F2BeDb2CA4b8b4DF633d")

	bs, err = NewEVMScanner(config, storage, big.NewInt(43112), ethClient, rpcClient, bridge, s.m, pkeyMgr, func(height int64) error {
		return nil
	}, nil)
	// whitelist the address for test
	bs.whitelistContracts = append(bs.whitelistContracts, "0x17aB05351fC94a1a67Bf3f56DdbB941aE6c63E25")
	c.Assert(err, IsNil)
	c.Assert(bs, NotNil)

	// smart contract - depositTKN
	tx = &etypes.Transaction{}
	c.Assert(tx.UnmarshalJSON(depositTknTx), IsNil)
	txInItem, err = bs.getTxInItem(tx)
	c.Assert(err, IsNil)
	c.Assert(txInItem, NotNil)
	c.Assert(txInItem.Sender, Equals, "0x970e8128ab834e8eac17ab8e3812f010678cf791")
	c.Assert(txInItem.To, Equals, "0x6F2744B3a9eba0C5929AAdc9e81183a48B9221FC")
	c.Assert(txInItem.Memo, Equals, "ADD:AVAX.TKN-0X333C3310824B7C685133F2BEDB2CA4B8B4DF633D:tswitch1k7wjdejuf2s9qltyjtvnx2jdn32y7txxgxddvk")
	c.Assert(txInItem.Tx, Equals, "08053d250f3897e1e27b29dc97bb71a7f99809a5dfd052117ea335c2ee0f55e5")
	// c.Assert(txInItem.Coins[0].Asset.String(), Equals, "AVAX.TKN-0X333C3310824B7C685133F2BEDB2CA4B8B4DF633D")
	c.Assert(txInItem.Coins[0].Amount.Uint64(), Equals, cosmos.NewUint(100000000).Uint64())

	// smart contract - transferOut
	tx = &etypes.Transaction{}
	c.Assert(tx.UnmarshalJSON(transferOutTx), IsNil)
	txInItem, err = bs.getTxInItem(tx)
	c.Assert(err, IsNil)
	c.Assert(txInItem, NotNil)
	c.Assert(txInItem.Sender, Equals, "0xb8bc698bc9c1ed0df7efc37d7367886602361ee5")
	c.Assert(txInItem.To, Equals, "0x970E8128AB834E8EAC17Ab8E3812F010678CF791")
	c.Assert(txInItem.Memo, Equals, "OUT:4A9DEE79350A69BD76B7CBA261B3CEC06546973DF2EACCEDB67EC98EAF21D861")
	c.Assert(txInItem.Tx, Equals, "1f451e1361a1374d135d3da413391cd0d0510e106488b681bed888f3e141bb04")
	c.Assert(txInItem.Coins[0].Asset.String(), Equals, "AVAX.TKN-0X333C3310824B7C685133F2BEDB2CA4B8B4DF633D")
	c.Assert(txInItem.Coins[0].Amount.Equal(cosmos.NewUint(24310000)), Equals, true)
}

func (s *BlockScannerTestSuite) TestProcessReOrg(c *C) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.RequestURI {
		case thorclient.PubKeysEndpoint:
			httpTestHandler(c, rw, "../../../../test/fixtures/endpoints/vaults/pubKeys.json")
		default:
			body, err := io.ReadAll(req.Body)
			c.Assert(err, IsNil)
			type RPCRequest struct {
				JSONRPC string          `json:"jsonrpc"`
				ID      interface{}     `json:"id"`
				Method  string          `json:"method"`
				Params  json.RawMessage `json:"params"`
			}
			var rpcRequest RPCRequest
			err = json.Unmarshal(body, &rpcRequest)
			c.Assert(err, IsNil)
			if rpcRequest.Method == "eth_getBlockByNumber" {
				_, err = rw.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{
				"parentHash":"0x8b535592eb3192017a527bbf8e3596da86b3abea51d6257898b2ced9d3a83826",
				"difficulty": "0x31962a3fc82b",
				"extraData": "0x4477617266506f6f6c",
				"gasLimit": "0x47c3d8",
				"gasUsed": "0x0",
				"hash": "0x78bfef68fccd4507f9f4804ba5c65eb2f928ea45b3383ade88aaa720f1209cba",
				"logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				"miner": "0x2a65aca4d5fc5b5c859090a6c34d164135398226",
				"nonce": "0xa5e8fb780cc2cd5e",
				"number": "0x0",
				"receiptsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
				"sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
				"size": "0x20e",
				"stateRoot": "0xdc6ed0a382e50edfedb6bd296892690eb97eb3fc88fd55088d5ea753c48253dc",
				"timestamp": "0x579f4981",
				"totalDifficulty": "0x25cff06a0d96f4bee",
				"transactions": [{
					"blockHash":"0x78bfef68fccd4507f9f4804ba5c65eb2f928ea45b3383ade88aaa720f1209cba",
					"blockNumber":"0x1",
					"from":"0xa7d9ddbe1f17865597fbd27ec712455208b6b76d",
					"gas":"0xc350",
					"gasPrice":"0x4a817c800",
					"hash":"0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b",
					"input":"0x68656c6c6f21",
					"nonce":"0x15",
					"to":"0xf02c1c8e6114b1dbe8937a39260b5b0a374432bb",
					"transactionIndex":"0x0",
					"value":"0xf3dbb76162000",
					"v":"0x25",
					"r":"0x1b5e176d927f8e9ab405058b2d2457392da3e20f328b16ddabcebc33eaac5fea",
					"s":"0x4ba69724e8f69de52f0125ad8b3c5c2cef33019bac3249e2c0a2192766d1721c"
				}],
				"transactionsRoot": "0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b",
				"uncles": [
			]}}`))
				c.Assert(err, IsNil)
			}
		}
	}))
	ethClient, err := ethclient.Dial(server.URL)
	c.Assert(err, IsNil)
	c.Assert(ethClient, NotNil)
	storage, err := blockscanner.NewBlockScannerStorage("", config.LevelDBOptions{})
	c.Assert(err, IsNil)
	bridge, err := thorclient.NewThorchainBridge(config.BifrostClientConfiguration{
		ChainID:         "thorchain",
		ChainHost:       server.Listener.Addr().String(),
		SignerName:      "bob",
		SignerPasswd:    "password",
		ChainHomeFolder: "",
	}, s.m, s.keys)
	c.Assert(err, IsNil)
	c.Assert(bridge, NotNil)
	pkeyMgr, err := pubkeymanager.NewPubKeyManager(bridge, s.m)
	c.Assert(err, IsNil)
	c.Assert(pkeyMgr.Start(), IsNil)
	defer func() {
		c.Assert(pkeyMgr.Stop(), IsNil)
	}()
	rpcClient, err := evm.NewEthRPC(ethClient, time.Second, "BSC")
	c.Assert(err, IsNil)
	cfg := getConfigForTest()
	cfg.ChainID = thorcommon.BSCChain // re-org on BSC only
	bs, err := NewEVMScanner(cfg, storage, big.NewInt(int64(Mainnet)), ethClient, rpcClient, s.bridge, s.m, pkeyMgr, func(height int64) error {
		return nil
	}, nil)
	c.Assert(err, IsNil)
	c.Assert(bs, NotNil)
	block, err := CreateBlock(0)
	c.Assert(err, IsNil)
	c.Assert(block, NotNil)
	blockNew, err := CreateBlock(1)
	c.Assert(err, IsNil)
	c.Assert(blockNew, NotNil)
	blockMeta := evmtypes.NewBlockMeta(block, stypes.TxIn{TxArray: []*stypes.TxInItem{{Tx: "0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b"}}})
	blockMeta.Transactions = append(blockMeta.Transactions, evmtypes.TransactionMeta{
		Hash:        "0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b",
		BlockHeight: block.Number.Int64(),
	})
	// add one UTXO which will trigger the re-org process next
	c.Assert(bs.blockMetaAccessor.SaveBlockMeta(0, blockMeta), IsNil)
	bs.globalErrataQueue = make(chan stypes.ErrataBlock, 1)
	reorgedBlocks, err := bs.processReorg(blockNew)
	c.Assert(err, IsNil)
	c.Assert(reorgedBlocks, IsNil)
	// make sure there is errata block in the queue
	c.Assert(bs.globalErrataQueue, HasLen, 1)
	blockMeta, err = bs.blockMetaAccessor.GetBlockMeta(0)
	c.Assert(err, IsNil)
	c.Assert(blockMeta, NotNil)
}

// -------------------------------------------------------------------------------------
// GasPriceV2
// -------------------------------------------------------------------------------------

func (s *BlockScannerTestSuite) TestUpdateGasPrice(c *C) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		c.Assert(err, IsNil)
		type RPCRequest struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      interface{}     `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		var rpcRequest RPCRequest
		err = json.Unmarshal(body, &rpcRequest)
		c.Assert(err, IsNil)
		if rpcRequest.Method == "eth_chainId" {
			_, err = rw.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x539"}`))
			c.Assert(err, IsNil)
		}
		if rpcRequest.Method == "eth_gasPrice" {
			_, err = rw.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
			c.Assert(err, IsNil)
		}
	}))
	storage, err := blockscanner.NewBlockScannerStorage("", config.LevelDBOptions{})
	c.Assert(err, IsNil)
	ethClient, err := ethclient.Dial(server.URL)
	c.Assert(err, IsNil)
	rpcClient, err := evm.NewEthRPC(ethClient, time.Second, "AVAX")
	c.Assert(err, IsNil)
	pubKeyManager, err := pubkeymanager.NewPubKeyManager(s.bridge, s.m)
	c.Assert(err, IsNil)
	solvencyReporter := func(height int64) error {
		return nil
	}
	conf := getConfigForTest()
	bs, err := NewEVMScanner(conf, storage, big.NewInt(int64(Mainnet)), ethClient, rpcClient, s.bridge, s.m, pubKeyManager, solvencyReporter, nil)
	c.Assert(err, IsNil)
	c.Assert(bs, NotNil)

	// almost fill gas cache
	for i := 0; i < 99; i++ {
		bs.updateGasPrice([]*big.Int{
			big.NewInt(1 * TestGasPriceResolution),
			big.NewInt(2 * TestGasPriceResolution),
			big.NewInt(3 * TestGasPriceResolution),
			big.NewInt(4 * TestGasPriceResolution),
			big.NewInt(5 * TestGasPriceResolution),
		})
	}

	// empty blocks should not count
	bs.updateGasPrice([]*big.Int{})
	c.Assert(len(bs.gasCache), Equals, 99)
	c.Assert(bs.gasPrice.Cmp(big.NewInt(0)), Equals, 0)

	// now we should get the median of medians
	bs.updateGasPrice([]*big.Int{
		big.NewInt(1 * TestGasPriceResolution),
		big.NewInt(2 * TestGasPriceResolution),
		big.NewInt(3 * TestGasPriceResolution),
		big.NewInt(4 * TestGasPriceResolution),
		big.NewInt(5 * TestGasPriceResolution),
	})
	c.Assert(len(bs.gasCache), Equals, 100)
	c.Assert(bs.gasPrice.String(), Equals, big.NewInt(3*TestGasPriceResolution).String())

	// add 49 more blocks with 2x the median and we should get the same
	for i := 0; i < 49; i++ {
		bs.updateGasPrice([]*big.Int{
			big.NewInt(2 * TestGasPriceResolution),
			big.NewInt(4 * TestGasPriceResolution),
			big.NewInt(6 * TestGasPriceResolution),
			big.NewInt(8 * TestGasPriceResolution),
			big.NewInt(10 * TestGasPriceResolution),
		})
	}
	c.Assert(len(bs.gasCache), Equals, 100)
	c.Assert(bs.gasPrice.String(), Equals, big.NewInt(3*TestGasPriceResolution).String())

	// after one more block with 2x the median we should get 2x
	bs.updateGasPrice([]*big.Int{
		big.NewInt(2 * TestGasPriceResolution),
		big.NewInt(4 * TestGasPriceResolution),
		big.NewInt(6 * TestGasPriceResolution),
		big.NewInt(8 * TestGasPriceResolution),
		big.NewInt(10 * TestGasPriceResolution),
	})
	c.Assert(bs.gasPrice.String(), Equals, big.NewInt(6*TestGasPriceResolution).String())

	// add 50 more blocks with half the median and we should get the same
	for i := 0; i < 50; i++ {
		bs.updateGasPrice([]*big.Int{
			big.NewInt(TestGasPriceResolution),
		})
	}
	c.Assert(len(bs.gasCache), Equals, 100)
	c.Assert(bs.gasPrice.String(), Equals, big.NewInt(6*TestGasPriceResolution).String())

	// after one more block with half the median we should get half
	bs.updateGasPrice([]*big.Int{
		big.NewInt(TestGasPriceResolution),
	})
	c.Assert(bs.gasPrice.String(), Equals, big.NewInt(TestGasPriceResolution).String())
}
