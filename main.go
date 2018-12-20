package main

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func main() {
	c := types.NewContext(nil, abci.Header{}, types.RunTxModeCheck, nil)
	for i:=0; i< 10; i++ {
		go c.WithValue("a", 1)
	}
	fmt.Println(c)
}
