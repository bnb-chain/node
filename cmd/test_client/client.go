package main

import (
	"encoding/json"
	"log"

	"github.com/binance-chain/go-sdk/client/rpc"
	sdkTypes "github.com/binance-chain/go-sdk/common/types"
	"github.com/binance-chain/go-sdk/keys"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	context "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"golang.org/x/xerrors"

	"github.com/bnb-chain/node/app"
	"github.com/bnb-chain/node/wire"
)

// bnbcli sdk version
type BinanceChainClient struct {
	ChainID   string
	Cdc       *wire.Codec
	RpcClient *rpc.HTTP
	Signer    keys.KeyManager
}

type TxOptions struct {
	ChainID       string `json:"chain_id"`
	AccountNumber int64  `json:"account_number"`
	Sequence      int64  `json:"sequence"`
	Memo          string `json:"memo"`
	Source        int64  `json:"source"`
}

func NewBinanceChainClient(nodeURI string, network sdkTypes.ChainNetwork, chainId string) *BinanceChainClient {
	// get the codec
	cdc := app.Codec
	// init config for bnb
	ctx := app.ServerContext
	config := cosmosTypes.GetConfig()
	config.SetBech32PrefixForAccount(ctx.Bech32PrefixAccAddr, ctx.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(ctx.Bech32PrefixValAddr, ctx.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(ctx.Bech32PrefixConsAddr, ctx.Bech32PrefixConsPub)
	config.Seal()

	return &BinanceChainClient{
		ChainID:   chainId,
		Cdc:       cdc,
		RpcClient: rpc.NewRPCClient(nodeURI, network),
	}
}

func (c BinanceChainClient) Connect(signer keys.KeyManager) *BinanceChainClient {
	c.Signer = signer
	return &c
}

func (c *BinanceChainClient) BuildTx(msgs []cosmosTypes.Msg, option *TxOptions) (context.StdSignMsg, error) {
	if c.Signer == nil {
		return context.StdSignMsg{}, xerrors.Errorf("Signer is not set")
	}
	from, err := c.RpcClient.GetAccount(c.Signer.GetAddr())
	if err != nil {
		return context.StdSignMsg{}, xerrors.Errorf("get account failed: %w", err)
	}
	if option == nil {
		option = &TxOptions{}
	}
	if option.ChainID == "" {
		option.ChainID = c.ChainID
	}
	if option.AccountNumber == 0 {
		option.AccountNumber = from.GetAccountNumber()
	}
	if option.Sequence == 0 {
		option.Sequence = from.GetSequence()
	}
	return context.StdSignMsg{
		ChainID:       option.ChainID,
		AccountNumber: option.AccountNumber,
		Sequence:      option.Sequence,
		Memo:          option.Memo,
		Msgs:          msgs,
		Source:        option.Source,
	}, nil
}

func (c *BinanceChainClient) Account(addr sdkTypes.AccAddress) (sdkTypes.Account, error) {
	return c.RpcClient.GetAccount(addr)
}

func (c *BinanceChainClient) SignAndSendMsgs(msgs []cosmosTypes.Msg, option *TxOptions) (*rpc.ResultBroadcastTxCommit, error) {
	stdSignMsg, err := c.BuildTx(msgs, option)
	if err != nil {
		return nil, xerrors.Errorf("build tx error: %w", err)
	}
	log.Printf("stdSignMsg: %+v\n", stdSignMsg)
	msg := stdSignMsg.Bytes()
	sigBytes, err := c.Signer.GetPrivKey().Sign(msg)
	if err != nil {
		return nil, xerrors.Errorf("sign error: %w", err)
	}
	//log.Printf("sigBytes: %+v\n", sigBytes)
	pubkey := c.Signer.GetPrivKey().PubKey()
	sig := auth.StdSignature{
		AccountNumber: stdSignMsg.AccountNumber,
		Sequence:      stdSignMsg.Sequence,
		PubKey:        pubkey,
		Signature:     sigBytes,
	}
	txBytes, err := c.Cdc.MarshalBinaryLengthPrefixed(auth.NewStdTx(stdSignMsg.Msgs, []auth.StdSignature{sig}, stdSignMsg.Memo, stdSignMsg.Source, stdSignMsg.Data))
	if err != nil {
		return nil, xerrors.Errorf("marshal tx error: %w", err)
	}
	//log.Printf("txBytes: %+v\n", txBytes)
	//res, err := c.RpcClient.BroadcastTxSync(txBytes)
	res, err := c.RpcClient.BroadcastTxCommit(txBytes)
	if err != nil {
		return nil, xerrors.Errorf("broadcast tx error: %w", err)
	}
	resJson, err := json.Marshal(res)
	if err != nil {
		return nil, xerrors.Errorf("marshal tx res error: %w", err)
	}
	log.Printf("tx res: %s\n", string(resJson))
	if res.CheckTx.IsErr() {
		return nil, xerrors.Errorf("check tx error: %s", res.CheckTx.Log)
	}
	return res, nil
}
