package tokens

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"

	miniToken "github.com/binance-chain/node/plugins/minitokens"
	"github.com/binance-chain/node/plugins/tokens/burn"
	"github.com/binance-chain/node/plugins/tokens/freeze"
	"github.com/binance-chain/node/plugins/tokens/issue"
	"github.com/binance-chain/node/plugins/tokens/store"
	"github.com/binance-chain/node/plugins/tokens/swap"
	"github.com/binance-chain/node/plugins/tokens/timelock"
)

func Routes(tokenMapper store.Mapper, miniTokenMapper miniToken.MiniTokenMapper, accKeeper auth.AccountKeeper, keeper bank.Keeper,
	timeLockKeeper timelock.Keeper, swapKeeper swap.Keeper) map[string]sdk.Handler {
	routes := make(map[string]sdk.Handler)
	routes[issue.Route] = issue.NewHandler(tokenMapper, miniTokenMapper, keeper)
	routes[burn.BurnRoute] = burn.NewHandler(tokenMapper, miniTokenMapper, keeper)
	routes[freeze.FreezeRoute] = freeze.NewHandler(tokenMapper, miniTokenMapper, accKeeper, keeper)
	routes[timelock.MsgRoute] = timelock.NewHandler(timeLockKeeper)
	routes[swap.AtomicSwapRoute] = swap.NewHandler(swapKeeper)
	return routes
}
