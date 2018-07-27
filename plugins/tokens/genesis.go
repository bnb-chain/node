package tokens

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/BiJie/BinanceChain/common/types"
)

func DefaultGenesisToken(owner sdk.AccAddress) types.Token {
	return types.NewToken(
		"Binance Chain Native Token",
		"BNB",
		2e16,
		owner,
	)
}
