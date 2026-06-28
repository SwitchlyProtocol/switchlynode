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
	"context"
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
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"

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

	c := newTestnetClient(logger)

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

// newTestnetClient builds a Client wired to the public Stellar testnet endpoints.
func newTestnetClient(logger zerolog.Logger) *Client {
	return &Client{
		logger:            logger,
		networkPassphrase: network.TestNetworkPassphrase,
		routerAddress:     testnetRouterContract,
		horizonClient:     &horizonclient.Client{HorizonURL: "https://horizon-testnet.stellar.org"},
		sorobanRPCClient:  NewSorobanRPCClient(config.BifrostChainConfiguration{ChainNetwork: "testnet"}, logger, StellarTestnet),
		vaultLocks:        make(map[string]*sync.Mutex),
	}
}

// buildDepositOp builds the router deposit(from, vault, asset, amount, memo) invocation,
// with `from` as the source account so its require_auth() is satisfied via source-account auth.
func (c *Client) buildDepositOp(t *testing.T, fromPubKey common.PubKey, vaultAddr, memo string, amount cosmos.Uint) *txnbuild.InvokeHostFunction {
	t.Helper()
	fromAddr := c.GetAddress(fromPubKey)
	fromSc, err := c.getScAddressFromString(fromAddr)
	if err != nil {
		t.Fatalf("from scaddress: %v", err)
	}
	vaultSc, err := c.getScAddressFromString(vaultAddr)
	if err != nil {
		t.Fatalf("vault scaddress: %v", err)
	}
	mapping, found := GetAssetBySwitchlyAsset(common.XLMAsset)
	if !found {
		t.Fatal("XLM asset mapping not found")
	}
	assetSc, err := c.assetContractScAddress(mapping)
	if err != nil {
		t.Fatalf("asset scaddress: %v", err)
	}
	routerSc, err := c.getScAddressFromString(c.routerAddress)
	if err != nil {
		t.Fatalf("router scaddress: %v", err)
	}
	amountVal, err := scvalI128FromBaseUnits(mapping.ConvertFromSwitchlyProtocolAmount(amount))
	if err != nil {
		t.Fatalf("amount: %v", err)
	}
	memoStr := xdr.ScString(memo)
	args := []xdr.ScVal{
		{Type: xdr.ScValTypeScvAddress, Address: &fromSc},
		{Type: xdr.ScValTypeScvAddress, Address: &vaultSc},
		{Type: xdr.ScValTypeScvAddress, Address: &assetSc},
		amountVal,
		{Type: xdr.ScValTypeScvString, Str: &memoStr},
	}
	return &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.InvokeContractArgs{
				ContractAddress: routerSc,
				FunctionName:    xdr.ScSymbol("deposit"),
				Args:            args,
			},
		},
		SourceAccount: fromAddr,
	}
}

// simulateSignSubmit builds, simulates (footprint + auth), signs and broadcasts an invocation,
// returning the tx hash. Mirrors the production buildRouterTransferOutTransaction pipeline.
func (c *Client) simulateSignSubmit(t *testing.T, op *txnbuild.InvokeHostFunction, sourcePubKey common.PubKey, seq int64) string {
	t.Helper()
	build := func() (*txnbuild.Transaction, error) {
		return txnbuild.NewTransaction(txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: op.SourceAccount, Sequence: seq},
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{op},
			BaseFee:              txnbuild.MinBaseFee,
			Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewInfiniteTimeout()},
		})
	}
	unsigned, err := build()
	if err != nil {
		t.Fatalf("build unsigned: %v", err)
	}
	xdrStr, err := unsigned.Base64()
	if err != nil {
		t.Fatalf("encode unsigned: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sim, err := c.sorobanRPCClient.SimulateTransaction(ctx, xdrStr)
	if err != nil {
		t.Fatalf("simulate: %v", err)
	}
	var sorobanData xdr.SorobanTransactionData
	if err = xdr.SafeUnmarshalBase64(sim.TransactionData, &sorobanData); err != nil {
		t.Fatalf("decode soroban data: %v", err)
	}
	op.Ext = xdr.TransactionExt{V: 1, SorobanData: &sorobanData}
	if len(sim.Results) > 0 {
		for _, a := range sim.Results[0].Auth {
			var entry xdr.SorobanAuthorizationEntry
			if err = xdr.SafeUnmarshalBase64(a, &entry); err != nil {
				t.Fatalf("decode auth entry: %v", err)
			}
			op.Auth = append(op.Auth, entry)
		}
	}
	prepared, err := build()
	if err != nil {
		t.Fatalf("build prepared: %v", err)
	}
	signed, err := c.signTransactionWithTSS(prepared, sourcePubKey, c.networkPassphrase)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	hash, err := c.submitTransactionViaHorizon(signed)
	if err != nil {
		t.Fatalf("broadcast: %v", err)
	}
	return hash
}

// TestTestnetDepositScanE2E fires a real router deposit on testnet, then drives our actual
// scanner (SorobanRPCClient.GetRouterEvents -> ParseContractEvent) to confirm it parses the
// emitted event into a RouterEvent with the full memo and correct fields.
func TestTestnetDepositScanE2E(t *testing.T) {
	if os.Getenv("STELLAR_TESTNET_E2E") == "" {
		t.Skip("set STELLAR_TESTNET_E2E=1 to run the live testnet deposit scan E2E test")
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen}).With().Timestamp().Logger()
	c := newTestnetClient(logger)

	fromPubKey := genVaultPubKey(t)
	vaultPubKey := genVaultPubKey(t)
	fromAddr := c.GetAddress(fromPubKey)
	vaultAddr := c.GetAddress(vaultPubKey)
	if fromAddr == "" || vaultAddr == "" {
		t.Fatal("failed to derive addresses")
	}
	t.Logf("from  = %s", fromAddr)
	t.Logf("vault = %s", vaultAddr)

	friendbotFund(t, fromAddr)
	friendbotFund(t, vaultAddr) // vault must exist to receive native XLM via the SAC

	memo := "=:ETH.ETH:0x71C7656EC7ab88b098defB751B7401B5f6d8976F:0/1/0" // inbound swap intent, > 28 bytes

	var sequence int64
	var err error
	for i := 0; i < 5; i++ {
		sequence, err = c.getNextSequence(fromPubKey)
		if err == nil {
			break
		}
		t.Logf("waiting for from account to settle: %v", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		t.Fatalf("get sequence: %v", err)
	}

	op := c.buildDepositOp(t, fromPubKey, vaultAddr, memo, cosmos.NewUint(100000000)) // 1 XLM
	hash := c.simulateSignSubmit(t, op, fromPubKey, sequence)
	t.Logf("deposit tx = %s", hash)
	t.Logf("explorer   = https://stellar.expert/explorer/testnet/tx/%s", hash)

	// Resolve the ledger the deposit landed in (retry while Horizon indexes it).
	var ledger uint32
	for i := 0; i < 10; i++ {
		txDetail, derr := c.horizonClient.TransactionDetail(hash)
		if derr == nil {
			ledger = uint32(txDetail.Ledger)
			break
		}
		time.Sleep(2 * time.Second)
	}
	if ledger == 0 {
		t.Fatal("could not resolve deposit tx ledger from Horizon")
	}
	t.Logf("deposit ledger = %d", ledger)

	// Drive the REAL scanner path to fetch + parse the event.
	var found *RouterEvent
	for i := 0; i < 15 && found == nil; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		events, gerr := c.sorobanRPCClient.GetRouterEvents(ctx, ledger, []string{c.routerAddress})
		cancel()
		if gerr != nil {
			t.Logf("getRouterEvents: %v", gerr)
		}
		for _, e := range events {
			if e.Type == "deposit" && e.Memo == memo {
				found = e
				break
			}
		}
		if found == nil {
			time.Sleep(3 * time.Second)
		}
	}
	if found == nil {
		t.Fatal("scanner did not find/parse the deposit event")
	}

	// Assert our scanner parsed the event correctly.
	if found.FromAddress != fromAddr {
		t.Errorf("from: got %s want %s", found.FromAddress, fromAddr)
	}
	if found.ToAddress != vaultAddr {
		t.Errorf("vault (to): got %s want %s", found.ToAddress, vaultAddr)
	}
	if found.Amount != "10000000" {
		t.Errorf("amount: got %s want 10000000 stroops", found.Amount)
	}
	if found.Memo != memo {
		t.Errorf("memo: got %q want %q", found.Memo, memo)
	}
	t.Logf("SUCCESS: scanner parsed deposit -> type=%s from=%s to=%s amount=%s memo=%q",
		found.Type, found.FromAddress, found.ToAddress, found.Amount, found.Memo)
}
