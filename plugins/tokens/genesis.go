package tokens

import (
	"github.com/BiJie/BinanceChain/common/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultGenesisToken(owner sdk.AccAddress) types.Token {
	return types.NewToken(
		"Binance Chain Native Token",
		"BNB",
		2e16,
		owner,
	)
}
