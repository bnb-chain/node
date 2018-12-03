package tokens

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func DefaultGenesisToken(owner sdk.AccAddress) types.Token {
	return types.NewToken(
		"Binance Chain Native Token",
		types.NativeToken,
		types.NativeTokenTotalSupply,
		owner,
	)
}

func InitGenesis(ctx sdk.Context, tokenMapper store.Mapper, coinKeeper bank.Keeper,
	tokens []types.Token, validators []sdk.AccAddress, transferAmtForEach int64) {
	var nativeTokenOwner sdk.AccAddress
	for _, token := range tokens {
		err := tokenMapper.NewToken(ctx, token)
		if err != nil {
			panic(err)
		}

		_, _, sdkErr := coinKeeper.AddCoins(ctx, token.Owner,
			sdk.Coins{ sdk.NewCoin(token.Symbol, token.TotalSupply.ToInt64())})
		if token.Symbol == types.NativeToken {
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
		outputs[i] = bank.NewOutput(val, sdk.Coins{sdk.NewCoin(types.NativeToken, amtForEach)})
	}

	inputs := []bank.Input{
		bank.NewInput(nativeTokenOwner, sdk.Coins{sdk.NewCoin(types.NativeToken, amtForEach * int64(numValidators))}),
	}
	coinKeeper.InputOutputCoins(ctx, inputs, outputs)
}
