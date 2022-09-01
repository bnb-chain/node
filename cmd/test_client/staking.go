package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/binance-chain/go-sdk/types/msg"
	"github.com/binance-chain/go-sdk/types/tx"
	"github.com/bnb-chain/node/common/types"
	stake "github.com/cosmos/cosmos-sdk/x/stake/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"log"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/binance-chain/go-sdk/client/rpc"
	sdkTypes "github.com/binance-chain/go-sdk/common/types"
	"github.com/binance-chain/go-sdk/keys"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	bankClient "github.com/cosmos/cosmos-sdk/x/bank/client"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto"
	cryptoAmino "github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/privval"
	"github.com/tidwall/gjson"
	"golang.org/x/xerrors"
)

var (
	txWithChainID tx.Option
)

type NodeInfo struct {
	Mnemonic      string
	DelegatorAddr cosmosTypes.AccAddress `json:"delegator_address"`
	ValidatorAddr cosmosTypes.ValAddress `json:"validator_address"`
	Addr          sdkTypes.AccAddress    `json:"address"`
	PubKey        crypto.PubKey          `json:"pubkey"`
	KeyManager    keys.KeyManager
}

func GetNodeInfo(i int) (*NodeInfo, error) {
	nodePath := path.Join("build", "devnet", fmt.Sprintf("node%d", i))
	seedPath := path.Join(nodePath, "testnodecli", "key_seed.json")
	content, err := os.ReadFile(seedPath)
	if err != nil {
		return nil, xerrors.Errorf("read file %s failed: %w", seedPath, err)
	}
	mnemonic := gjson.GetBytes(content, "secret").String()
	// key manager
	keyManager, err := keys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		return nil, xerrors.Errorf("new key manager failed: %w", err)
	}
	// load validator key
	privValKeyFile := path.Join(nodePath, "testnoded", "config", "priv_validator_key.json")
	keyJSONBytes, err := os.ReadFile(privValKeyFile)
	if err != nil {
		return nil, xerrors.Errorf("read file %s failed: %w", privValKeyFile, err)
	}
	pvKey := privval.FilePVKey{}
	cdc := amino.NewCodec()
	cryptoAmino.RegisterAmino(cdc)
	privval.RegisterRemoteSignerMsg(cdc)
	err = cdc.UnmarshalJSON(keyJSONBytes, &pvKey)
	if err != nil {
		return nil, xerrors.Errorf("Error reading PrivValidator key from %v: %v", privValKeyFile, err)
	}
	pvKey.PubKey = pvKey.PrivKey.PubKey()
	pvKey.Address = pvKey.PubKey.Address()
	return &NodeInfo{
		Mnemonic:      mnemonic,
		ValidatorAddr: cosmosTypes.ValAddress(keyManager.GetAddr()),
		DelegatorAddr: cosmosTypes.AccAddress(keyManager.GetAddr()),
		Addr:          keyManager.GetAddr(),
		PubKey:        pvKey.PrivKey.PubKey(),
		KeyManager:    keyManager,
	}, nil
}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenKeyManager() (km keys.KeyManager, err error) {
	pk, err := randomHex(32)
	log.Println("pk:", pk)
	if err != nil {
		return
	}
	return keys.NewPrivateKeyManager(pk)
}

func GenKeyManagerWithBNB(client *rpc.HTTP, tokenFrom keys.KeyManager) (km keys.KeyManager, err error) {
	km, err = GenKeyManager()
	if err != nil {
		return nil, xerrors.Errorf("GenKeyManager err: %w", err)
	}
	// send coin to the account
	client.SetKeyManager(tokenFrom)
	transfer := msg.Transfer{ToAddr: km.GetAddr(), Coins: sdkTypes.Coins{sdkTypes.Coin{
		Denom:  "BNB",
		Amount: 150e8,
	}}}
	txRes, err := client.SendToken([]msg.Transfer{transfer}, rpc.Commit, txWithChainID)
	if err != nil {
		return nil, xerrors.Errorf("send token error: %w", err)
	}
	assert(txRes.Code == 0, fmt.Sprintf("send token error, tx: %+v", txRes))
	return km, nil
}

func getConfigFromEnv() Config {
	env, ok := os.LookupEnv("STAKE_ENV")
	if !ok {
		env = "integration" // default env
	}
	switch env {
	case "integration":
		return Config{
			RPCAddr: "tcp://127.0.0.1:26657",
			Secret:  "bottom quick strong ranch section decide pepper broken oven demand coin run jacket curious business achieve mule bamboo remain vote kid rigid bench rubber",
		}
	case "multi":
		node, err := GetNodeInfo(0)
		if err != nil {
			panic(err)
		}
		return Config{
			RPCAddr: "tcp://127.0.0.1:8100",
			Secret:  node.Mnemonic,
		}
	default:
		panic("unknown env")
	}
}

type Config struct {
	RPCAddr string `json:"rpc_addr"` // rpc address to connect to
	Secret  string `json:"secret"`   // account which has enough coin to send
}

func Staking() error {
	rand.Seed(time.Now().UnixNano())
	// rpc client
	config := getConfigFromEnv()
	node0RpcAddr := config.RPCAddr
	c0 := rpc.NewRPCClient(node0RpcAddr, sdkTypes.ProdNetwork)
	status, err := c0.Status()
	chainId := status.NodeInfo.Network
	txWithChainID = tx.WithChainID(chainId)
	if err != nil {
		return xerrors.Errorf("get status error: %w", err)
	}
	log.Printf("chainId: %s\n", chainId)
	log.Printf("node0 status")
	log.Println(Pretty(status))
	// bob
	bob_secret := config.Secret
	bobKM, err := keys.NewMnemonicKeyManager(bob_secret)
	if err != nil {
		return xerrors.Errorf("new key manager failed: %w", err)
	}
	log.Printf("bob address: %s\n", bobKM.GetAddr())
	// create a random account
	validator0, err := GenKeyManagerWithBNB(c0, bobKM)
	if err != nil {
		return xerrors.Errorf("GenKeyManager err: %w", err)
	}
	log.Printf("validator0 address: %s\n", validator0.GetAddr())
	// validators
	validators, err := c0.GetStakeValidators()
	if err != nil {
		return xerrors.Errorf("get validators error: %w", err)
	}
	log.Println(Pretty(validators))
	validatorsLenBeforeCreate := len(validators)
	assert(validatorsLenBeforeCreate >= 1, "validators len should be >= 1")
	assert(len(validators[0].StakeSnapshots) != 0, "validators stake snapshot should not be 0")
	// query validators count (including jailed)
	validatorsCount, err := c0.GetAllValidatorsCount(true)
	if err != nil {
		return xerrors.Errorf("get all validators count error: %w", err)
	}
	log.Printf("validators count: %d\n", validatorsCount)
	// query validators count (excluding jailed)
	validatorsCountWithoutJail, err := c0.GetAllValidatorsCount(false)
	if err != nil {
		return xerrors.Errorf("get all validators count error: %w", err)
	}
	log.Printf("validators count: %d\n", validatorsCountWithoutJail)
	assert(validatorsCount == validatorsCountWithoutJail, "there is no jailed validators yet")
	// create validator
	amount := sdkTypes.Coin{Denom: "BNB", Amount: 123e8}
	des := sdkTypes.Description{Moniker: "node1"}
	rate, _ := sdkTypes.NewDecFromStr("1")
	maxRate, _ := sdkTypes.NewDecFromStr("1")
	maxChangeRate, _ := sdkTypes.NewDecFromStr("1")
	consensusPrivKey := ed25519.GenPrivKey()
	consensusPubKey := consensusPrivKey.PubKey()
	commission := sdkTypes.CommissionMsg{
		Rate:          rate,
		MaxRate:       maxRate,
		MaxChangeRate: maxChangeRate,
	}
	c0.SetKeyManager(validator0)
	txRes, err := c0.CreateValidator(amount, msg.Description(des), commission, consensusPubKey, rpc.Commit, tx.WithChainID(chainId))
	if err != nil {
		return xerrors.Errorf("create validator error: %w", err)
	}
	log.Printf("create validator tx: %+v\n", txRes)
	assert(txRes.Code == 0, "create validator tx return err")
	// check validators change
	validatorsCountAfterCreate, err := c0.GetAllValidatorsCount(true)
	if err != nil {
		return xerrors.Errorf("get all validators count error: %w", err)
	}
	log.Printf("validators count: %d\n", validatorsCountAfterCreate)
	assert(validatorsCountAfterCreate == validatorsCount+1, "validators count should be +1")
	// query top validators
	topValidators, err := c0.QueryTopValidators(1)
	if err != nil {
		return xerrors.Errorf("query top validators error: %w", err)
	}
	log.Printf("top validators: %+v\n", topValidators)
	assert(len(topValidators) == 1, "top validators should be 1")
	topValidator := topValidators[0]
	// query validator
	validator, err := c0.QueryValidator(sdkTypes.ValAddress(validator0.GetAddr()))
	if err != nil {
		return xerrors.Errorf("query validator error: %w", err)
	}
	log.Printf("query validator: %+v\n", validator)
	assert(validator != nil, "validator should not be nil")
	assert(bytes.Equal(validator.OperatorAddr, validator0.GetAddr()), "validator address should be equal")
	assert(validator.Tokens.String() == "12300000000", "validator tokens should be 123e8")
	assert(validator.Description == des, "validator description should be equal")
	assert(validator.Commission.Rate.Equal(rate), "validator rate should be equal")
	assert(validator.ConsPubKey == sdkTypes.MustBech32ifyConsPub(consensusPubKey), "validator cons pub key should be equal")
	// edit validator
	des2 := sdkTypes.Description{Moniker: "node1_v2"}
	consensusPrivKey2 := ed25519.GenPrivKey()
	consensusPubKey2 := consensusPrivKey2.PubKey()
	txRes, err = c0.EditValidator(msg.Description(des2), nil, consensusPubKey2, rpc.Commit, tx.WithChainID(chainId))
	if err != nil {
		return xerrors.Errorf("edit validator error: %w", err)
	}
	assert(txRes.Code == 0, fmt.Sprintf("edit validator tx return err, tx: %+v", txRes))
	// check edit validator change
	validator, err = c0.QueryValidator(sdkTypes.ValAddress(validator0.GetAddr()))
	if err != nil {
		return xerrors.Errorf("query validator error: %w", err)
	}
	log.Printf("query validator: %+v\n", validator)
	assert(validator != nil, "validator should not be nil")
	assert(bytes.Equal(validator.OperatorAddr, validator0.GetAddr()), "validator address should be equal")
	assert(validator.Description == des2, "validator description should be equal")
	assert(validator.ConsPubKey == sdkTypes.MustBech32ifyConsPub(consensusPubKey2), "validator cons pub key should be equal")
	tokenBeforeDelegate := validator.Tokens
	// delegate
	delegator, err := GenKeyManagerWithBNB(c0, bobKM)
	if err != nil {
		return xerrors.Errorf("GenKeyManager err: %w", err)
	}
	c0.SetKeyManager(delegator)
	var delegateAmount int64 = 5e8
	delegateCoin := sdkTypes.Coin{Denom: "BNB", Amount: delegateAmount}
	txRes, err = c0.Delegate(sdkTypes.ValAddress(validator0.GetAddr()), delegateCoin, rpc.Commit, tx.WithChainID(chainId))
	if err != nil {
		return xerrors.Errorf("delegate error: %w", err)
	}
	assert(txRes.Code == 0, fmt.Sprintf("delegate tx return err, tx: %+v", txRes))
	// check delegation
	validator, err = c0.QueryValidator(sdkTypes.ValAddress(validator0.GetAddr()))
	if err != nil {
		return xerrors.Errorf("query validator error: %w", err)
	}
	log.Printf("query validator: %+v\n", validator)
	tokenAfterDelegate := validator.Tokens
	assert(tokenAfterDelegate.Sub(tokenBeforeDelegate).Equal(sdkTypes.NewDec(delegateAmount)), "delegate tokens should be equal")
	// query delegation
	delegationQuery, err := c0.QueryDelegation(delegator.GetAddr(), sdkTypes.ValAddress(validator0.GetAddr()))
	if err != nil {
		return xerrors.Errorf("query delegation error: %w", err)
	}
	log.Printf("query delegation: %+v\n", delegationQuery)
	assert(delegationQuery.Delegation.Shares.Equal(sdkTypes.NewDec(delegateAmount)), "delegation shares should be equal")
	assert(delegationQuery.Balance.IsEqual(delegateCoin), "delegation balance should be equal")
	// query delegations
	delegations, err := c0.QueryDelegations(delegator.GetAddr())
	if err != nil {
		return xerrors.Errorf("query delegations error: %w", err)
	}
	log.Printf("query delegations: %+v\n", delegations)
	// check redelegate preparation
	topValAddr := topValidator.OperatorAddr
	validator0Addr := sdkTypes.ValAddress(validator0.GetAddr())
	topValidatorBeforeRedelegate, err := c0.QueryValidator(topValAddr)
	if err != nil {
		return xerrors.Errorf("query validator error: %w", err)
	}
	log.Printf("top validator before redelegate: %+v\n", topValidatorBeforeRedelegate)
	// redelegate from validator0 to top validator, should success immediately
	var redelegateAmount int64 = 2e8
	redelegateCoin := sdkTypes.Coin{Denom: "BNB", Amount: redelegateAmount}
	c0.SetKeyManager(delegator)
	txRes, err = c0.Redelegate(validator0Addr, topValAddr, redelegateCoin, rpc.Commit, tx.WithChainID(chainId))
	if err != nil {
		return xerrors.Errorf("redelegate error: %w", err)
	}
	assert(txRes.Code == 0, fmt.Sprintf("redelegate tx return err, tx: %+v", txRes))
	topValidatorAfterRedelegate, err := c0.QueryValidator(topValAddr)
	if err != nil {
		return xerrors.Errorf("query validator error: %w", err)
	}
	log.Printf("top validator after redelegate: %+v\n", topValidatorAfterRedelegate)
	assert(topValidatorAfterRedelegate.Tokens.Sub(topValidatorBeforeRedelegate.Tokens).Equal(sdkTypes.NewDec(redelegateAmount)), "redelegate tokens should be equal")
	// undelegate
	c0.SetKeyManager(delegator)
	txRes, err = c0.Undelegate(topValAddr, redelegateCoin, rpc.Commit, tx.WithChainID(chainId))
	if err != nil {
		return xerrors.Errorf("undelegate error: %w", err)
	}
	assert(txRes.Code == 0, fmt.Sprintf("undelegate tx return err, tx: %+v", txRes))
	topValidatorAfterUndelegate, err := c0.QueryValidator(topValAddr)
	if err != nil {
		return xerrors.Errorf("query validator error: %w", err)
	}
	log.Printf("top validator after undelegate: %+v\n", topValidatorAfterUndelegate)
	assert(topValidatorAfterUndelegate.Tokens.Equal(topValidatorBeforeRedelegate.Tokens), "check undelegation token change")
	// query pool
	pool, err := c0.GetPool()
	if err != nil {
		return xerrors.Errorf("get pool error: %w", err)
	}
	log.Printf("pool: %+v\n", pool)
	// query unbonding delegation
	unbondingDelegation, err := c0.QueryUnbondingDelegation(topValAddr, delegator.GetAddr())
	log.Printf("query unbonding delegation: %+v, err: %v\n", unbondingDelegation, err)
	if err != nil {
		return xerrors.Errorf("query unbonding delegation error: %w", err)
	}
	// query unbonding delegations
	unbondingDelegations, err := c0.QueryUnbondingDelegations(delegator.GetAddr())
	log.Printf("query unbonding delegations: %+v, err: %v\n", unbondingDelegations, err)
	if err != nil {
		return xerrors.Errorf("query unbonding delegations error: %w", err)
	}
	// query unbonding delegations by validator
	unbondingDelegationsByValidator, err := c0.GetUnBondingDelegationsByValidator(topValAddr)
	log.Printf("query unbonding delegations by validator: %+v, err: %v\n", unbondingDelegationsByValidator, err)
	if err != nil {
		return xerrors.Errorf("query unbonding delegations by validator error: %w", err)
	}
	// qeury redelegations by validator
	redelegationsByValidator, err := c0.GetRedelegationsByValidator(topValAddr)
	log.Printf("query redelegations by validator: %+v, err: %v\n", redelegationsByValidator, err)
	redelegationsByValidator, err = c0.GetRedelegationsByValidator(validator0Addr)
	log.Printf("query redelegations by validator: %+v, err: %v\n", redelegationsByValidator, err)
	if err != nil {
		return xerrors.Errorf("query redelegations by validator error: %w", err)
	}
	assert(len(redelegationsByValidator) > 0, "redelegations by validator should not be empty")
	// delegate to top validator and then redelegate
	delegator0, err := GenKeyManagerWithBNB(c0, bobKM)
	if err != nil {
		return xerrors.Errorf("GenKeyManager err: %w", err)
	}
	c0.SetKeyManager(delegator0)
	txRes, err = c0.Delegate(topValAddr, delegateCoin, rpc.Commit, tx.WithChainID(chainId))
	log.Printf("delegate to top validator tx: %+v, err: %v\n", txRes, err)
	if err != nil {
		return xerrors.Errorf("delegate error: %w", err)
	}
	assert(txRes.Code == 0, fmt.Sprintf("delegate tx return err, tx: %+v", txRes))
	c0.SetKeyManager(delegator0)
	log.Printf("dest validator: %+v\n", topValAddr)
	log.Printf("validator0 val address: %+v\n", sdkTypes.ValAddress(validator0.GetAddr()))
	log.Printf("delegator address: %+v\n", delegator0.GetAddr())
	txRes, err = c0.Redelegate(topValAddr, sdkTypes.ValAddress(validator0.GetAddr()), delegateCoin, rpc.Commit, tx.WithChainID(chainId))
	log.Printf("redelegate to validator0 tx: %+v, err: %v\n", txRes, err)
	if err != nil {
		return xerrors.Errorf("redelegate error: %w", err)
	}
	assert(txRes.Code == 0, fmt.Sprintf("redelegate tx return err, tx: %+v", txRes))
	// query redelegation
	redelegation, err := c0.QueryRedelegation(delegator0.GetAddr(), topValAddr, sdkTypes.ValAddress(validator0.GetAddr()))
	log.Printf("query redelegation: %+v, err: %v\n", redelegation, err)
	if err != nil {
		return xerrors.Errorf("query redelegation error: %w", err)
	}
	// query redelegations
	redelegations, err := c0.QueryRedelegations(delegator0.GetAddr())
	log.Printf("query redelegations: %+v, err: %v\n", redelegations, err)
	if err != nil {
		return xerrors.Errorf("query redelegations error: %w", err)
	}
	// query redelegations by source validator
	redelegationsByValidator, err = c0.GetRedelegationsByValidator(topValAddr)
	log.Printf("query redelegations by validator: %+v, err: %v\n", redelegationsByValidator, err)

	return nil
}

func assert(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

func StakingBackup() error {
	// rpc client
	node0RpcAddr := "tcp://127.0.0.1:8100"
	c0 := rpc.NewRPCClient(node0RpcAddr, sdkTypes.ProdNetwork)
	status, err := c0.Status()
	chainId := status.NodeInfo.Network
	if err != nil {
		return xerrors.Errorf("get status error: %w", err)
	}
	log.Printf("chainId: %s\n", chainId)
	log.Printf("node0 status")
	log.Println(Pretty(status))
	node1RpcAddr := "tcp://127.0.0.1:8101"
	c1 := rpc.NewRPCClient(node1RpcAddr, sdkTypes.ProdNetwork)
	status, err = c1.Status()
	if err != nil {
		return xerrors.Errorf("get status error: %w", err)
	}
	log.Printf("node1 status")
	log.Println(Pretty(status))

	// binance client
	bc0 := NewBinanceChainClient(node0RpcAddr, sdkTypes.ProdNetwork, chainId)
	//chainIdOption := func(msg *tx.StdSignMsg) *tx.StdSignMsg {
	//	msg.ChainID = chainId
	//	return msg
	//}

	// current validator
	validators, err := c0.GetStakeValidators()
	if err != nil {
		return xerrors.Errorf("get validators error: %w", err)
	}
	log.Printf("validators: %+v\n", validators)

	// accounts
	node0Info, err := GetNodeInfo(0)
	if err != nil {
		return xerrors.Errorf("get node0 info error: %w", err)
	}
	log.Printf("node0 address: %s", node0Info.Addr)
	node1Info, err := GetNodeInfo(1)
	if err != nil {
		return xerrors.Errorf("get node1 info error: %w", err)
	}
	log.Printf("node1 address: %s", node1Info.Addr)

	// transfer 2000000000000 BNB to node1
	sendCoinsMsg := bankClient.CreateMsg(node0Info.DelegatorAddr, node1Info.DelegatorAddr, cosmosTypes.Coins{cosmosTypes.NewCoin("BNB", 20000000000000)})
	_, err = bc0.Connect(node0Info.KeyManager).SignAndSendMsgs([]cosmosTypes.Msg{sendCoinsMsg}, nil)
	if err != nil {
		return xerrors.Errorf("failed to send coins: %w", err)
	}
	node1Account, err := bc0.Account(node1Info.Addr)
	if err != nil {
		return xerrors.Errorf("get bob account error: %w", err)
	}
	log.Printf("node1 account: %+v\n", node1Account)

	// stake
	stakeMsg := stake.MsgCreateValidator{
		Description: stake.Description{
			Moniker: "node1",
		},
		DelegatorAddr: node1Info.DelegatorAddr,
		ValidatorAddr: node1Info.ValidatorAddr,
		PubKey:        node1Info.PubKey,
		Delegation:    cosmosTypes.NewCoin(types.NativeTokenSymbol, 20000e8),
	}
	_, err = bc0.Connect(node1Info.KeyManager).SignAndSendMsgs([]cosmosTypes.Msg{stakeMsg}, nil)
	if err != nil {
		return xerrors.Errorf("failed to stake: %w", err)
	}
	// verify validator change
	validators, err = c0.GetStakeValidators()
	if err != nil {
		return xerrors.Errorf("get validators error: %w", err)
	}
	log.Println(Pretty(validators))
	return nil
}

func Pretty(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}
