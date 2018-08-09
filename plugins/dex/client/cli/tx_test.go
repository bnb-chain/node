package commands

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	rpcclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/BiJie/BinanceChain/common"
)

func newCoreContext(seq int64) context.CoreContext {
	// below does not work. tries to `mkdir .` for some reason
	// cctx := context.NewCoreContextFromViper()
	// TODO: necessary to make a CoreContext, maybe revisit
	nodeURI := "tcp://localhost:26657"
	rpc := rpcclient.NewHTTP(nodeURI, "/")
	return context.CoreContext{
		Client:          rpc,
		NodeURI:         nodeURI,
		AccountStore:    cmn.AccountStoreName,
		Sequence:        seq,
		FromAddressName: "me",
	}
}

func TestGenerateOrderId(t *testing.T) {
	cctx := newCoreContext(5)
	viper.SetDefault(client.FlagSequence, "5")

	saddr := "cosmosaccaddr1atcjghcs273lg95p2kcrn509gdyx2h2g83l0mj"
	addr, err := sdk.AccAddressFromBech32(saddr)
	if err != nil {
		panic(err)
	}

	var orderID string
	orderID, cctx, err = generateOrderID(cctx, addr)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "cosmosaccaddr1atcjghcs273lg95p2kcrn509gdyx2h2g83l0mj-5", orderID)
}
