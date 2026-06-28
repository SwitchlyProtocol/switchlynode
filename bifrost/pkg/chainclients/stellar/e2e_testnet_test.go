package stellar

// End-to-end test that drives the REAL outbound code path
// (buildRouterTransferOutTransaction -> SimulateTransaction -> signTransactionWithTSS ->
// submitTransactionViaHorizon) against the live Stellar testnet router contract.
//
// It is gated behind STELLAR_TESTNET_E2E=1 because it hits the network, funds accounts via
// friendbot, and submits a real transaction. Run with:
//
//	STELLAR_TESTNET_E2E=1 go test -tags mocknet -count=1 -v \
//	    -run TestTestnetTransferOutE2E ./bifrost/pkg/chainclients/stellar/

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/rs/zerolog"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"

	stypes "github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/config"
)

// testnetRouterContract is the deployed router used for the live test.
const testnetRouterContract = "CAWZ7WYQBG2ENE7S7PAKPYVWKTOV3FMXMIKSOBBZE3JBJXTQWDGZDRGH"

func genVaultPubKey(t *testing.T) common.PubKey {
	t.Helper()
	priv := secp256k1.GenPrivKey()
	tmp, err := codec.FromTmPubKeyInterface(priv.PubKey())
	if err != nil {
		t.Fatalf("convert pubkey: %v", err)
	}
	bech32, err := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeAccPub, tmp)
	if err != nil {
		t.Fatalf("bech32ify: %v", err)
	}
	pk, err := common.NewPubKey(bech32)
	if err != nil {
		t.Fatalf("new pubkey: %v", err)
	}
	return pk
}

func friendbotFund(t *testing.T, addr string) {
	t.Helper()
	resp, err := http.Get("https://friendbot.stellar.org/?addr=" + addr)
	if err != nil {
		t.Fatalf("friendbot request for %s: %v", addr, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("friendbot funding failed for %s (status %d): %s", addr, resp.StatusCode, string(body))
	}
	t.Logf("funded %s", addr)
}

func TestTestnetTransferOutE2E(t *testing.T) {
	if os.Getenv("STELLAR_TESTNET_E2E") == "" {
		t.Skip("set STELLAR_TESTNET_E2E=1 to run the live testnet router E2E test")
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen}).With().Timestamp().Logger()

	// A vault whose Stellar address is derived from its pubkey; the signing key derives from the
	// same pubkey, so funding this address lets our signer authorize as the source account.
	vaultPubKey := genVaultPubKey(t)

	c := &Client{
		logger:            logger,
		networkPassphrase: network.TestNetworkPassphrase,
		routerAddress:     testnetRouterContract,
		horizonClient:     &horizonclient.Client{HorizonURL: "https://horizon-testnet.stellar.org"},
		sorobanRPCClient:  NewSorobanRPCClient(config.BifrostChainConfiguration{ChainNetwork: "testnet"}, logger, StellarTestnet),
		vaultLocks:        make(map[string]*sync.Mutex),
	}

	vaultAddr := c.GetAddress(vaultPubKey)
	if vaultAddr == "" {
		t.Fatal("failed to derive vault Stellar address")
	}
	t.Logf("vault pubkey  = %s", vaultPubKey.String())
	t.Logf("vault address = %s", vaultAddr)

	// Recipient: a throwaway funded testnet account.
	recipient, err := keypair.Random()
	if err != nil {
		t.Fatalf("random recipient: %v", err)
	}

	friendbotFund(t, vaultAddr)
	friendbotFund(t, recipient.Address())

	memo := "OUT:56D832CB5365562BC87F8A309CB3D3A518A5D86715C574D6BED791F42F2F9762" // 68 bytes
	txOut := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address(recipient.Address()),
		VaultPubKey: vaultPubKey,
		Coins:       common.Coins{common.NewCoin(common.XLMAsset, cosmos.NewUint(100000000))}, // 1 XLM
		Memo:        memo,
	}

	// Fetch the vault sequence (retry briefly while friendbot funding settles).
	var sequence int64
	for i := 0; i < 5; i++ {
		sequence, err = c.getNextSequence(vaultPubKey)
		if err == nil {
			break
		}
		t.Logf("waiting for vault account to settle: %v", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		t.Fatalf("get sequence: %v", err)
	}

	// THE REAL CODE PATH: build + simulate + sign the router transfer_out.
	signedTx, err := c.buildRouterTransferOutTransaction(txOut, sequence, memo)
	if err != nil {
		t.Fatalf("buildRouterTransferOutTransaction: %v", err)
	}

	hash, err := c.submitTransactionViaHorizon(signedTx)
	if err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	t.Logf("SUCCESS: transfer_out submitted via our outbound pipeline")
	t.Logf("tx hash = %s", hash)
	t.Logf("explorer = https://stellar.expert/explorer/testnet/tx/%s", hash)
	fmt.Printf("E2E_TX_HASH=%s\n", hash)
}
