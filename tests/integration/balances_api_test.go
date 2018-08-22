package integration_test

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gorilla/mux"

	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"

	cmn "github.com/BiJie/BinanceChain/common"
	common "github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/tokens/client/rest"
	tkstore "github.com/BiJie/BinanceChain/plugins/tokens/store"
)

func setupMultiStore() (sdk.MultiStore, *sdk.KVStoreKey, *sdk.KVStoreKey) {
	db := dbm.NewMemDB()
	accKey := cmn.AccountStoreKey
	tokensKey := cmn.TokenStoreKey
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(accKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tokensKey, sdk.StoreTypeIAVL, db)
	ms.LoadLatestVersion()
	return ms, accKey, tokensKey
}

func newCoreContext() context.CoreContext {
	// below does not work. tries to `mkdir .` for some reason
	// cctx := context.NewCoreContextFromViper()
	nodeURI := "tcp://localhost:26657"
	rpc := rpcclient.NewHTTP(nodeURI, "/")
	return context.CoreContext{
		Client:       rpc,
		NodeURI:      nodeURI,
		AccountStore: cmn.AccountStoreName,
	}
}

func TestSuccessNoCoins(t *testing.T) {
	ms, accKey, tokensKey := setupMultiStore()

	cdc := wire.NewCodec()
	auth.RegisterBaseAccount(cdc)

	ctx := sdk.NewContext(ms, abci.Header{}, false, log.NewNopLogger())
	mapper := tkstore.NewMapper(cdc, tokensKey)
	saddr := "cosmosaccaddr1atcjghcs273lg95p2kcrn509gdyx2h2g83l0mj"
	addr, err := sdk.AccAddressFromBech32(saddr)
	if err != nil {
		panic(err)
	}
	mapper.NewToken(ctx, common.NewToken("BNB", "BNB", 1000000000, addr))

	amapper := auth.NewAccountMapper(cdc, accKey, auth.ProtoBaseAccount)
	acc := amapper.NewAccountWithAddress(ctx, addr)
	// coins := sdk.Coins{sdk.Coin{Denom: "BNB", Amount: sdk.NewInt(100000000)}}
	// acc.SetCoins(coins)
	amapper.SetAccount(ctx, acc)

	cctx := newCoreContext()
	router := mux.NewRouter()
	rest.RegisterBalancesRoute(cctx, router, cdc, mapper)

	req := httptest.NewRequest("GET", fmt.Sprintf("/balances/%s", saddr), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, string(body), "{\"address\":\"cosmosaccaddr1atcjghcs273lg95p2kcrn509gdyx2h2g83l0mj\",\"balances\":[]}")
}
