package main

import (
	"encoding/json"
	"fmt"
	"github.com/binance-chain/go-sdk/client/rpc"
	sdkTypes "github.com/binance-chain/go-sdk/common/types"
	"github.com/binance-chain/go-sdk/keys"
	"github.com/bnb-chain/node/app"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	bankClient "github.com/cosmos/cosmos-sdk/x/bank/client"
	"github.com/cosmos/cosmos-sdk/x/stake"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/privval"
	"github.com/tidwall/gjson"
	"golang.org/x/xerrors"
	"log"
	"os"
	"path"
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
	privValStateFile := path.Join(nodePath, "testnoded", "data", "priv_validator_state.json")
	filePV := privval.LoadFilePV(privValKeyFile, privValStateFile)
	return &NodeInfo{
		Mnemonic:      mnemonic,
		ValidatorAddr: cosmosTypes.ValAddress(keyManager.GetAddr()),
		DelegatorAddr: cosmosTypes.AccAddress(keyManager.GetAddr()),
		Addr:          keyManager.GetAddr(),
		PubKey:        filePV.GetPubKey(),
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
	PrettyPrint(status)
	node1RpcAddr := "tcp://127.0.0.1:8101"
	c1 := rpc.NewRPCClient(node1RpcAddr, sdkTypes.ProdNetwork)
	status, err = c1.Status()
	if err != nil {
		return xerrors.Errorf("get status error: %w", err)
	}
	log.Printf("node1 status")
	PrettyPrint(status)

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
	sendCoinsMsg := bankClient.CreateMsg(node0Info.DelegatorAddr, node1Info.DelegatorAddr, cosmosTypes.Coins{cosmosTypes.NewCoin("BNB", 2000000000000)})
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
		Delegation:    app.DefaultSelfDelegationToken,
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
	PrettyPrint(validators)
	return nil
}

func PrettyPrint(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	log.Println(string(b))
}
