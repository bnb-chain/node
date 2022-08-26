package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/binance-chain/go-sdk/client/rpc"
	sdkTypes "github.com/binance-chain/go-sdk/common/types"
	"github.com/binance-chain/go-sdk/keys"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	bankClient "github.com/cosmos/cosmos-sdk/x/bank/client"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto"
	cryptoAmino "github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/privval"
	"github.com/tidwall/gjson"
	"golang.org/x/xerrors"

	"github.com/bnb-chain/node/common/types"
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

func Staking() error {
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
