package bank

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

const (
	MiniTokenSymbolSuffixLen          = 4 // probably enough. if it collides (unlikely) the issuer can just use another tx.
	MiniTokenSymbolMSuffix            = "M"
	MiniTokenMinExecutionAmount int64 = 100000000 // 1 with 8 decimal digits
)

func CheckAndValidateMiniTokenCoins(ctx sdk.Context, am auth.AccountKeeper, addr sdk.AccAddress, coins sdk.Coins) sdk.Error {
	var err sdk.Error
	for _, coin := range coins {
		if isMiniTokenSymbol(coin.Denom) {
			err = validateMiniTokenAmount(ctx, am, addr, coin)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func isMiniTokenSymbol(symbol string) bool {
	parts, err := splitSuffixedTokenSymbol(symbol)
	if err != nil {
		return false
	}
	suffixPart := parts[1]

	return len(suffixPart) == MiniTokenSymbolSuffixLen && strings.HasSuffix(suffixPart, MiniTokenSymbolMSuffix)
}

func validateMiniTokenAmount(ctx sdk.Context, am auth.AccountKeeper, addr sdk.AccAddress, coin sdk.Coin) sdk.Error {
	if MiniTokenMinExecutionAmount <= coin.Amount {
		return nil
	}

	coins := getCoins(ctx, am, addr)
	balance := coins.AmountOf(coin.Denom)
	if balance < coin.Amount {
		return sdk.ErrInsufficientCoins("not enough token to send")
	}

	useAllBalance := balance == coin.Amount

	if !useAllBalance {
		return sdk.ErrInvalidCoins(fmt.Sprintf("transfer amount is too small, the min amount is %d or total account balance",
			MiniTokenMinExecutionAmount))
	}

	return nil
}

func splitSuffixedTokenSymbol(suffixed string) ([]string, error) {

	split := strings.SplitN(suffixed, "-", 2)

	if len(split) != 2 {
		return nil, fmt.Errorf("suffixed token symbol must contain a hyphen ('-')")
	}

	if strings.Contains(split[1], "-") {
		return nil, fmt.Errorf("suffixed token symbol must contain just one hyphen ('-')")
	}

	return split, nil
}
