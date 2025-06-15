package thorchain

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/v3/common/cosmos"
	"gitlab.com/thorchain/thornode/v3/x/thorchain/types"
)

type queryServer struct {
	mgr      *Mgrs
	kbs      cosmos.KeybaseStore
	regInit  bool
	txConfig client.TxConfig
}

var _ types.QueryServer = &queryServer{}

func NewQueryServerImpl(mgr *Mgrs, txConfig client.TxConfig, kbs cosmos.KeybaseStore) types.QueryServer {
	return &queryServer{mgr: mgr, txConfig: txConfig, kbs: kbs}
}

func (s *queryServer) unwrapSdkContext(c context.Context) sdk.Context {
	ctx := sdk.UnwrapSDKContext(c)
	if s.regInit {
		return ctx
	}
	initManager(ctx, s.mgr) // NOOP except regtest
	s.regInit = true
	return ctx
}

func checkHeightParam(height string) error {
	if len(height) > 0 {
		_, err := strconv.ParseInt(height, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid height param, %w", err)
		}
	}
	return nil
}

func (s *queryServer) Pool(c context.Context, req *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryPool(ctx, req)
}

func (s *queryServer) Pools(c context.Context, req *types.QueryPoolsRequest) (*types.QueryPoolsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryPools(ctx, req)
}

func (s *queryServer) DerivedPool(c context.Context, req *types.QueryDerivedPoolRequest) (*types.QueryDerivedPoolResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryDerivedPool(ctx, req)
}

func (s *queryServer) DerivedPools(c context.Context, req *types.QueryDerivedPoolsRequest) (*types.QueryDerivedPoolsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryDerivedPools(ctx, req)
}

func (s *queryServer) LiquidityProvider(c context.Context, req *types.QueryLiquidityProviderRequest) (*types.QueryLiquidityProviderResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryLiquidityProvider(ctx, req)
}

func (s *queryServer) LiquidityProviders(c context.Context, req *types.QueryLiquidityProvidersRequest) (*types.QueryLiquidityProvidersResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryLiquidityProviders(ctx, req)
}

func (s *queryServer) Saver(c context.Context, req *types.QuerySaverRequest) (*types.QuerySaverResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.querySaver(ctx, req)
}

func (s *queryServer) Savers(c context.Context, req *types.QuerySaversRequest) (*types.QuerySaversResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.querySavers(ctx, req)
}

func (s *queryServer) Borrower(c context.Context, req *types.QueryBorrowerRequest) (*types.QueryBorrowerResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryBorrower(ctx, req)
}

func (s *queryServer) Borrowers(c context.Context, req *types.QueryBorrowersRequest) (*types.QueryBorrowersResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryBorrowers(ctx, req)
}

func (s *queryServer) TradeUnit(c context.Context, req *types.QueryTradeUnitRequest) (*types.QueryTradeUnitResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTradeUnit(ctx, req)
}

func (s *queryServer) TradeUnits(c context.Context, req *types.QueryTradeUnitsRequest) (*types.QueryTradeUnitsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTradeUnits(ctx, req)
}

func (s *queryServer) TradeAccount(c context.Context, req *types.QueryTradeAccountRequest) (*types.QueryTradeAccountsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTradeAccount(ctx, req)
}

func (s *queryServer) TradeAccounts(c context.Context, req *types.QueryTradeAccountsRequest) (*types.QueryTradeAccountsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTradeAccounts(ctx, req)
}

func (s *queryServer) SecuredAsset(c context.Context, req *types.QuerySecuredAssetRequest) (*types.QuerySecuredAssetResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.querySecuredAsset(ctx, req)
}

func (s *queryServer) SecuredAssets(c context.Context, req *types.QuerySecuredAssetsRequest) (*types.QuerySecuredAssetsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.querySecuredAssets(ctx, req)
}

func (s *queryServer) Node(c context.Context, req *types.QueryNodeRequest) (*types.QueryNodeResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryNode(ctx, req)
}

func (s *queryServer) Nodes(c context.Context, req *types.QueryNodesRequest) (*types.QueryNodesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryNodes(ctx, req)
}

func (s *queryServer) PoolSlip(c context.Context, req *types.QueryPoolSlipRequest) (*types.QueryPoolSlipsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryPoolSlips(ctx, req.Asset)
}

func (s *queryServer) PoolSlips(c context.Context, req *types.QueryPoolSlipsRequest) (*types.QueryPoolSlipsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryPoolSlips(ctx, "")
}

func (s *queryServer) OutboundFee(c context.Context, req *types.QueryOutboundFeeRequest) (*types.QueryOutboundFeesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryOutboundFees(ctx, req.Asset)
}

func (s *queryServer) OutboundFees(c context.Context, req *types.QueryOutboundFeesRequest) (*types.QueryOutboundFeesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryOutboundFees(ctx, "")
}

func (s *queryServer) StreamingSwap(c context.Context, req *types.QueryStreamingSwapRequest) (*types.QueryStreamingSwapResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryStreamingSwap(ctx, req)
}

func (s *queryServer) StreamingSwaps(c context.Context, req *types.QueryStreamingSwapsRequest) (*types.QueryStreamingSwapsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryStreamingSwaps(ctx, req)
}

func (s *queryServer) Ban(c context.Context, req *types.QueryBanRequest) (*types.BanVoter, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryBan(ctx, req)
}

func (s *queryServer) Ragnarok(c context.Context, req *types.QueryRagnarokRequest) (*types.QueryRagnarokResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryRagnarok(ctx, req)
}

func (s *queryServer) RunePool(c context.Context, req *types.QueryRunePoolRequest) (*types.QueryRunePoolResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryRUNEPool(ctx, req)
}

func (s *queryServer) RuneProvider(c context.Context, req *types.QueryRuneProviderRequest) (*types.QueryRuneProviderResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryRUNEProvider(ctx, req)
}

func (s *queryServer) RuneProviders(c context.Context, req *types.QueryRuneProvidersRequest) (*types.QueryRuneProvidersResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryRUNEProviders(ctx, req)
}

func (s *queryServer) MimirValues(c context.Context, req *types.QueryMimirValuesRequest) (*types.QueryMimirValuesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryMimirValues(ctx, req)
}

func (s *queryServer) MimirWithKey(c context.Context, req *types.QueryMimirWithKeyRequest) (*types.QueryMimirWithKeyResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryMimirWithKey(ctx, req)
}

func (s *queryServer) MimirAdminValues(c context.Context, req *types.QueryMimirAdminValuesRequest) (*types.QueryMimirAdminValuesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryMimirAdminValues(ctx, req)
}

func (s *queryServer) MimirNodesAllValues(c context.Context, req *types.QueryMimirNodesAllValuesRequest) (*types.QueryMimirNodesAllValuesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryMimirNodesAllValues(ctx, req)
}

func (s *queryServer) MimirNodesValues(c context.Context, req *types.QueryMimirNodesValuesRequest) (*types.QueryMimirNodesValuesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryMimirNodesValues(ctx, req)
}

func (s *queryServer) MimirNodeValues(c context.Context, req *types.QueryMimirNodeValuesRequest) (*types.QueryMimirNodeValuesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryMimirNodeValues(ctx, req)
}

func (s *queryServer) InboundAddresses(c context.Context, req *types.QueryInboundAddressesRequest) (*types.QueryInboundAddressesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryInboundAddresses(ctx, req)
}

func (s *queryServer) Version(c context.Context, req *types.QueryVersionRequest) (*types.QueryVersionResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryVersion(ctx, req)
}

func (s *queryServer) Thorname(c context.Context, req *types.QueryThornameRequest) (*types.QueryThornameResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTHORName(ctx, req)
}

func (s *queryServer) Invariant(c context.Context, req *types.QueryInvariantRequest) (*types.QueryInvariantResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryInvariant(ctx, req)
}

func (s *queryServer) Invariants(c context.Context, req *types.QueryInvariantsRequest) (*types.QueryInvariantsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryInvariants(ctx, req)
}

func (s *queryServer) Network(c context.Context, req *types.QueryNetworkRequest) (*types.QueryNetworkResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryNetwork(ctx, req)
}

func (s *queryServer) BalanceModule(c context.Context, req *types.QueryBalanceModuleRequest) (*types.QueryBalanceModuleResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryBalanceModule(ctx, req)
}

func (s *queryServer) QuoteSwap(c context.Context, req *types.QueryQuoteSwapRequest) (*types.QueryQuoteSwapResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryQuoteSwap(ctx, req)
}

func (s *queryServer) QuoteSaverDeposit(c context.Context, req *types.QueryQuoteSaverDepositRequest) (*types.QueryQuoteSaverDepositResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryQuoteSaverDeposit(ctx, req)
}

func (s *queryServer) QuoteSaverWithdraw(c context.Context, req *types.QueryQuoteSaverWithdrawRequest) (*types.QueryQuoteSaverWithdrawResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryQuoteSaverWithdraw(ctx, req)
}

func (s *queryServer) QuoteLoanOpen(c context.Context, req *types.QueryQuoteLoanOpenRequest) (*types.QueryQuoteLoanOpenResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryQuoteLoanOpen(ctx, req)
}

func (s *queryServer) QuoteLoanClose(c context.Context, req *types.QueryQuoteLoanCloseRequest) (*types.QueryQuoteLoanCloseResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryQuoteLoanClose(ctx, req)
}

func (s *queryServer) ConstantValues(c context.Context, req *types.QueryConstantValuesRequest) (*types.QueryConstantValuesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryConstantValues(ctx, req)
}

func (s *queryServer) SwapQueue(c context.Context, req *types.QuerySwapQueueRequest) (*types.QuerySwapQueueResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.querySwapQueue(ctx, req)
}

func (s *queryServer) LastBlocks(c context.Context, req *types.QueryLastBlocksRequest) (*types.QueryLastBlocksResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryLastBlockHeights(ctx, "")
}

func (s *queryServer) ChainsLastBlock(c context.Context, req *types.QueryChainsLastBlockRequest) (*types.QueryLastBlocksResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryLastBlockHeights(ctx, req.Chain)
}

func (s *queryServer) Vault(c context.Context, req *types.QueryVaultRequest) (*types.QueryVaultResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryVault(ctx, req)
}

func (s *queryServer) AsgardVaults(c context.Context, req *types.QueryAsgardVaultsRequest) (*types.QueryAsgardVaultsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryAsgardVaults(ctx, req)
}

func (s *queryServer) VaultsPubkeys(c context.Context, req *types.QueryVaultsPubkeysRequest) (*types.QueryVaultsPubkeysResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryVaultsPubkeys(ctx, req)
}

func (s *queryServer) TxStages(c context.Context, req *types.QueryTxStagesRequest) (*types.QueryTxStagesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTxStages(ctx, req)
}

func (s *queryServer) TxStatus(c context.Context, req *types.QueryTxStatusRequest) (*types.QueryTxStatusResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTxStatus(ctx, req)
}

func (s *queryServer) TxVoters(c context.Context, req *types.QueryTxVotersRequest) (*types.QueryObservedTxVoter, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTxVoters(ctx, req)
}

func (s *queryServer) TxVotersOld(c context.Context, req *types.QueryTxVotersRequest) (*types.QueryObservedTxVoter, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTxVoters(ctx, req)
}

func (s *queryServer) Tx(c context.Context, req *types.QueryTxRequest) (*types.QueryTxResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTx(ctx, req)
}

func (s *queryServer) Clout(c context.Context, req *types.QuerySwapperCloutRequest) (*types.SwapperClout, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.querySwapperClout(ctx, req)
}

func (s *queryServer) Queue(c context.Context, req *types.QueryQueueRequest) (*types.QueryQueueResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryQueue(ctx, req)
}

func (s *queryServer) ScheduledOutbound(c context.Context, req *types.QueryScheduledOutboundRequest) (*types.QueryOutboundResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryScheduledOutbound(ctx, req)
}

func (s *queryServer) PendingOutbound(c context.Context, req *types.QueryPendingOutboundRequest) (*types.QueryOutboundResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryPendingOutbound(ctx, req)
}

func (s *queryServer) Block(c context.Context, req *types.QueryBlockRequest) (*types.QueryBlockResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryBlock(ctx, req)
}

func (s *queryServer) TssKeygenMetric(c context.Context, req *types.QueryTssKeygenMetricRequest) (*types.QueryTssKeygenMetricResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTssKeygenMetric(ctx, req)
}

func (s *queryServer) TssMetric(c context.Context, req *types.QueryTssMetricRequest) (*types.QueryTssMetricResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTssMetric(ctx, req)
}

func (s *queryServer) Keysign(c context.Context, req *types.QueryKeysignRequest) (*types.QueryKeysignResponse, error) {
	ctx := s.unwrapSdkContext(c)
	return s.queryKeysign(ctx, req.Height, "")
}

func (s *queryServer) KeysignPubkey(c context.Context, req *types.QueryKeysignPubkeyRequest) (*types.QueryKeysignResponse, error) {
	ctx := s.unwrapSdkContext(c)
	return s.queryKeysign(ctx, req.Height, req.PubKey)
}

func (s *queryServer) Keygen(c context.Context, req *types.QueryKeygenRequest) (*types.QueryKeygenResponse, error) {
	ctx := s.unwrapSdkContext(c)
	return s.queryKeygen(ctx, req)
}

func (s *queryServer) UpgradeProposal(c context.Context, req *types.QueryUpgradeProposalRequest) (*types.QueryUpgradeProposalResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryUpgradeProposal(ctx, req)
}

func (s *queryServer) UpgradeProposals(c context.Context, req *types.QueryUpgradeProposalsRequest) (*types.QueryUpgradeProposalsResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryUpgradeProposals(ctx, req)
}

func (s *queryServer) UpgradeVotes(c context.Context, req *types.QueryUpgradeVotesRequest) (*types.QueryUpgradeVotesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryUpgradeVotes(ctx, req)
}

func (s *queryServer) Export(c context.Context, req *types.QueryExportRequest) (*types.QueryExportResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	contentBz, err := queryExport(ctx, s.mgr)
	if err != nil {
		return nil, err
	}

	return &types.QueryExportResponse{
		Content: contentBz,
	}, nil
}

func (s *queryServer) Account(c context.Context, req *types.QueryAccountRequest) (*types.QueryAccountResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryAccount(ctx, req)
}

func (s *queryServer) Balances(c context.Context, req *types.QueryBalancesRequest) (*types.QueryBalancesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryBalances(ctx, req)
}

func (s *queryServer) TCYStaker(c context.Context, req *types.QueryTCYStakerRequest) (*types.QueryTCYStakerResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTCYStaker(ctx, req)
}

func (s *queryServer) TCYStakers(c context.Context, req *types.QueryTCYStakersRequest) (*types.QueryTCYStakersResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTCYStakers(ctx, req)
}

func (s *queryServer) TCYClaimer(c context.Context, req *types.QueryTCYClaimerRequest) (*types.QueryTCYClaimerResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTCYClaimer(ctx, req)
}

func (s *queryServer) TCYClaimers(c context.Context, req *types.QueryTCYClaimersRequest) (*types.QueryTCYClaimersResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryTCYClaimers(ctx, req)
}

func (s *queryServer) Codes(c context.Context, req *types.QueryCodesRequest) (*types.QueryCodesResponse, error) {
	if err := checkHeightParam(req.Height); err != nil {
		return nil, err
	}
	ctx := s.unwrapSdkContext(c)
	return s.queryCodes(ctx, req)
}
