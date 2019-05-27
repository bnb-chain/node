package list

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"

	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/common/upgrade"
	dexTypes "github.com/binance-chain/node/plugins/dex/types"
)

func TestWrongTypeOfProposal(t *testing.T) {
	hooks := NewListHooks(nil, nil)
	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeCreateValidator,
		Description:  "nonsense",
	}

	require.Panics(t, func() {
		hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	}, "should panic here")
}

func TestUnmarshalError(t *testing.T) {
	hooks := NewListHooks(nil, nil)
	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeListTradingPair,
		Description:  "nonsense",
	}

	err := hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "unmarshal list params error")
}

func TestBaseAssetEmpty(t *testing.T) {
	hooks := NewListHooks(nil, nil)

	listParams := gov.ListTradingPairParams{
		BaseAssetSymbol: "",
	}

	listParamsBz, err := json.Marshal(listParams)
	require.Nil(t, err, "marshal list params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeListTradingPair,
		Description:  string(listParamsBz),
	}

	err = hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "base asset symbol should not be empty")
}

func TestQuoteAssetEmpty(t *testing.T) {
	hooks := NewListHooks(nil, nil)

	listParams := gov.ListTradingPairParams{
		BaseAssetSymbol:  "BNB",
		QuoteAssetSymbol: "",
	}

	listParamsBz, err := json.Marshal(listParams)
	require.Nil(t, err, "marshal list params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeListTradingPair,
		Description:  string(listParamsBz),
	}

	err = hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "quote asset symbol should not be empty")
}

func TestEqualBaseAssetAndQuoteAsset(t *testing.T) {
	hooks := NewListHooks(nil, nil)

	listParams := gov.ListTradingPairParams{
		BaseAssetSymbol:  "BNB",
		QuoteAssetSymbol: "BNB",
	}

	listParamsBz, err := json.Marshal(listParams)
	require.Nil(t, err, "marshal list params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeListTradingPair,
		Description:  string(listParamsBz),
	}

	err = hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "base token and quote token should not be the same")
}

func TestWrongPrice(t *testing.T) {
	hooks := NewListHooks(nil, nil)

	listParams := gov.ListTradingPairParams{
		BaseAssetSymbol:  "BNB",
		QuoteAssetSymbol: "BTC",
		InitPrice:        -1,
	}

	listParamsBz, err := json.Marshal(listParams)
	require.Nil(t, err, "marshal list params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeListTradingPair,
		Description:  string(listParamsBz),
	}

	err = hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "init price should larger than zero")
}

func TestWrongExpireTime(t *testing.T) {
	hooks := NewListHooks(nil, nil)

	listParams := gov.ListTradingPairParams{
		BaseAssetSymbol:  "BNB",
		QuoteAssetSymbol: "BTC",
		InitPrice:        1,
		ExpireTime:       time.Now(),
	}

	listParamsBz, err := json.Marshal(listParams)
	require.Nil(t, err, "marshal list params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeListTradingPair,
		Description:  string(listParamsBz),
	}

	ctx := sdk.NewContext(nil, abci.Header{Time: time.Now().Add(10 * time.Second)}, sdk.RunTxModeDeliver, log.NewNopLogger())
	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "expire time should after now")
}

func TestTradingPairExists(t *testing.T) {
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

	err = tokenMapper.NewToken(ctx, types.Token{
		Name:        "Native Token",
		Symbol:      listParams.BaseAssetSymbol,
		OrigSymbol:  listParams.BaseAssetSymbol,
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = tokenMapper.NewToken(ctx, types.Token{
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

func TestPrerequisiteTradingPair(t *testing.T) {
	listParams := gov.ListTradingPairParams{
		BaseAssetSymbol:  "BTC-ABC",
		QuoteAssetSymbol: "ETH-ABC",
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

	err = tokenMapper.NewToken(ctx, types.Token{
		Name:        "Native Token",
		Symbol:      listParams.BaseAssetSymbol,
		OrigSymbol:  "BTC",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = tokenMapper.NewToken(ctx, types.Token{
		Name:        "Native Token",
		Symbol:      listParams.QuoteAssetSymbol,
		OrigSymbol:  "ETH",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.NotNil(t, err, "err should not be nil")
	require.Contains(t, err.Error(), "token BTC-ABC should be listed against BNB before against ETH-ABC")

	pair := dexTypes.NewTradingPair(listParams.BaseAssetSymbol, types.NativeTokenSymbol, listParams.InitPrice)
	err = orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	require.Nil(t, err, "add trading pair error")

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.NotNil(t, err, "err should not be nil")
	require.Contains(t, err.Error(), "token ETH-ABC should be listed against BNB before listing BTC-ABC against ETH-ABC")

	pair = dexTypes.NewTradingPair(listParams.QuoteAssetSymbol, types.NativeTokenSymbol, listParams.InitPrice)
	err = orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	require.Nil(t, err, "add trading pair error")

	err = tokenMapper.NewToken(ctx, types.Token{
		Name:        "Native Token",
		Symbol:      listParams.BaseAssetSymbol,
		OrigSymbol:  "BTC",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = tokenMapper.NewToken(ctx, types.Token{
		Name:        "Native Token",
		Symbol:      listParams.QuoteAssetSymbol,
		OrigSymbol:  "ETH",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.Nil(t, err, "err should be nil")
}

func TestBaseTokenDoesNotExist(t *testing.T) {
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

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "base token does not exist")
}

func TestQuoteTokenDoesNotExist(t *testing.T) {
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

	err = tokenMapper.NewToken(ctx, types.Token{
		Name:        "Native Token",
		Symbol:      listParams.BaseAssetSymbol,
		OrigSymbol:  "BNB",
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "quote token does not exist")
}

func TestRightProposal(t *testing.T) {
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

	err = tokenMapper.NewToken(ctx, types.Token{
		Name:        "Native Token",
		Symbol:      listParams.BaseAssetSymbol,
		OrigSymbol:  listParams.BaseAssetSymbol,
		TotalSupply: 10000,
		Owner:       sdk.AccAddress("testacc"),
	})
	require.Nil(t, err, "new token error")

	err = tokenMapper.NewToken(ctx, types.Token{
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

func TestDelistWrongTypeOfProposal(t *testing.T) {
	sdk.UpgradeMgr.AddUpgradeHeight(upgrade.BEP6, 1)
	sdk.UpgradeMgr.SetHeight(2)

	hooks := NewDelistHooks(nil)
	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeCreateValidator,
		Description:  "nonsense",
	}

	require.Panics(t, func() {
		hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	}, "should panic here")
}

func TestDelistBeforeUpgrade(t *testing.T) {
	sdk.UpgradeMgr.AddUpgradeHeight(upgrade.BEP6, 2)
	sdk.UpgradeMgr.SetHeight(1)

	hooks := NewDelistHooks(nil)
	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeDelistTradingPair,
		Description:  "nonsense",
	}

	err := hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "proposal type DelistTradingPair is not supported")
}

func TestDelistBaseAssetEmpty(t *testing.T) {
	sdk.UpgradeMgr.AddUpgradeHeight(upgrade.BEP6, 1)
	sdk.UpgradeMgr.SetHeight(2)

	hooks := NewDelistHooks(nil)

	delistParams := gov.DelistTradingPairParams{
		BaseAssetSymbol: "",
	}

	delistParamsBz, err := json.Marshal(delistParams)
	require.Nil(t, err, "marshal delist params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeDelistTradingPair,
		Description:  string(delistParamsBz),
	}

	err = hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "base asset symbol should not be empty")
}

func TestDelistQuoteAssetEmpty(t *testing.T) {
	sdk.UpgradeMgr.AddUpgradeHeight(upgrade.BEP6, 1)
	sdk.UpgradeMgr.SetHeight(2)

	hooks := NewDelistHooks(nil)

	delistParams := gov.DelistTradingPairParams{
		BaseAssetSymbol:  "BNB",
		QuoteAssetSymbol: "",
	}

	delistParamsBz, err := json.Marshal(delistParams)
	require.Nil(t, err, "marshal delist params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeDelistTradingPair,
		Description:  string(delistParamsBz),
	}

	err = hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "quote asset symbol should not be empty")
}

func TestDelistEqualBaseAssetAndQuoteAsset(t *testing.T) {
	sdk.UpgradeMgr.AddUpgradeHeight(upgrade.BEP6, 1)
	sdk.UpgradeMgr.SetHeight(2)

	hooks := NewDelistHooks(nil)

	delistParams := gov.DelistTradingPairParams{
		BaseAssetSymbol:  "BNB",
		QuoteAssetSymbol: "BNB",
	}

	delistParamsBz, err := json.Marshal(delistParams)
	require.Nil(t, err, "marshal delist params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeDelistTradingPair,
		Description:  string(delistParamsBz),
	}

	err = hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "base asset symbol and quote asset symbol should not be the same")
}

func TestDelistEmptyJustification(t *testing.T) {
	sdk.UpgradeMgr.AddUpgradeHeight(upgrade.BEP6, 1)
	sdk.UpgradeMgr.SetHeight(2)

	hooks := NewDelistHooks(nil)

	delistParams := gov.DelistTradingPairParams{
		BaseAssetSymbol:  "BNB",
		QuoteAssetSymbol: "BTC-2BD",
	}

	delistParamsBz, err := json.Marshal(delistParams)
	require.Nil(t, err, "marshal delist params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeDelistTradingPair,
		Description:  string(delistParamsBz),
	}

	err = hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "justification should not be empty")
}

func TestDelistTrueIsDelisted(t *testing.T) {
	sdk.UpgradeMgr.AddUpgradeHeight(upgrade.BEP6, 1)
	sdk.UpgradeMgr.SetHeight(2)

	hooks := NewDelistHooks(nil)

	delistParams := gov.DelistTradingPairParams{
		BaseAssetSymbol:  "BNB",
		QuoteAssetSymbol: "BTC-2BD",
		Justification:    "the reason to delist",
		IsExecuted:       true,
	}

	delistParamsBz, err := json.Marshal(delistParams)
	require.Nil(t, err, "marshal delist params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeDelistTradingPair,
		Description:  string(delistParamsBz),
	}

	err = hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "is_executed should be false")
}

func TestDelistTradingPairDoesNotExist(t *testing.T) {
	sdk.UpgradeMgr.AddUpgradeHeight(upgrade.BEP6, 1)
	sdk.UpgradeMgr.SetHeight(2)

	delistParams := gov.DelistTradingPairParams{
		BaseAssetSymbol:  "BNB",
		QuoteAssetSymbol: "BTC-2BD",
		Justification:    "the reason to delist",
		IsExecuted:       false,
	}

	delistParamsBz, err := json.Marshal(delistParams)
	require.Nil(t, err, "marshal delist params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeDelistTradingPair,
		Description:  string(delistParamsBz),
	}

	cdc := MakeCodec()
	ms, orderKeeper, _, _ := MakeKeepers(cdc)
	hooks := NewDelistHooks(orderKeeper)

	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "trading pair BNB_BTC-2BD does not exist")
}

func TestDelistPrerequisiteTradingPair(t *testing.T) {
	sdk.UpgradeMgr.AddUpgradeHeight(upgrade.BEP6, 1)
	sdk.UpgradeMgr.SetHeight(2)

	ethSymbol := "ETH-2CD"
	btcSymbol := "BTC-2BD"

	delistParams := gov.DelistTradingPairParams{
		BaseAssetSymbol:  ethSymbol,
		QuoteAssetSymbol: types.NativeTokenSymbol,
		Justification:    "the reason to delist",
		IsExecuted:       false,
	}

	delistParamsBz, err := json.Marshal(delistParams)
	require.Nil(t, err, "marshal delist params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeDelistTradingPair,
		Description:  string(delistParamsBz),
	}

	cdc := MakeCodec()
	ms, orderKeeper, _, _ := MakeKeepers(cdc)
	hooks := NewDelistHooks(orderKeeper)

	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())

	pair := dexTypes.NewTradingPair(ethSymbol, btcSymbol, 1000)
	err = orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	require.Nil(t, err, "add trading pair error")

	pair = dexTypes.NewTradingPair(ethSymbol, types.NativeTokenSymbol, 1000)
	err = orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	require.Nil(t, err, "add trading pair error")

	pair = dexTypes.NewTradingPair(btcSymbol, types.NativeTokenSymbol, 1000)
	err = orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	require.Nil(t, err, "add trading pair error")

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.NotNil(t, err, "err should not be nil")

	require.Contains(t, err.Error(), "trading pair ETH-2CD_BTC-2BD should not exist before delisting ETH-2CD_BNB")
}

func TestDelistProperTradingPair(t *testing.T) {
	sdk.UpgradeMgr.AddUpgradeHeight(upgrade.BEP6, 1)
	sdk.UpgradeMgr.SetHeight(2)

	ethSymbol := "ETH-2CD"
	btcSymbol := "BTC-2BD"

	delistParams := gov.DelistTradingPairParams{
		BaseAssetSymbol:  ethSymbol,
		QuoteAssetSymbol: btcSymbol,
		Justification:    "the reason to delist",
		IsExecuted:       false,
	}

	delistParamsBz, err := json.Marshal(delistParams)
	require.Nil(t, err, "marshal delist params error")

	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeDelistTradingPair,
		Description:  string(delistParamsBz),
	}

	cdc := MakeCodec()
	ms, orderKeeper, _, _ := MakeKeepers(cdc)
	hooks := NewDelistHooks(orderKeeper)

	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())

	pair := dexTypes.NewTradingPair(ethSymbol, btcSymbol, 1000)
	err = orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	require.Nil(t, err, "add trading pair error")

	pair = dexTypes.NewTradingPair(ethSymbol, types.NativeTokenSymbol, 1000)
	err = orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	require.Nil(t, err, "add trading pair error")

	pair = dexTypes.NewTradingPair(btcSymbol, types.NativeTokenSymbol, 1000)
	err = orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	require.Nil(t, err, "add trading pair error")

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.Nil(t, err, "err should not be nil")
}
