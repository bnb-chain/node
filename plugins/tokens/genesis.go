package tokens

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

type GenesisToken struct {
	Name        string         `json:"name"`
	Symbol      string         `json:"symbol"`
	TotalSupply int64          `json:"total_supply"`
	Owner       sdk.AccAddress `json:"owner"`
}

func DefaultGenesisToken(owner sdk.AccAddress) GenesisToken {
	token, err := types.NewToken(
		"Binance Chain Native Token",
		types.NativeTokenSymbol,
		types.NativeTokenTotalSupply,
		owner,
	)
	if err != nil {
		panic(err)
	}
	return GenesisToken{
		Name:        token.Name,
		Symbol:      token.Symbol,
		TotalSupply: token.TotalSupply.ToInt64(),
		Owner:       token.Owner,
	}
}

func InitGenesis(ctx sdk.Context, tokenMapper store.Mapper, coinKeeper bank.Keeper,
	geneTokens []GenesisToken, validators []sdk.AccAddress, transferAmtForEach int64) {
	var nativeTokenOwner sdk.AccAddress
	for _, geneToken := range geneTokens {
		token, err := types.NewToken(geneToken.Name, geneToken.Symbol, geneToken.TotalSupply, geneToken.Owner)
		if err != nil {
			panic(err)
		}
		err = tokenMapper.NewToken(ctx, *token)
		if err != nil {
			panic(err)
		}

		_, _, sdkErr := coinKeeper.AddCoins(ctx, token.Owner,
			sdk.Coins{sdk.NewCoin(token.Symbol, token.TotalSupply.ToInt64())})
		if token.Symbol == types.NativeTokenSymbol {
			nativeTokenOwner = token.Owner
		}

		if sdkErr != nil {
			panic(sdkErr)
		}
	}

	transferNativeTokensToValidators(ctx, coinKeeper, nativeTokenOwner, validators, transferAmtForEach)
}

func transferNativeTokensToValidators(ctx sdk.Context, coinKeeper bank.Keeper,
	nativeTokenOwner sdk.AccAddress, validators []sdk.AccAddress, amtForEach int64) {
	numValidators := len(validators)
	outputs := make([]bank.Output, numValidators)
	for i, val := range validators {
		outputs[i] = bank.NewOutput(val, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, amtForEach)})
	}

	inputs := []bank.Input{
		bank.NewInput(nativeTokenOwner, sdk.Coins{sdk.NewCoin(types.NativeTokenSymbol, amtForEach*int64(numValidators))}),
	}
	coinKeeper.InputOutputCoins(ctx, inputs, outputs)
}
