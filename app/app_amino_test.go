package app

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/binance-chain/node/common/testutils"
	"github.com/binance-chain/node/plugins/amino"
)

func TestAminoEncodeDecodeTx(t *testing.T) {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("bnb", "bnbp")

	ms, _, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := MakeCodec()
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	querier := amino.NewQuerier(cdc)

	txjson := `{"type":"auth/StdTx","value":{"msg":[{"type":"cosmos-sdk/Send","value":{"inputs":[{"address":"bnb15dt3vnxpyn8rurtapxnll2wnfwmntqgnsh5r5a","coins":[{"denom":"BNB","amount":"100000000"}]}],"outputs":[{"address":"bnb1ljn28cfc638qq9e30ug7zqwk200tm82hg4p0t2","coins":[{"denom":"BNB","amount":"100000000"}]}]}}],"signatures":[{"pub_key":{"type":"tendermint/PubKeySecp256k1","value":"A2uekxgZJwRKXMvreoERHBSQsWIGnnaRSddt00SgP62x"},"signature":"A5FNT3FWn+RSC+XQ4FHXhvm813VOyGqz53aS2LHCQ0YogMYVV+odQIZ7Zv+X6poB+uBS3MRwK5288s3zV4c5jw==","account_number":"0","sequence":"3"}],"memo":"","source":"0","data":null}}`
	req := abci.RequestQuery{
		Data: []byte(txjson),
	}

	response, err := querier(ctx, []string{amino.EncodeTx}, req)
	require.NoError(t, err)

	txBytes, decodeErr := hex.DecodeString("c001f0625dee0a4c2a2c87fa0a220a14a357164cc124ce3e0d7d09a7ffa9d34bb7358113120a0a03424e421080c2d72f12220a14fca6a3e138d44e0017317f11e101d653debd9d57120a0a03424e421080c2d72f126c0a26eb5ae98721036b9e93181927044a5ccbeb7a81111c1490b162069e769149d76dd344a03fadb1124003914d4f71569fe4520be5d0e051d786f9bcd7754ec86ab3e77692d8b1c243462880c61557ea1d40867b66ff97ea9a01fae052dcc4702b9dbcf2cdf35787398f2003")
	require.NoError(t, decodeErr)
	require.Equal(t, txBytes, response)

	req = abci.RequestQuery{
		Data: txBytes,
	}
	response, err = querier(ctx, []string{amino.DecodeTx}, req)
	require.NoError(t, err)
	require.Equal(t, []byte(txjson), response)
}

func TestAminoDecodeAcc(t *testing.T)  {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("bnb", "bnbp")

	ms, _, _ := testutils.SetupMultiStoreForUnitTest()
	cdc := MakeCodec()
	ctx := sdk.NewContext(ms, abci.Header{}, sdk.RunTxModeDeliver, log.NewNopLogger())
	querier := amino.NewQuerier(cdc)

	accBytes, _ := hex.DecodeString("4bdc4c270a500a14a357164cc124ce3e0d7d09a7ffa9d34bb7358113120e0a03424e4210809aedc7bf9fc3231a26eb5ae98721036b9e93181927044a5ccbeb7a81111c1490b162069e769149d76dd344a03fadb1280412056e6f646530")
	req := abci.RequestQuery{
		Data: accBytes,
	}

	response, err := querier(ctx, []string{amino.DecodeAcc}, req)
	require.NoError(t, err)

	jsonAcc := `{"type":"bnbchain/Account","value":{"base":{"address":"bnb15dt3vnxpyn8rurtapxnll2wnfwmntqgnsh5r5a","coins":[{"denom":"BNB","amount":"19998999700000000"}],"public_key":{"type":"tendermint/PubKeySecp256k1","value":"A2uekxgZJwRKXMvreoERHBSQsWIGnnaRSddt00SgP62x"},"account_number":"0","sequence":"4"},"name":"node0","frozen":null,"locked":null,"flags":"0"}}`
	require.Equal(t, jsonAcc, string(response))
}
