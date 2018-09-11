package ico

import (
	"fmt"

	"github.com/BiJie/BinanceChain/common/types"
)

// TODO: just a template
func EndBlockAsync(ctx types.Context) chan interface{} {
	// scan and clearing.
	finish := make(chan interface{})
	go func() {
		fmt.Println(ctx.BlockHeight())
		finish <- struct{}{}
	}()

	return finish
}
