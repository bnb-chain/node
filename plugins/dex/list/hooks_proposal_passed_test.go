package list

import (
	"encoding/json"
	"github.com/binance-chain/node/common/upgrade"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/common/types"
	dexTypes "github.com/binance-chain/node/plugins/dex/types"
)

func prepare() {
	sdk.UpgradeMgr.AddUpgradeHeight(upgrade.ListRefactor, 100)
	sdk.UpgradeMgr.SetHeight(200)
}

func TestIncorrectProposalType(t *testing.T) {
	prepare()
	hooks := NewListHooks(nil, nil)
	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeText,
	}
	err := hooks.OnProposalPassed(sdk.Context{}, &proposal)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "proposal type")
}

func TestIncorrectProposalStatus(t *testing.T) {
	prepare()
	hooks := NewListHooks(nil, nil)
	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeListTradingPair,
		Status:       gov.StatusVotingPeriod,
	}
	err := hooks.OnProposalPassed(sdk.Context{}, &proposal)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "proposal status")
}

func TestDuplicatedTradingPair(t *testing.T) {
	listParams := gov.ListTradingPairParams{
		BaseAssetSymbol:  "BNB",
		QuoteAssetSymbol: "BTC-ABC",
		InitPrice:        1,
		ExpireTime:       time.Now(),
	}

	listParamsBz, err := json.Marshal(listParams)
	require.Nil(t, err, "marshal list params error")
	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeListTradingPair,
		Description:  string(listParamsBz),
	}

	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	hooks := NewListHooks(orderKeeper, tokenMapper)

	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())

	err = tokenMapper.NewToken(ctx, &types.Token{
		Name:        "Native Token",
		Symbol:      listParams.BaseAssetSymbol,
		OrigSymbol:  listParams.BaseAssetSymbol,
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = tokenMapper.NewToken(ctx, &types.Token{
		Name:        "Native Token",
		Symbol:      listParams.QuoteAssetSymbol,
		OrigSymbol:  "BTC",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	pair := dexTypes.NewTradingPair(listParams.BaseAssetSymbol, listParams.QuoteAssetSymbol, listParams.InitPrice)
	err = orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	require.Nil(t, err, "add trading pair error")

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "trading pair exists")
}

func TestPassed(t *testing.T) {
	listParams := gov.ListTradingPairParams{
		BaseAssetSymbol:  "BNB",
		QuoteAssetSymbol: "BTC-ABC",
		InitPrice:        1,
		ExpireTime:       time.Now(),
	}

	listParamsBz, err := json.Marshal(listParams)
	require.Nil(t, err, "marshal list params error")
	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeListTradingPair,
		Status:       gov.StatusPassed,
		Description:  string(listParamsBz),
	}

	cdc := MakeCodec()
	ms, orderKeeper, tokenMapper, _ := MakeKeepers(cdc)
	hooks := NewListHooks(orderKeeper, tokenMapper)

	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())

	err = tokenMapper.NewToken(ctx, &types.Token{
		Name:        "Native Token",
		Symbol:      listParams.BaseAssetSymbol,
		OrigSymbol:  listParams.BaseAssetSymbol,
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = tokenMapper.NewToken(ctx, &types.Token{
		Name:        "Native Token",
		Symbol:      listParams.QuoteAssetSymbol,
		OrigSymbol:  "BTC",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.Nil(t, err, "err should be nil")
}
