package ico

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO: just a template
func EndBlockAsync(ctx sdk.Context) chan interface{} {
	// scan and clearing.
	finish := make(chan interface{})
	go func() {
		fmt.Println(ctx.BlockHeight())
		finish <- struct{}{}
	}()

	return finish
}
