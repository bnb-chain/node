package list

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdkStore "github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/BiJie/BinanceChain/common"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/dex/store"
	dexTypes "github.com/BiJie/BinanceChain/plugins/dex/types"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	tokenStore "github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func MakeCodec() *codec.Codec {
	var cdc = codec.New()

	bank.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	tokens.RegisterWire(cdc)
	types.RegisterWire(cdc)
	gov.RegisterCodec(cdc)

	return cdc
}

func MakeKeepers(cdc *codec.Codec) (ms sdkStore.CommitMultiStore, orderKeeper *order.Keeper, tokenMapper tokenStore.Mapper, govKeeper gov.Keeper) {
	accKey := sdk.NewKVStoreKey("acc")
	pairKey := sdk.NewKVStoreKey("pair")
	tokenKey := sdk.NewKVStoreKey("token")
	paramKey := sdk.NewKVStoreKey("param")
	paramTKey := sdk.NewTransientStoreKey("t_param")
	stakeKey := sdk.NewKVStoreKey("stake")
	stakeTKey := sdk.NewTransientStoreKey("t_stake")
	govKey := sdk.NewKVStoreKey("gov")

	memDB := db.NewMemDB()
	ms = sdkStore.NewCommitMultiStore(memDB)
	ms.MountStoreWithDB(accKey, sdk.StoreTypeIAVL, memDB)
	ms.MountStoreWithDB(pairKey, sdk.StoreTypeIAVL, memDB)
	ms.MountStoreWithDB(tokenKey, sdk.StoreTypeIAVL, memDB)
	ms.MountStoreWithDB(paramKey, sdk.StoreTypeIAVL, memDB)
	ms.MountStoreWithDB(stakeKey, sdk.StoreTypeIAVL, memDB)
	ms.MountStoreWithDB(govKey, sdk.StoreTypeIAVL, memDB)
	ms.LoadLatestVersion()

	accKeeper := auth.NewAccountKeeper(cdc, accKey, types.ProtoAppAccount)
	codespacer := sdk.NewCodespacer()
	pairMapper := store.NewTradingPairMapper(cdc, pairKey)
	orderKeeper = order.NewKeeper(common.DexStoreKey, accKeeper, pairMapper,
		codespacer.RegisterNext(dexTypes.DefaultCodespace), 2, cdc, false)

	tokenMapper = tokenStore.NewMapper(cdc, tokenKey)

	paramsKeeper := params.NewKeeper(cdc, paramKey, paramTKey)
	bankKeeper := bank.NewBaseKeeper(accKeeper)
	stakeKeeper := stake.NewKeeper(
		cdc,
		stakeKey, stakeTKey,
		bankKeeper, paramsKeeper.Subspace(stake.DefaultParamspace),
		stake.DefaultCodespace,
	)
	govKeeper = gov.NewKeeper(cdc, govKey,
		paramsKeeper, paramsKeeper.Subspace(gov.DefaultParamspace),
		bankKeeper,
		stakeKeeper,
		gov.DefaultCodespace)

	return ms, orderKeeper, tokenMapper, govKeeper
}

func getProposal(lowerCase bool) gov.Proposal {
	baseAssetSymbol := "BTC-000"
	if lowerCase {
		baseAssetSymbol = strings.ToLower(baseAssetSymbol)
	}
	listParams := gov.ListTradingPairParams{
		BaseAssetSymbol:  baseAssetSymbol,
		QuoteAssetSymbol: types.NativeTokenSymbol,
		InitPrice:        1000,
		Description:      "list BTC-000/BNB",
		ExpireTime:       time.Date(2018, 11, 27, 0, 0, 0, 0, time.UTC),
	}

	listParamsBz, _ := json.Marshal(listParams)
	proposal := &gov.TextProposal{
		ProposalID:   1,
		Title:        "list BTC-000/BNB",
		Description:  string(listParamsBz),
		ProposalType: gov.ProposalTypeListTradingPair,
		Status:       gov.StatusDepositPeriod,
		TallyResult:  gov.EmptyTallyResult(),
		TotalDeposit: sdk.Coins{},
		SubmitTime:   time.Now(),
	}
	return proposal
}

func TestListHandler(t *testing.T) {
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, govKeeper := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())

	// proposal does not exist
	result := handleList(ctx, orderKeeper, tokenMapper, govKeeper, ListMsg{
		ProposalId: 1,
	})
	require.Contains(t, result.Log, "proposal 1 does not exist")

	proposal := getProposal(false)

	// wrong status
	govKeeper.SetProposal(ctx, proposal)
	result = handleList(ctx, orderKeeper, tokenMapper, govKeeper, ListMsg{
		ProposalId: 1,
	})
	require.Contains(t, result.Log, "proposal status 1 is not not passed")

	// wrong type
	proposal = getProposal(false)
	proposal.SetProposalType(gov.ProposalTypeParameterChange)
	proposal.SetStatus(gov.StatusPassed)
	govKeeper.SetProposal(ctx, proposal)
	result = handleList(ctx, orderKeeper, tokenMapper, govKeeper, ListMsg{
		ProposalId: 1,
	})
	require.Contains(t, result.Log, "proposal type ParameterChange is not equal to ListTradingPair")

	// wrong params
	proposal = getProposal(false)
	proposal.SetStatus(gov.StatusPassed)
	proposal.SetDescription("wrong params")
	govKeeper.SetProposal(ctx, proposal)
	result = handleList(ctx, orderKeeper, tokenMapper, govKeeper, ListMsg{
		ProposalId: 1,
	})
	require.Contains(t, result.Log, "unmarshal list params error")

	// msg not right
	proposal = getProposal(false)
	proposal.SetStatus(gov.StatusPassed)
	govKeeper.SetProposal(ctx, proposal)
	result = handleList(ctx, orderKeeper, tokenMapper, govKeeper, ListMsg{
		ProposalId: 1,
	})
	require.Contains(t, result.Log, "list msg is not identical to proposal")

	// time expired
	proposal = getProposal(false)
	proposal.SetStatus(gov.StatusPassed)
	govKeeper.SetProposal(ctx, proposal)
	expiredTime := time.Date(2018, 11, 28, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockHeader(abci.Header{
		Time: expiredTime,
	})
	result = handleList(ctx, orderKeeper, tokenMapper, govKeeper, ListMsg{
		ProposalId:       1,
		BaseAssetSymbol:  "BTC-000",
		QuoteAssetSymbol: types.NativeTokenSymbol,
		InitPrice:        1000,
	})
	require.Contains(t, result.Log, "list time expired")

	// token not found
	ctx = sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	result = handleList(ctx, orderKeeper, tokenMapper, govKeeper, ListMsg{
		ProposalId:       1,
		BaseAssetSymbol:  "BTC-000",
		QuoteAssetSymbol: types.NativeTokenSymbol,
		InitPrice:        1000,
	})
	require.Contains(t, result.Log, "token(BTC-000) not found")

	tokenMapper.NewToken(ctx, types.Token{
		Name:        "Bitcoin",
		Symbol:      "BTC-000",
		OrigSymbol:  "BTC",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})

	// no quote asset
	result = handleList(ctx, orderKeeper, tokenMapper, govKeeper, ListMsg{
		ProposalId:       1,
		BaseAssetSymbol:  "BTC-000",
		QuoteAssetSymbol: types.NativeTokenSymbol,
		InitPrice:        1000,
	})
	require.Contains(t, result.Log, "only the owner of the token can list the token")

	result = handleList(ctx, orderKeeper, tokenMapper, govKeeper, ListMsg{
		ProposalId:       1,
		BaseAssetSymbol:  "BTC-000",
		QuoteAssetSymbol: types.NativeTokenSymbol,
		InitPrice:        1000,
		From:             sdk.AccAddress("testacc"),
	})
	require.Contains(t, result.Log, "quote token does not exist")

	tokenMapper.NewToken(ctx, types.Token{
		Name:        "Native Token",
		Symbol:      types.NativeTokenSymbol,
		OrigSymbol:  types.NativeTokenSymbol,
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})

	// right case
	result = handleList(ctx, orderKeeper, tokenMapper, govKeeper, ListMsg{
		ProposalId:       1,
		BaseAssetSymbol:  "BTC-000",
		QuoteAssetSymbol: types.NativeTokenSymbol,
		InitPrice:        1000,
		From:             sdk.AccAddress("testacc"),
	})
	require.Equal(t, result.Code, sdk.ABCICodeOK)
}

func TestListHandler_LowerCase(t *testing.T) {
	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, govKeeper := MakeKeepers(cdc)
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	tokenMapper.NewToken(ctx, types.Token{
		Name:        "Bitcoin",
		Symbol:      "BTC-000",
		OrigSymbol:  "BTC",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})

	tokenMapper.NewToken(ctx, types.Token{
		Name:        "Native Token",
		Symbol:      types.NativeTokenSymbol,
		OrigSymbol:  types.NativeTokenSymbol,
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})

	proposal := getProposal(true)
	proposal.SetStatus(gov.StatusPassed)
	govKeeper.SetProposal(ctx, proposal)
	//ctx = sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	listMsg := ListMsg{
		ProposalId:       1,
		BaseAssetSymbol:  "BTC-000",
		QuoteAssetSymbol: types.NativeTokenSymbol,
		InitPrice:        1000,
		From:             sdk.AccAddress("testacc"),
	}
	result := handleList(ctx, orderKeeper, tokenMapper, govKeeper, listMsg)
	require.Equal(t, sdk.ABCICodeOK, result.Code)
}
