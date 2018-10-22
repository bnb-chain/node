package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/bank"
	rpcclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/BiJie/BinanceChain/app"
	"github.com/BiJie/BinanceChain/wire"
)

const (
	flagHome        = "home"
	flagClusterMode = "cluster_mode"
)

type RainConfig struct {
	// Chain info
	Chain      string
	Amounts    string
	MakerNames []string
	MakerInUse string
	PassWord   string
	ListenAddr string
	NodeAddr   string

	// deploy info
	HomePath         string
	ClusterMode      bool
	MaxMessageInChan int
	TransGap         time.Duration
}

type ClaimCoin struct {
	Address string
}

type ErrorMess struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func main() {
	var conf RainConfig
	conf.addFlags()
	pflag.Parse()
	conf.initConfig()

	c := make(chan ClaimCoin, conf.MaxMessageInChan)
	go startWorker(conf, c)
	http.HandleFunc("/claim", getCoinsHandler(c))

	if err := http.ListenAndServe(conf.ListenAddr, nil); err != nil {
		log.Fatal("Failed to start server", err)
	}
}

func startWorker(conf RainConfig, c <-chan ClaimCoin) {
	cdc := app.MakeCodec()
	for {

		addresses := make([]string, 0)
		addressSet := make(map[string]struct{}, 0)
		// Can't  transfer coins to rainmaker itself.
		addressSet[conf.MakerInUse] = struct{}{}
		// Collect messages in limit time and do it in one transaction.
		tick := time.After(conf.TransGap)
	Exit:
		for {
			select {
			case claim := <-c:
				// Filter repeated address
				if _, ok := addressSet[claim.Address]; !ok {
					addresses = append(addresses, claim.Address)
					addressSet[claim.Address] = struct{}{}
				}
			case <-tick:
				break Exit
			}
		}
		if len(addresses) == 0 {
			continue
		}
		err := sendTx(cdc, addresses, conf.Amounts, conf)
		if err != nil {
			log.Println(fmt.Sprintf("Sent tx failed %v", err))
			fmt.Printf("Sent tx failed %v", err)
		}
	}
}

func (c *RainConfig) addFlags() {
	pflag.StringVar(&c.HomePath, flagHome, "/bnbchaind/config/", "Directory for config and data . Default \"/bnbchaind/config/\")")
	pflag.BoolVar(&c.ClusterMode, flagClusterMode, false, "Whether run in cluster mode. Default false")
}

func (c *RainConfig) initConfig() {
	viper.SetConfigName("rainmaker")
	viper.AddConfigPath(c.HomePath)
	viper.Set("home", c.HomePath)
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	c.Chain = viper.GetString("chain_id")
	c.Amounts = viper.GetString("amounts")
	c.MakerNames = viper.GetStringSlice("rainmakers")
	c.PassWord = viper.GetString("rainmaker_password")
	c.ListenAddr = viper.GetString("listen_addr")
	c.NodeAddr = viper.GetString("node_addr")
	c.MaxMessageInChan = viper.GetInt("max_message")
	c.TransGap = viper.GetDuration("trans_gap")

	if len(c.MakerNames) < 1 {
		panic("MakeNames is missing")
	}
	if c.ClusterMode == false {
		c.MakerInUse = c.MakerNames[0]
	} else {
		hostname, err := os.Hostname()
		if err != nil {
			panic(fmt.Sprintf("Get hostname failed. Error: %v", err))
		}
		// In cluster mode, is deployed as Statefulset of Kubernetes. The hostname of different instance will be like rainmaker-0, rainmaker-1... rainmaker-n
		ins := strings.TrimPrefix(hostname, "rainmaker-")
		instance, err := strconv.Atoi(ins)
		if err != nil {
			panic(fmt.Sprintf("Hostname %s is invalid", hostname))
		}
		if len(c.MakerNames) < instance+1 {
			panic(fmt.Sprintf("The length of rainmaker_addr(%d) is less than the instance number (%d).", len(c.MakerNames), instance+1))
		}
		c.MakerInUse = c.MakerNames[instance]
	}
}

func getCoinsHandler(c chan<- ClaimCoin) func(w http.ResponseWriter, request *http.Request) {
	return func(w http.ResponseWriter, request *http.Request) {
		var claim ClaimCoin

		// decode JSON response from front end
		decoder := json.NewDecoder(request.Body)
		decoderErr := decoder.Decode(&claim)
		if decoderErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err := sdk.AccAddressFromBech32(claim.Address)
		if err != nil {
			handleErrMes(w, &ErrorMess{Message: fmt.Sprintf("The address in not in bech32 format. %v", err)}, http.StatusBadRequest)
			return
		}
		tick := time.After(time.Second)
		select {
		case c <- claim:
			w.WriteHeader(http.StatusOK)
			return
		case <-tick:
			handleErrMes(w, &ErrorMess{Message: "Timeout to deliver message"}, http.StatusNotAcceptable)
			return
		}
	}
}

func handleErrMes(w http.ResponseWriter, obj interface{}, code int) {
	bs, _ := json.Marshal(obj)
	w.WriteHeader(code)
	w.Write(bs)
}

func sendTx(cdc *wire.Codec, toStrs []string, amount string, conf RainConfig) error {
	ctx := NewCoreContext(conf.NodeAddr, conf.Chain, conf.MakerInUse)
	ctx.CoreContext.Decoder = authcmd.GetAccountDecoder(cdc)
	// get the from/to address
	from, err := ctx.GetFromAddress()
	if err != nil {
		return err
	}
	//Skip check from address since we make sure it exist.
	tos := make([]sdk.AccAddress, len(toStrs))
	for index, toStr := range toStrs {
		to, err := sdk.AccAddressFromBech32(toStr)
		if err != nil {
			return err
		}
		tos[index] = to
	}
	// parse coins trying to be sent
	coins, err := sdk.ParseCoins(amount)
	if err != nil {
		return err
	}
	// skip ensure account has enough coins.
	msg := BuildMsg(from, tos, coins)

	err = ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, []sdk.Msg{msg}, cdc, conf.PassWord)
	if err != nil {
		return err
	}
	return nil

}

// Todo delete this when go-sdk is provided
type CoreContext struct {
	context.CoreContext
}

func NewCoreContext(nodeURI string, chainID string, keyName string) CoreContext {
	var rpc rpcclient.Client
	if nodeURI != "" {
		rpc = rpcclient.NewHTTP(nodeURI, "/websocket")
	}
	return CoreContext{context.CoreContext{
		ChainID:         chainID,
		Gas:             200000,
		Fee:             "",
		TrustNode:       true,
		FromAddressName: keyName,
		NodeURI:         nodeURI,
		AccountNumber:   0,
		Sequence:        0,
		Memo:            "Magic coin",
		Client:          rpc,
		Decoder:         nil,
		AccountStore:    "acc",
		UseLedger:       false,
		Async:           false,
		JSON:            true,
		PrintResponse:   true,
	}}
}

func (ctx CoreContext) EnsureSignBuildBroadcast(name string, msgs []sdk.Msg, cdc *wire.Codec, passPhrase string) (err error) {

	txBytes, err := ctx.ensureSignBuild(name, msgs, cdc, passPhrase)
	if err != nil {
		return err
	}

	if ctx.Async {
		res, err := ctx.BroadcastTxAsync(txBytes)
		if err != nil {
			return err
		}
		if ctx.JSON {
			type toJSON struct {
				TxHash string
			}
			valueToJSON := toJSON{res.Hash.String()}
			JSON, err := cdc.MarshalJSON(valueToJSON)
			if err != nil {
				return err
			}
			fmt.Println(string(JSON))
		} else {
			fmt.Println("Async tx sent. tx hash: ", res.Hash.String())
		}
		return nil
	}
	res, err := ctx.BroadcastTx(txBytes)
	if err != nil {
		return err
	}
	if ctx.JSON {
		// Since JSON is intended for automated scripts, always include response in JSON mode
		type toJSON struct {
			Height   int64
			TxHash   string
			Response string
		}
		valueToJSON := toJSON{res.Height, res.Hash.String(), fmt.Sprintf("%+v", res.DeliverTx)}
		JSON, err := cdc.MarshalJSON(valueToJSON)
		if err != nil {
			return err
		}
		fmt.Println(string(JSON))
		return nil
	}
	if ctx.PrintResponse {
		fmt.Printf("Committed at block %d. Hash: %s Response:%+v \n", res.Height, res.Hash.String(), res.DeliverTx)
	} else {
		fmt.Printf("Committed at block %d. Hash: %s \n", res.Height, res.Hash.String())
	}
	return nil
}

func (ctx CoreContext) ensureSignBuild(name string, msgs []sdk.Msg, cdc *wire.Codec, passPhrase string) (tyBytes []byte, err error) {
	err = context.EnsureAccountExists(ctx.CoreContext, name)
	if err != nil {
		return nil, err
	}

	ctx.CoreContext, err = context.EnsureAccountNumber(ctx.CoreContext)
	if err != nil {
		return nil, err
	}
	// default to next sequence number if none provided
	ctx.CoreContext, err = context.EnsureSequence(ctx.CoreContext)
	if err != nil {
		return nil, err
	}
	var txBytes []byte

	txBytes, err = ctx.SignAndBuild(name, passPhrase, msgs, cdc)
	if err != nil {
		return nil, fmt.Errorf("Error signing transaction: %v. ", err)
	}

	return txBytes, err
}

func BuildMsg(from sdk.AccAddress, tos []sdk.AccAddress, coins sdk.Coins) sdk.Msg {
	totalCoin := coins
	for i := 0; i < len(tos)-1; i++ {
		totalCoin = totalCoin.Plus(coins)
	}
	input := bank.NewInput(from, totalCoin)
	output := make([]bank.Output, len(tos))
	for index, to := range tos {
		output[index] = bank.NewOutput(to, coins)
	}
	msg := bank.NewMsgSend([]bank.Input{input}, output)
	return msg
}
