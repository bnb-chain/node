package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/BiJie/BinanceChain/common/client"
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
	"github.com/BiJie/BinanceChain/plugins/dex"
	"github.com/BiJie/BinanceChain/plugins/dex/order"
	"github.com/BiJie/BinanceChain/plugins/tokens"
	"github.com/BiJie/BinanceChain/wire"
	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/spf13/viper"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
	txbuilder "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/client/keys"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("bnc", "bncp")
	config.SetBech32PrefixForValidator("bva", "bvap")
	config.SetBech32PrefixForConsensusNode("bca",  "bcap")
	config.Seal()
}

const (
	createTask = 1
	submitTask = 2
	buy = 1
	sell = 2
)

var cHome *string
var owner *string
var vNode *string
var vNodes []string
var chainId *string
var ordersChnBuf *int
var createPoolSize *int
var transChnBuf *int
var submitPoolSize *int
var diskCachePath *string
var userPrefix *string
var generateToken *bool
var initialAccount *bool
var batchSize *int
var runCreate *bool
var runSubmit *bool
var submitOnlyPath *string
var submitOnlySize *int
var submitThinkTime *int64
var csvPath *string

type DEXOrder struct {
	ctx context.CLIContext
	txBldr txbuilder.TxBuilder
	addr sdk.AccAddress
	side int8
	symbol string
	price int64
	qty int64
	tifCode int8
}

type sequence struct {
	m sync.Mutex
	seqMap map[string]int64
}

var ordersChn chan DEXOrder
var transChn chan []byte
var clientSeq sequence
var rpcs []*rpcclient.HTTP

func init() {
	cHome = flag.String("cHome", "/home/test/.bnbcli", "client home folder")
	owner = flag.String("owner", "test4", "owner account")
	vNode = flag.String("vNode", "0.0.0.0:26657", "target validator ip:port")
	chainId = flag.String("chainId", "test-chain-sT34W7", "chain id")
	ordersChnBuf = flag.Int("ordersChnBuf", 8, "orders channel buffer")
	createPoolSize = flag.Int("createPoolSize", 4, "create orders pool size")
	transChnBuf = flag.Int("transChnBuf", 128, "trans channel buffer")
	submitPoolSize = flag.Int("submitPoolSize", 64, "submit trans pool size")
	diskCachePath = flag.String("diskCachePath", "/home/test/orders", "disk cache path")
	userPrefix = flag.String("userPrefix", "node1_user", "user account prefix")
	generateToken = flag.Bool("generateToken", true, "if to generate tokens")
	initialAccount = flag.Bool("initialAccount", true, "if to initial accounts")
	batchSize = flag.Int("batchSize", 100, "batch size, i.e. # of prices generated")
	runCreate = flag.Bool("runCreate", false, "if to run create")
	runSubmit = flag.Bool("runSubmit", false, "if to run submit")
	submitOnlyPath = flag.String("submitOnlyPath", "/home/test/orders2", "disk cache path")
	submitOnlySize = flag.Int("submitOnlySize", 1, "# of submits")
	submitThinkTime = flag.Int64("submitThinkTime", 0, "submit think time in ms")
	csvPath = flag.String("csvPath", "/home/test/trans.csv", "csv path")
	flag.Parse()
	ordersChn = make(chan DEXOrder, *ordersChnBuf)
	transChn = make(chan []byte, *transChnBuf)
	clientSeq = sequence {seqMap: make(map[string]int64)}
	vNodes = strings.Split(*vNode, ",")
	rpcs = make([]*rpcclient.HTTP, len(vNodes))
	for i, v := range vNodes {
		rpcs[i] = rpcclient.NewHTTP(v, "/websocket")
	}
}

func MakeCodec() *wire.Codec {
	var cdc = wire.NewCodec()
	wire.RegisterCrypto(cdc)
	bank.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	dex.RegisterWire(cdc)
	tokens.RegisterWire(cdc)
	types.RegisterWire(cdc)
	tx.RegisterWire(cdc)
	return cdc
}

func generatePrices(noOfPrices int, margin float64) []int64 {
	rand.Seed(1)
	prices := make([]int64, noOfPrices)
	for i := 0; i < noOfPrices; i++ {
		f := rand.Float64() + margin
		s := fmt.Sprintf("%.4f", f)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			panic(err)
		}
		prices[i] = int64(f*10000)*10000
	}
	return prices
}

func build(user string, side int8, symbol string, price int64, qty int64, tif string) DEXOrder {
	cdc := MakeCodec()
	viper.Set("node", vNodes[0])
	viper.Set("chain-id", *chainId)
	viper.Set("home", fmt.Sprintf("%s", *cHome))
	viper.Set("from", user)
	viper.Set("trust-node", true)
	ctx, txBldr := client.PrepareCtx(cdc)
	ctx.Client = rpcs[0]
	addr, err := ctx.GetFromAddress()
	if err != nil {
		fmt.Println(err)
	}
	accNum, err := ctx.GetAccountNumber(addr)
	if err != nil {
		fmt.Println(err)
	}
	txBldr = txBldr.WithAccountNumber(accNum)
	tifCode, err := order.TifStringToTifCode(tif)
	if err != nil {
		fmt.Println(err)
	}
	return DEXOrder{ctx, txBldr, addr, side, symbol, price, qty, tifCode}
}

func allocateOrders(tokens []string, users []string) {
	var noOfPrices int = *batchSize
	/*
	when margin is 1:1, almost no lower and higher are produced
	when margin is 1.01:1, ~1% of lower and higher are produced
	when margin is 1.10:1, ~10% of lower anb higher are produced
 	*/
	var buyPrices []int64 = generatePrices(noOfPrices, 1.00)
	var sellPrices []int64 = generatePrices(noOfPrices, 1.01)

	orderIndex := 0
	userIndex := 0

	for i := 0; i < noOfPrices; i++ {
		for j := 0; j < len(tokens); j++ {
			symbol := fmt.Sprintf("%s_BNB", tokens[j])
			fmt.Printf("allocating #%d\n", orderIndex)
			ordersChn <- build(users[userIndex], buy, symbol, buyPrices[i], 100000000, "GTC")
			orderIndex+=1
			userIndex+=1
			if userIndex == len(users) {
				userIndex = 0
			}
			ordersChn <- build(users[userIndex], sell, symbol, sellPrices[i], 100000000, "GTC")
			orderIndex+=1
			userIndex+=1
			if userIndex == len(users) {
				userIndex = 0
			}
			fmt.Println("b:", buyPrices[i], "s:", sellPrices[i])
		}
	}
	close(ordersChn)
}

func create(wg *sync.WaitGroup, s *sequence) {
	for item := range ordersChn {
		name, err := item.ctx.GetFromName()
		if err != nil {
			fmt.Println(err)
			continue
		}
		s.m.Lock()
		seq, hasKey := s.seqMap[name]
		s.m.Unlock()
		if hasKey == false {
			var err error
			seq, err = item.ctx.GetAccountSequence(item.addr)
			if err != nil {
				fmt.Println(err)
				continue
			}
		}
		item.txBldr = item.txBldr.WithSequence(seq)
		id := fmt.Sprintf("%X-%d", item.addr, seq+1)
		msg := order.NewOrderMsg{
			//Version: 0x01,
			Sender: item.addr,
			Id: id,
			Symbol: item.symbol,
			OrderType: order.OrderType.LIMIT,
			Side: item.side,
			Price: item.price,
			Quantity: item.qty,
			TimeInForce: order.TimeInForce.GTC,
		}
		msg.TimeInForce = item.tifCode
		msgs := []sdk.Msg{msg}
		// txBytes, err := item.txBldr.BuildAndSign(name, "1qaz2wsx", msgs)
		ssMsg := txbuilder.StdSignMsg {
			ChainID: item.txBldr.ChainID,
			AccountNumber: item.txBldr.AccountNumber,
			Sequence: item.txBldr.Sequence,
			Memo: item.txBldr.Memo,
			Msgs: msgs,
		}
		keybase, err := keys.GetKeyBaseFromDir(*cHome)
		if err != nil {
			fmt.Println(err)
			continue
		}
		sigBytes, pubkey, err := keybase.Sign(name, "1qaz2wsx", ssMsg.Bytes())
		if err != nil {
			fmt.Println(err)
			continue
		}
		sig := auth.StdSignature {
			AccountNumber: ssMsg.AccountNumber,
			Sequence: ssMsg.Sequence,
			PubKey: pubkey,
			Signature: sigBytes,
		}
		txBytes, err := item.txBldr.Codec.MarshalBinary(auth.NewStdTx(ssMsg.Msgs, []auth.StdSignature{sig}, ssMsg.Memo))
		if err != nil {
			fmt.Println("failed to sign tran: %v", err)
			continue
		}
		ts := fmt.Sprintf("%d", time.Now().UnixNano())
		file := filepath.Join(*diskCachePath, ts + "_" + name)
		fmt.Println("Acc-", item.txBldr.AccountNumber, "signed tran saved,", file)
		err = ioutil.WriteFile(file, txBytes, 0777)
		if err != nil {
			fmt.Println(err)
			continue
		}
		s.m.Lock()
		s.seqMap[name] = seq+1
		s.m.Unlock()
	}
	wg.Done()
}

func doCreateTask(tokens []string, users []string) {
	go allocateOrders(tokens, users)
	execute(*createPoolSize, createTask)
}

func allocateTrans() {
	var trans [][]byte
	files, err := ioutil.ReadDir(*diskCachePath)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		tran, err := ioutil.ReadFile(filepath.Join(*diskCachePath, file.Name()))
		if err != nil {
			panic(err)
		}
		trans = append(trans, tran)
	}
	for index, item := range trans {
		fmt.Printf("allocate tran #%d\n", index)
		transChn <- item
	}
	close(transChn)
}

func doRecover() {
	if r := recover(); r != nil {
		fmt.Println("recoved from", r)
		debug.PrintStack()
	}
}

func async(ctx context.CLIContext, txBldr txbuilder.TxBuilder, tran []byte) {
	defer doRecover()
	res, err := ctx.BroadcastTxAsync(tran)
	if err != nil {
		fmt.Println(err)
	}
	if ctx.JSON {
		type toJSON struct {
			TxHash string
		}
		valueToJSON := toJSON{res.Hash.String()}
		JSON, err := txBldr.Codec.MarshalJSON(valueToJSON)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(string(JSON))
	} else {
		fmt.Println("tran hash: ", res.Hash.String())
	}
}

func submit(wg *sync.WaitGroup, ctx context.CLIContext, txBldr txbuilder.TxBuilder) {
	for tran := range transChn {
		async(ctx, txBldr, tran)
		time.Sleep(time.Duration(*submitThinkTime) * time.Millisecond)
	}
	wg.Done()
}

func doSubmitTask() {
	go allocateTrans()
	execute(*submitPoolSize, submitTask)
}

func execute(poolSize int, mode int) {
	var wg sync.WaitGroup
	wg.Add(poolSize)
	vIndex := 0
	for i := 0; i < poolSize; i++ {
		if mode == createTask {
			go create(&wg, &clientSeq)
		}
		if mode == submitTask {
			cdc := MakeCodec()
			viper.Set("node", vNodes[vIndex])
			viper.Set("chain-id", *chainId)
			viper.Set("gas", 20000000000000)
			ctx, txBldr := client.PrepareCtx(cdc)
			ctx.Client = rpcs[vIndex]
			vIndex+=1
			if vIndex == len(vNodes) {
				vIndex = 0
			}
			go submit(&wg, ctx, txBldr)
		}
	}
	wg.Wait()
}

func generateAccounts(keyword string) [][]string {
	var users []string
	var addresses []string
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bnbcli", "--home="+*cHome, "keys", "list")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		panic(fmt.Sprint(err) + " : " + stderr.String())
	}
	expr := "(" + keyword + "[\\d]+).+(bnc.+).+bnc"
	res, err := regexp.Compile(expr)
	if err != nil {
		panic(err)
	}
	m := res.FindAllStringSubmatch(stdout.String(), -1)
	if m != nil {
		for _, v := range m {
			users = append(users, v[1])
			addresses = append(addresses, v[2])
		}
	} else {
		panic("no matching accounts found")
	}
	return [][]string{users, addresses}
}

func generateTokens(sI int, eI int, flag bool) []string {
	var tokens []string
	for sI <= eI {
		var token string
		if sI < 10 {
			token = fmt.Sprintf("X0%d", sI)
		} else {
			token = fmt.Sprintf("X%d", sI)
		}
		if flag == true {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd := exec.Command("bnbcli", "token", "issue", "--home="+*cHome, "--node="+*vNode, "--token-name="+token, "--symbol="+token, "--total-supply=20000000000000000", "--from="+*owner, "--chain-id="+*chainId)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			if err != nil {
				panic(fmt.Sprint(err) + " : " + stderr.String())
			}
			fmt.Println(token, stdout.String())
			time.Sleep(5 * time.Second)
			stdout.Reset()
			stderr.Reset()
			cmd = exec.Command("bnbcli", "dex", "list", "--home="+*cHome, "--node="+*vNode, "--base-asset-symbol="+token, "--quote-asset-symbol=BNB", "--init-price=100000000", "--from="+*owner, "--chain-id="+*chainId)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err = cmd.Run()
			if err != nil {
				panic(fmt.Sprint(err) + " : " + stderr.String())
			}
			fmt.Println(token, stdout.String())
			time.Sleep(5 * time.Second)
			stdout.Reset()
			stderr.Reset()
		}
		tokens = append(tokens, token)
		sI++
	}
	return tokens
}

func initializeAccounts(addresses []string, tokens []string, flag bool) {
	tokens = append(tokens, "BNB")
	if flag == true {
		var buffer bytes.Buffer
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		for i := 0; i < len(tokens); i++ {
			for j := 0; j < len(addresses); j++ {
				buffer.WriteString(addresses[j])
				buffer.WriteString(":")
				if j != 0 {
					if j%2000 == 0 || j == len(addresses)-1 {
						fmt.Println(tokens[i], j)
						l := buffer.String()
						res := l[:len(l)-1]
						buffer.Reset()
						arg := []string{"token", "multi-send", "--home="+*cHome, "--node="+*vNode, "--chain-id="+*chainId, "--from="+*owner, "--amount=10000000000:"+tokens[i], "--to="+res}
						cmd := exec.Command("bnbcli", arg...)
						cmd.Stdout = &stdout
						cmd.Stderr = &stderr
						err := cmd.Run()
						if err != nil {
							fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
							stdout.Reset()
							stderr.Reset()
							time.Sleep(5 * time.Second)
							// multi-send, retry
							fmt.Println("retry ...")
							cmdR := exec.Command("bnbcli", arg...)
							cmdR.Stdout = &stdout
							cmdR.Stderr = &stderr
							err = cmd.Run()
							if err != nil {
								panic(err)
							}
							fmt.Println("retry, OK")
						}
						fmt.Println(stdout.String())
						stdout.Reset()
						stderr.Reset()
						time.Sleep(5 * time.Second)
					}
				}
			}
		}
	}
	tokens = tokens[:len(tokens)-1]
}

func createFolder(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}
}

func emptyFolder(path string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		err = os.RemoveAll(filepath.Join(path, file.Name()))
		if err != nil {
			panic(err)
		}
	}
}

func moveFiles(srcPath string, dstPath string, count int) {
	files, err := ioutil.ReadDir(srcPath)
	if err != nil {
		panic(err)
	}
	for i, file := range files {
		if i < count {
			src := filepath.Join(srcPath, file.Name())
			dst := filepath.Join(dstPath, file.Name())
			err := os.Rename(src, dst)
			if err != nil {
				panic(err)
			}
		}
	}
}

func generateCSV() {
	csvFile, err := os.OpenFile(*csvPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()
	writer := bufio.NewWriter(csvFile)
	files, err := ioutil.ReadDir(*diskCachePath)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		txBytes, err := ioutil.ReadFile(filepath.Join(*diskCachePath, file.Name()))
		if err != nil {
			panic(err)
		}
		hexBytes := make([]byte, len(txBytes) * 2)
		hex.Encode(hexBytes, txBytes)
		line := fmt.Sprintf("%s\n", hexBytes)
		_, err = writer.WriteString(line)
		if err != nil {
			fmt.Println(file.Name(), err)
			continue
		}
	}
	writer.Flush()
}

func main() {

	fmt.Println("-cHome", *cHome)
	fmt.Println("-owner", *owner)
	fmt.Println("-vNode", *vNode)
	fmt.Println("vNodes", vNodes)
	fmt.Println("-chainId", *chainId)
	fmt.Println("-ordersChnBuf", *ordersChnBuf)
	fmt.Println("-createPoolSize", *createPoolSize)
	fmt.Println("-transChnBuf", *transChnBuf)
	fmt.Println("-submitPoolSize", *submitPoolSize)
	fmt.Println("-diskCachePath", *diskCachePath)
	fmt.Println("-userPrefix", *userPrefix)
	fmt.Println("-generateToken", *generateToken)
	fmt.Println("-initialAccount", *initialAccount)
	fmt.Println("-batchSize", *batchSize)
	fmt.Println("-runCreate", *runCreate)
	fmt.Println("-runSubmit", *runSubmit)
	fmt.Println("-submitOnlyPath", *submitOnlyPath)
	fmt.Println("-submitOnlySize", *submitOnlySize)
	fmt.Println("-submitThinkTime", *submitThinkTime)
	fmt.Println("-csvPath", *csvPath)

	myAccounts := generateAccounts(*userPrefix)
	myUsers := myAccounts[0]
	fmt.Println(len(myUsers))
	myAddresses := myAccounts[1]
	fmt.Println(len(myAddresses))

	myTokens := generateTokens(0, 2, *generateToken)
	fmt.Println(myTokens)

	initializeAccounts(myAddresses, myTokens, *initialAccount)

	if *runCreate == true {
		createFolder(*diskCachePath)
		emptyFolder(*diskCachePath)
		// do create task
		sT := time.Now()
		doCreateTask(myTokens, myUsers)
		eT := time.Now()
		elapsedC := eT.Sub(sT)
		fmt.Println("start:", sT)
		fmt.Println("end:", eT)
		fmt.Println("elapsed:", elapsedC)

		if *runSubmit == true {
			// do submit task
			sT := time.Now()
			doSubmitTask()
			eT := time.Now()
			elapsedS := eT.Sub(sT)
			fmt.Println("start:", sT)
			fmt.Println("end:", eT)
			fmt.Println("elapsed:", elapsedS)
			os.Exit(0)
		}
	}

	if *runSubmit == true {
		// do submit task **ONLY**
		createFolder(*submitOnlyPath)
		emptyFolder(*submitOnlyPath)
		moveFiles(*diskCachePath, *submitOnlyPath, *submitOnlySize)

		temp := *diskCachePath
		*diskCachePath = *submitOnlyPath

		sT := time.Now()
		doSubmitTask()
		eT := time.Now()
		elapsedS := eT.Sub(sT)
		fmt.Println("start:", sT)
		fmt.Println("end:", eT)
		fmt.Println("elapsed:", elapsedS)
		fmt.Println("total trans:", *submitOnlySize)

		*diskCachePath = temp
	}

	generateCSV()

}