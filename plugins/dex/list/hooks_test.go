package list

import (
	"encoding/json"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/types"
	dexTypes "github.com/binance-chain/node/plugins/dex/types"
)

func TestWrongTypeOfProposal(t *testing.T) {
	hooks := NewListHooks(nil, nil)
	proposal := gov.TextProposal{
		ProposalType: gov.ProposalTypeCreateValidator,
		Description:  "nonsense",
	}

	err := hooks.OnProposalSubmitted(sdk.Context{}, &proposal)
	require.Nil(t, err, "err should be nil")
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

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.NotNil(t, err, "err should not be nil")
	require.Contains(t, err.Error(), "trading pair BTC-ABC against native token should exist before listing other trading pairs")

	pair := dexTypes.NewTradingPair(listParams.BaseAssetSymbol, types.NativeTokenSymbol, listParams.InitPrice)
	err = orderKeeper.PairMapper.AddTradingPair(ctx, pair)
	require.Nil(t, err, "add trading pair error")

	err = hooks.OnProposalSubmitted(ctx, &proposal)
	require.NotNil(t, err, "err should not be nil")
	require.Contains(t, err.Error(), "trading pair ETH-ABC against native token should exist before listing other trading pairs")

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
		OrigSymbol:  listParams.BaseAssetSymbol,
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
