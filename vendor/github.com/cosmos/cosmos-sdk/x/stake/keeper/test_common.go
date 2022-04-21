package keeper

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/sidechain"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/stake/types"
)

// dummy addresses used for testing
var (
	Addrs       = createTestAddrs(500)
	PKs         = createTestPubKeys(500)
	emptyAddr   sdk.AccAddress
	emptyPubkey crypto.PubKey

	addrDels = []sdk.AccAddress{
		Addrs[0],
		Addrs[1],
	}
	addrVals = []sdk.ValAddress{
		sdk.ValAddress(Addrs[2]),
		sdk.ValAddress(Addrs[3]),
		sdk.ValAddress(Addrs[4]),
		sdk.ValAddress(Addrs[5]),
		sdk.ValAddress(Addrs[6]),
	}
)

//_______________________________________________________________________________________

// intended to be used with require/assert:  require.True(ValEq(...))
func ValEq(t *testing.T, exp, got types.Validator) (*testing.T, bool, string, types.Validator, types.Validator) {
	return t, exp.Equal(got), "expected:\t%v\ngot:\t\t%v", exp, got
}

//_______________________________________________________________________________________

// create a codec used only for testing
func MakeTestCodec() *codec.Codec {
	var cdc = codec.New()

	// Register Msgs
	cdc.RegisterInterface((*sdk.Msg)(nil), nil)
	cdc.RegisterConcrete(bank.MsgSend{}, "test/stake/Send", nil)
	cdc.RegisterConcrete(types.MsgCreateValidator{}, "test/stake/CreateValidator", nil)
	cdc.RegisterConcrete(types.MsgRemoveValidator{}, "test/stake/RemoveValidator", nil)
	cdc.RegisterConcrete(types.MsgCreateValidatorProposal{}, "test/stake/CreateValidatorProposal", nil)
	cdc.RegisterConcrete(types.MsgEditValidator{}, "test/stake/EditValidator", nil)
	cdc.RegisterConcrete(types.MsgBeginUnbonding{}, "test/stake/BeginUnbonding", nil)
	cdc.RegisterConcrete(types.MsgBeginRedelegate{}, "test/stake/BeginRedelegate", nil)

	// Register AppAccount
	cdc.RegisterInterface((*sdk.Account)(nil), nil)
	cdc.RegisterConcrete(&auth.BaseAccount{}, "test/stake/Account", nil)
	codec.RegisterCrypto(cdc)

	return cdc
}

func getAccountCache(cdc *codec.Codec, ms sdk.MultiStore, accountKey *sdk.KVStoreKey) sdk.AccountCache {
	accountStore := ms.GetKVStore(accountKey)
	accountStoreCache := auth.NewAccountStoreCache(cdc, accountStore, 10)
	return auth.NewAccountCache(accountStoreCache)
}

// hodgepodge of all sorts of input required for testing
func CreateTestInput(t *testing.T, isCheckTx bool, initCoins int64) (sdk.Context, auth.AccountKeeper, Keeper) {

	keyStake := sdk.NewKVStoreKey("stake")
	keyStakeReward := sdk.NewKVStoreKey("stake_reward")
	tkeyStake := sdk.NewTransientStoreKey("transient_stake")
	keyAcc := sdk.NewKVStoreKey("acc")
	keyParams := sdk.NewKVStoreKey("params")
	tkeyParams := sdk.NewTransientStoreKey("transient_params")
	keyIbc := sdk.NewKVStoreKey("ibc")
	keySideChain := sdk.NewKVStoreKey("sc")

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(tkeyStake, sdk.StoreTypeTransient, nil)
	ms.MountStoreWithDB(keyStake, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStakeReward, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyIbc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySideChain, sdk.StoreTypeIAVL, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	mode := sdk.RunTxModeDeliver
	if isCheckTx {
		mode = sdk.RunTxModeCheck
	}

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "foochainid"}, mode, log.NewNopLogger())
	cdc := MakeTestCodec()
	accountKeeper := auth.NewAccountKeeper(
		cdc,                   // amino codec
		keyAcc,                // target store
		auth.ProtoBaseAccount, // prototype
	)

	accountCache := getAccountCache(cdc, ms, keyAcc)
	ctx = ctx.WithAccountCache(accountCache)

	ck := bank.NewBaseKeeper(accountKeeper)

	pk := params.NewKeeper(cdc, keyParams, tkeyParams)
	scKeeper := sidechain.NewKeeper(keySideChain, pk.Subspace(sidechain.DefaultParamspace), cdc)

	ibcKeeper := ibc.NewKeeper(keyIbc, pk.Subspace(ibc.DefaultParamspace), ibc.DefaultCodespace, scKeeper)
	keeper := NewKeeper(cdc, keyStake, keyStakeReward, tkeyStake, ck, nil, pk.Subspace(DefaultParamspace), types.DefaultCodespace)
	keeper.SetPool(ctx, types.InitialPool())
	keeper.SetParams(ctx, types.DefaultParams())
	sdk.UpgradeMgr.AddUpgradeHeight(sdk.LaunchBscUpgrade, 1)
	sdk.UpgradeMgr.AddUpgradeHeight(sdk.BEP128, 100)
	sdk.UpgradeMgr.Height = 100
	keeper.SetParams(ctx, types.DefaultParams())
	keeper.SetupForSideChain(&scKeeper, &ibcKeeper)

	// fill all the addresses with some coins, set the loose pool tokens simultaneously
	for _, addr := range Addrs {
		pool := keeper.GetPool(ctx)
		_, _, err := ck.AddCoins(ctx, addr, sdk.Coins{
			{keeper.BondDenom(ctx), sdk.NewDecWithoutFra(initCoins).RawInt()},
		})
		require.Nil(t, err)
		pool.LooseTokens = pool.LooseTokens.Add(sdk.NewDecWithoutFra(initCoins))
		keeper.SetPool(ctx, pool)
	}

	return ctx, accountKeeper, keeper
}

// hodgepodge of all sorts of input required for testing
func CreateTestInputWithGov(t *testing.T, isCheckTx bool, initCoins int64) (sdk.Context, auth.AccountKeeper, Keeper, gov.Keeper, *codec.Codec) {

	keyStake := sdk.NewKVStoreKey("stake")
	keyStakeReward := sdk.NewKVStoreKey("stake_reward")
	tkeyStake := sdk.NewTransientStoreKey("transient_stake")
	keyAcc := sdk.NewKVStoreKey("acc")
	keyParams := sdk.NewKVStoreKey("params")
	tkeyParams := sdk.NewTransientStoreKey("transient_params")
	govKey := sdk.NewKVStoreKey("gov")

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(tkeyStake, sdk.StoreTypeTransient, nil)
	ms.MountStoreWithDB(keyStake, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStakeReward, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(govKey, sdk.StoreTypeIAVL, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	mode := sdk.RunTxModeDeliver
	if isCheckTx {
		mode = sdk.RunTxModeCheck
	}

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "foochainid"}, mode, log.NewNopLogger())
	cdc := MakeTestCodec()
	gov.RegisterCodec(cdc)
	accountKeeper := auth.NewAccountKeeper(
		cdc,                   // amino codec
		keyAcc,                // target store
		auth.ProtoBaseAccount, // prototype
	)

	accountCache := getAccountCache(cdc, ms, keyAcc)
	ctx = ctx.WithAccountCache(accountCache)

	ck := bank.NewBaseKeeper(accountKeeper)
	pk := params.NewKeeper(cdc, keyParams, tkeyParams)
	keeper := NewKeeper(cdc, keyStake, keyStakeReward, tkeyStake, ck, nil, pk.Subspace(DefaultParamspace), types.DefaultCodespace)

	govKeeper := gov.NewKeeper(cdc, govKey, pk, pk.Subspace(gov.DefaultParamSpace), ck, keeper, gov.DefaultCodespace, &sdk.Pool{})

	keeper.SetPool(ctx, types.InitialPool())
	keeper.SetParams(ctx, types.DefaultParams())

	// fill all the addresses with some coins, set the loose pool tokens simultaneously
	for _, addr := range Addrs {
		pool := keeper.GetPool(ctx)
		_, _, err := ck.AddCoins(ctx, addr, sdk.Coins{
			{keeper.BondDenom(ctx), sdk.NewDecWithoutFra(initCoins).RawInt()},
		})
		require.Nil(t, err)
		pool.LooseTokens = pool.LooseTokens.Add(sdk.NewDecWithoutFra(initCoins))
		keeper.SetPool(ctx, pool)
	}

	return ctx, accountKeeper, keeper, govKeeper, cdc
}

func NewPubKey(pk string) (res crypto.PubKey) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		panic(err)
	}
	//res, err = crypto.PubKeyFromBytes(pkBytes)
	var pkEd ed25519.PubKeyEd25519
	copy(pkEd[:], pkBytes[:])
	return pkEd
}

// for incode address generation
func TestAddr(addr string, bech string) sdk.AccAddress {

	res, err := sdk.AccAddressFromHex(addr)
	if err != nil {
		panic(err)
	}
	bechexpected := res.String()
	if bech != bechexpected {
		panic("Bech encoding doesn't match reference")
	}

	bechres, err := sdk.AccAddressFromBech32(bech)
	if err != nil {
		panic(err)
	}
	if bytes.Compare(bechres, res) != 0 {
		panic("Bech decode and hex decode don't match")
	}

	return res
}

// nolint: unparam
func createTestAddrs(numAddrs int) []sdk.AccAddress {
	var addresses []sdk.AccAddress
	var buffer bytes.Buffer

	// start at 100 so we can make up to 999 test addresses with valid test addresses
	for i := 100; i < (numAddrs + 100); i++ {
		numString := strconv.Itoa(i)
		buffer.WriteString("A58856F0FD53BF058B4909A21AEC019107BA6") //base address string

		buffer.WriteString(numString) //adding on final two digits to make addresses unique
		res, _ := sdk.AccAddressFromHex(buffer.String())
		bech := res.String()
		addresses = append(addresses, TestAddr(buffer.String(), bech))
		buffer.Reset()
	}
	return addresses
}

func CreateTestAddr() sdk.AccAddress {
	bz := make([]byte, 20)
	_, _ = rand.Read(bz)
	return sdk.AccAddress(bz)
}

// nolint: unparam
func createTestPubKeys(numPubKeys int) []crypto.PubKey {
	var publicKeys []crypto.PubKey
	var buffer bytes.Buffer

	//start at 10 to avoid changing 1 to 01, 2 to 02, etc
	for i := 100; i < (numPubKeys + 100); i++ {
		numString := strconv.Itoa(i)
		buffer.WriteString("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AF") //base pubkey string
		buffer.WriteString(numString)                                                       //adding on final two digits to make pubkeys unique
		publicKeys = append(publicKeys, NewPubKey(buffer.String()))
		buffer.Reset()
	}
	return publicKeys
}

//_____________________________________________________________________________________

// does a certain by-power index record exist
func ValidatorByPowerIndexExists(ctx sdk.Context, keeper Keeper, power []byte) bool {
	store := ctx.KVStore(keeper.storeKey)
	return store.Has(power)
}

// update validator for testing
func TestingUpdateValidator(keeper Keeper, ctx sdk.Context, validator types.Validator) types.Validator {
	keeper.SetValidator(ctx, validator)
	keeper.SetValidatorByPowerIndex(ctx, validator)
	keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	validator, found := keeper.GetValidator(ctx, validator.OperatorAddr)
	if !found {
		panic("validator expected but not found")
	}
	return validator
}

func validatorByPowerIndexExists(k Keeper, ctx sdk.Context, power []byte) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(power)
}
