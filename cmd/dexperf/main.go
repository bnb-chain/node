package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/ratelimit"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	txbuilder "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/spf13/viper"
	abci "github.com/tendermint/tendermint/abci/types"
	rpcclient "github.com/tendermint/tendermint/rpc/client"

	"github.com/binance-chain/node/common/client"
	"github.com/binance-chain/node/common/tx"
	"github.com/binance-chain/node/common/types"
	"github.com/binance-chain/node/plugins/dex"
	"github.com/binance-chain/node/plugins/dex/order"
	"github.com/binance-chain/node/plugins/tokens"
	"github.com/binance-chain/node/wire"
)

const (
	retry      = 25
	stime      = 2000
	createTask = 1
	submitTask = 2
	buy        = 1
	sell       = 2
)

var home *string
var node *string
var chainId *string
var owner *string
var userPrefix *string
var votingTime *int
var batchSize *int
var generateToken *bool
var initiateAccount *bool
var runCreate *bool
var createChnBuf *int
var createPoolSize *int
var createPath *string
var runSubmit *bool
var submitChnBuf *int
var submitPoolSize *int
var submitAsync *bool
var submitPath *string
var submitPause *int
var csvPath *string

type DEXCreate struct {
	ctx     context.CLIContext
	txBldr  txbuilder.TxBuilder
	addr    sdk.AccAddress
	side    int8
	symbol  string
	price   int64
	qty     int64
	tifCode int8
}
type DEXSubmit struct {
	ctx     context.CLIContext
	txBldr  txbuilder.TxBuilder
	txBytes []byte
}

type sequence struct {
	m      sync.Mutex
	seqMap map[string]int64
}
type txhash struct {
	m     sync.Mutex
	trans []string
}

var createChn chan DEXCreate
var submitChn chan DEXSubmit

var clientSeq sequence
var hashReturned txhash

var nodes []string
var rpcs []*rpcclient.HTTP

// used as throughput controller
// 1000 means 1000 calls per sec
var rl = ratelimit.New(1000)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("tbnb", "bnbp")
	config.SetBech32PrefixForValidator("bva", "bvap")
	config.SetBech32PrefixForConsensusNode("bca", "bcap")
	config.Seal()
	home = flag.String("home", "/home/test/.bnbcli", "bnbcli --home")
	node = flag.String("node", "0.0.0.0:26657", "bnbcli --node")
	chainId = flag.String("chainId", "chain-bnb", "bnbcli --chain-id")
	owner = flag.String("owner", "test", "chain's master user")
	userPrefix = flag.String("userPrefix", "node2_user", "user prefix")
	votingTime = flag.Int("votingTime", 10, "voting time in sec")
	batchSize = flag.Int("batchSize", 1, "# of create/submit tasks")
	generateToken = flag.Bool("generateToken", false, "if to generate tokens")
	initiateAccount = flag.Bool("initiateAccount", false, "if to initiate accounts")
	runCreate = flag.Bool("runCreate", false, "if to run create task")
	createChnBuf = flag.Int("createChnBuf", 1, "create channel buffer size")
	createPoolSize = flag.Int("createPoolSize", 1, "create pool size")
	createPath = flag.String("createPath", "/home/test/create", "create path")
	runSubmit = flag.Bool("runSubmit", false, "if to run submit task")
	submitChnBuf = flag.Int("submitChnBuf", 1, "submit channel buffer size")
	submitPoolSize = flag.Int("submitPoolSize", 1, "submit pool size")
	submitAsync = flag.Bool("submitAsync", false, "submit in async mode")
	submitPath = flag.String("submitPath", "/home/test/submit", "submit path")
	submitPause = flag.Int("submitPause", 1000, "rate limit")
	csvPath = flag.String("csvPath", "/home/test", "csv path")
	flag.Parse()
	createChn = make(chan DEXCreate, *createChnBuf)
	submitChn = make(chan DEXSubmit, *submitChnBuf)
	clientSeq = sequence{seqMap: make(map[string]int64)}
	hashReturned = txhash{trans: make([]string, 0, 0)}
	nodes = strings.Split(*node, ",")
	rpcs = make([]*rpcclient.HTTP, len(nodes))
	for i, v := range nodes {
		rpcs[i] = rpcclient.NewHTTP(v, "/websocket")
	}
	rl = ratelimit.New(*submitPause)
}

var accToAdd map[string]string
var sortKeys []string
var accToIp map[string]string

func main() {
	fmt.Println("-home", *home)
	fmt.Println("-node", *node)
	fmt.Println("-chainId", *chainId)
	fmt.Println("-owner", *owner)
	fmt.Println("-userPrefix", *userPrefix)
	fmt.Println("-votingTime", *votingTime)
	fmt.Println("-batchSize", *batchSize)
	fmt.Println("-generateToken", *generateToken)
	fmt.Println("-initiateAccount", *initiateAccount)
	fmt.Println("-runCreate", *runCreate)
	fmt.Println("-createChnBuf", *createChnBuf)
	fmt.Println("-createPoolSize", *createPoolSize)
	fmt.Println("-createPath", *createPath)
	fmt.Println("-runSubmit", *runSubmit)
	fmt.Println("-submitChnBuf", *submitChnBuf)
	fmt.Println("-submitPoolSize", *submitPoolSize)
	fmt.Println("-submitPath", *submitPath)
	fmt.Println("-submitPause", *submitPause)
	fmt.Println("-csvPath", *csvPath)

	lookupAccounts()

	tokens := generateTokens(0, 9, *generateToken)
	if tokens == nil {
		path := filepath.Join(*csvPath, "tokens.csv")
		file, err := os.Open(path)
		defer file.Close()
		if err != nil {
			panic(err)
		}
		s := bufio.NewScanner(file)
		for s.Scan() {
			tokens = append(tokens, s.Text())
		}
		fmt.Println("issued tokens:", tokens)
	}
	initializeAccounts(tokens, *initiateAccount)

	if *runCreate == true {
		createFolder(*createPath)
		emptyFolder(*createPath)
		sT := time.Now()
		doCreateTask(tokens)
		eT := time.Now()
		elapsed := eT.Sub(sT)
		fmt.Println("start:", sT)
		fmt.Println("end:", eT)
		fmt.Println("elapsed:", elapsed)
	}

	if *runSubmit == true {
		createFolder(*submitPath)
		emptyFolder(*submitPath)
		moveFiles(*createPath, *submitPath, *batchSize)
		sT := time.Now()
		doSubmitTask()
		eT := time.Now()
		elapsed := eT.Sub(sT)
		fmt.Println("start:", sT)
		fmt.Println("end:", eT)
		fmt.Println("elapsed:", elapsed)
	}

	// to generate data for AP and QS test
	save_txhash()
	save_hextx()
}

func execCommand(name string, arg ...string) *bytes.Buffer {
	var err error
	for i := 0; i < retry; i++ {
		fmt.Println("running round", ":", i, name, arg)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd := exec.Command(name, arg...)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err), stderr.String())
			continue
		}
		return &stdout
	}
	panic(err)
}

func lookupAccounts() {
	stdout := execCommand("tbnbcli", "--home="+*home, "keys", "list")
	expr := "(" + *userPrefix + "[\\d]+).+(tbnb.+).+bnb"
	res, err := regexp.Compile(expr)
	if err != nil {
		panic(err)
	}
	accToAdd = make(map[string]string)
	matched := res.FindAllStringSubmatch(stdout.String(), -1)
	if matched != nil {
		for _, v := range matched {
			accToAdd[v[1]] = v[2]
		}
	} else {
		panic("no account found")
	}
	sortKeys = make([]string, 0, len(accToAdd))
	for key, _ := range accToAdd {
		sortKeys = append(sortKeys, key)
	}
	sort.Strings(sortKeys)
	n, err := strconv.ParseInt((*userPrefix)[4:5], 10, 0)
	if err != nil {
		panic(err)
	}
	accToIp = make(map[string]string)
	index := 0
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			for c := 0; c < 256; c++ {
				ip := fmt.Sprintf("%d.%d.%d.%d", n, a, b, c)
				accToIp[sortKeys[index]] = ip
				index++
				if index == len(sortKeys) {
					return
				}
			}
		}
	}
}

func generateTokens(sIndex int, eIndex int, flag bool) []string {
	var tokens []string
	if flag == true {
		path := filepath.Join(*csvPath, "tokens.csv")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		writer := bufio.NewWriter(file)
		for sIndex <= eIndex {
			var token string
			if sIndex < 10 {
				token = fmt.Sprintf("X0%d", sIndex)
			} else if sIndex >= 10 && sIndex < 100 {
				token = fmt.Sprintf("X%d", sIndex)
			} else {
				panic("token index was out of range")
			}
			type toJSON struct {
				Height   int64
				TxHash   string
				Response abci.ResponseDeliverTx
			}
			issueRep := execCommand("tbnbcli", "token", "issue", "--home="+*home, "--node="+*node, "--token-name="+token, "--symbol="+token, "--total-supply=20000000000000000", "--from="+*owner, "--chain-id="+*chainId, "--json=true")
			issueJson := toJSON{}
			err = MakeCodec().UnmarshalJSON(issueRep.Bytes(), &issueJson)
			if err != nil {
				panic(err)
			}
			expr := "^Msg 0: Issued (.+)$"
			res, err := regexp.Compile(expr)
			if err != nil {
				panic(err)
			}
			matched := res.FindStringSubmatch(issueJson.Response.Log)
			if matched != nil {
				token = matched[1]
				writer.WriteString(token + "\n")
				writer.Flush()
			} else {
				panic("token issue failed")
			}
			time.Sleep(stime * time.Millisecond)
			expireTime := strconv.FormatInt(time.Now().Unix()+3600, 10)
			proposalRep := execCommand("tbnbcli", "gov", "submit-list-proposal", "--home="+*home, "--node="+*node, "--chain-id="+*chainId, "--from="+*owner, "--deposit=200000000000:BNB", "--base-asset-symbol="+token, "--quote-asset-symbol=BNB", "--init-price=100000000", "--title="+token+":BNB", "--description="+token+":BNB", "--expire-time="+expireTime, "--json=true")
			proposalJson := toJSON{}
			err = MakeCodec().UnmarshalJSON(proposalRep.Bytes(), &proposalJson)
			if err != nil {
				panic(err)
			}
			var pid string
			for _, tag := range proposalJson.Response.Tags {
				if string(tag.Key) == "proposal-id" {
					pid = string(tag.Value)
				}
			}
			time.Sleep(stime * time.Millisecond)
			execCommand("tbnbcli", "gov", "vote", "--home="+*home, "--node="+*node, "--chain-id="+*chainId, "--from="+*owner, "--proposal-id="+pid, "--option=yes")
			time.Sleep(time.Duration(*votingTime) * time.Second)
			execCommand("tbnbcli", "dex", "list", "--home="+*home, "--node="+*node, "--base-asset-symbol="+token, "--quote-asset-symbol=BNB", "--init-price=100000000", "--from="+*owner, "--chain-id="+*chainId, "--proposal-id="+pid)
			time.Sleep(stime * time.Millisecond)
			tokens = append(tokens, token)
			sIndex++
		}
	}
	return tokens
}

func initializeAccounts(tokens []string, flag bool) {
	tokens = append(tokens, "BNB")
	if flag == true {
		type Transfer struct {
			To     string `json:to`
			Amount string `json:amount`
		}
		b := 0
		transfers := make([]Transfer, 2000)
		for i, key := range sortKeys {
			var buffer bytes.Buffer
			for j, token := range tokens {
				buffer.WriteString("50000000000:")
				buffer.WriteString(token)
				if j != (len(tokens) - 1) {
					buffer.WriteString(",")
				}
			}
			transfers[b] = Transfer{
				To:     accToAdd[key],
				Amount: buffer.String(),
			}
			b++
			if b == len(transfers) || i == len(sortKeys)-1 {
				b = 0
				bytes, err := json.Marshal(transfers)
				if err != nil {
					panic(err)
				}
				path := filepath.Join(*csvPath, fmt.Sprintf("transfers_%d.data", i))
				file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					panic(err)
				}
				defer file.Close()
				writer := bufio.NewWriter(file)
				writer.Write(bytes)
				writer.Flush()
				execCommand("tbnbcli", "token", "multi-send", "--home="+*home, "--node="+*node, "--chain-id="+*chainId, "--from="+*owner, "--transfers-file", path)
				time.Sleep(stime * time.Millisecond)
			}
		}
	}
	tokens = tokens[:len(tokens)-1]
}

func createFolder(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0644)
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

func execute(poolSize int, mode int) {
	var wg sync.WaitGroup
	wg.Add(poolSize)
	for i := 0; i < poolSize; i++ {
		if mode == createTask {
			go create(&wg, &clientSeq)
		}
		if mode == submitTask {
			go submit(&wg, &hashReturned)
		}
	}
	wg.Wait()
}

func doCreateTask(tokens []string) {
	go allocateCreate(tokens)
	execute(*createPoolSize, createTask)
}

func allocateCreate(tokens []string) {
	var buyPrices []int64 = generatePrices(*batchSize, 0.00)
	var sellPrices []int64 = generatePrices(*batchSize, 0.01)
	var largeBuyers []int
	createIndex := 0
	nameIndex := 0
	for i := 0; i < *batchSize; i++ {
		for j := 0; j < len(tokens); j++ {
			symbol := fmt.Sprintf("%s_BNB", tokens[j])
			fmt.Printf("allocating #%d\n", createIndex)
			if largeBuyers != nil && isLargeBuyer(nameIndex, largeBuyers) {
				createChn <- buildC(sortKeys[nameIndex], buy, symbol, 9990000, 10000000000, "GTE")
			} else {
				createChn <- buildC(sortKeys[nameIndex], buy, symbol, buyPrices[i], 100000000, "GTE")
			}
			createIndex++
			if createIndex == *batchSize {
				close(createChn)
				return
			}
			nameIndex++
			if nameIndex == len(sortKeys) {
				nameIndex = 0
				largeBuyers = make([]int, len(sortKeys)/1000)
				generateLargeBuyers(largeBuyers)
			}
			createChn <- buildC(sortKeys[nameIndex], sell, symbol, sellPrices[i], 100000000, "GTE")
			createIndex++
			if createIndex == *batchSize {
				close(createChn)
				return
			}
			nameIndex++
			if nameIndex == len(sortKeys) {
				nameIndex = 0
				largeBuyers = make([]int, len(sortKeys)/1000)
				generateLargeBuyers(largeBuyers)
			}
		}
	}
}

func generatePrices(noOfPrices int, margin float64) []int64 {
	rand.Seed(1)
	prices := make([]int64, noOfPrices)
	for i := 0; i < noOfPrices; i++ {
		f := rand.Float64()/10 + margin
		s := fmt.Sprintf("%.4f", f)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			panic(err)
		}
		prices[i] = int64(f*10000) * 10000
	}
	return prices
}

func generateLargeBuyers(largeBuyers []int) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < len(largeBuyers); i++ {
		largeBuyers[i] = rand.Intn(len(sortKeys))
	}
	sort.Ints(largeBuyers)
}

func isLargeBuyer(index int, largeBuyers []int) bool {
	for _, v := range largeBuyers {
		if v == index {
			return true
		}
	}
	return false
}

func buildC(from string, side int8, symbol string, price int64, qty int64, tif string) DEXCreate {
	cdc := MakeCodec()
	viper.Set("home", fmt.Sprintf("%s", *home))
	viper.Set("node", nodes[0])
	viper.Set("chain-id", *chainId)
	viper.Set("from", from)
	viper.Set("trust-node", true)
	ctx, txBldr := client.PrepareCtx(cdc)
	ctx.Client = rpcs[0]
	addr, err := ctx.GetFromAddress()
	if err != nil {
		panic(err)
	}
	accNum, err := ctx.GetAccountNumber(addr)
	if err != nil {
		panic(err)
	}
	txBldr = txBldr.WithAccountNumber(accNum)
	tifCode, err := order.TifStringToTifCode(tif)
	if err != nil {
		panic(err)
	}
	return DEXCreate{ctx, txBldr, addr, side, symbol, price, qty, tifCode}
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

func create(wg *sync.WaitGroup, s *sequence) {
	for item := range createChn {
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
			Sender:      item.addr,
			Id:          id,
			Symbol:      item.symbol,
			OrderType:   order.OrderType.LIMIT,
			Side:        item.side,
			Price:       item.price,
			Quantity:    item.qty,
			TimeInForce: order.TimeInForce.GTE,
		}
		msg.TimeInForce = item.tifCode
		msgs := []sdk.Msg{msg}
		ssMsg := txbuilder.StdSignMsg{
			ChainID:       item.txBldr.ChainID,
			AccountNumber: item.txBldr.AccountNumber,
			Sequence:      item.txBldr.Sequence,
			Memo:          item.txBldr.Memo,
			Msgs:          msgs,
		}
		keybase, err := keys.GetKeyBaseFromDir(*home)
		if err != nil {
			fmt.Println(err)
			continue
		}
		sigBytes, pubkey, err := keybase.Sign(name, "1qaz2wsx", ssMsg.Bytes())
		if err != nil {
			fmt.Println(err)
			continue
		}
		sig := auth.StdSignature{
			AccountNumber: ssMsg.AccountNumber,
			Sequence:      ssMsg.Sequence,
			PubKey:        pubkey,
			Signature:     sigBytes,
		}
		txBytes, err := item.txBldr.Codec.MarshalBinaryLengthPrefixed(auth.NewStdTx(ssMsg.Msgs, []auth.StdSignature{sig}, ssMsg.Memo, ssMsg.Source, ssMsg.Data))
		if err != nil {
			fmt.Printf("failed to sign tran: %v\n", err)
			continue
		}
		ts := fmt.Sprintf("%d", time.Now().UnixNano())
		file := filepath.Join(*createPath, ts+"_"+name)
		fmt.Println("Acc-", item.txBldr.AccountNumber, "signed tran saved,", file)
		err = ioutil.WriteFile(file, txBytes, 0644)
		if err != nil {
			fmt.Println(err)
			continue
		}
		s.m.Lock()
		s.seqMap[name] = seq + 1
		s.m.Unlock()
	}
	wg.Done()
}

func doSubmitTask() {
	go allocateSubmit()
	execute(*submitPoolSize, submitTask)
}

func allocateSubmit() {
	expr := "_(" + *userPrefix + "[\\d]+)$"
	res, err := regexp.Compile(expr)
	if err != nil {
		panic(err)
	}
	files, err := ioutil.ReadDir(*submitPath)
	if err != nil {
		panic(err)
	}
	nodeIndex := 0
	userNodeMap := make(map[string]int)
	for i, file := range files {
		matched := res.FindStringSubmatch(file.Name())
		if matched != nil {
			tran, err := ioutil.ReadFile(filepath.Join(*submitPath, file.Name()))
			if err != nil {
				panic(err)
			}
			fmt.Printf("allocate tran #%d\n", i)
			_, hasKey := userNodeMap[matched[1]]
			if hasKey == false {
				userNodeMap[matched[1]] = nodeIndex
			}
			submitChn <- buildS(userNodeMap[matched[1]], tran)
			nodeIndex++
			if nodeIndex == len(nodes) {
				nodeIndex = 0
			}
		} else {
			panic("invalid filename")
		}
	}
	close(submitChn)
}

func buildS(index int, txBytes []byte) DEXSubmit {
	cdc := MakeCodec()
	viper.Set("node", nodes[index])
	viper.Set("chain-id", *chainId)
	ctx, txBldr := client.PrepareCtx(cdc)
	ctx.Client = rpcs[index]
	return DEXSubmit{ctx, txBldr, txBytes}
}

func submit(wg *sync.WaitGroup, txh *txhash) {
	for item := range submitChn {
		// Take() will sleep until the following
		// goroutine can execute
		rl.Take()
		if *submitAsync == true {
			go async(item.ctx, item.txBytes, txh)
		} else {
			async(item.ctx, item.txBytes, txh)
		}
	}
	wg.Done()
}

func async(ctx context.CLIContext, txBytes []byte, txh *txhash) {
	defer doRecover()
	res, err := ctx.BroadcastTxAsync(txBytes)
	if err != nil {
		fmt.Println(err)
	}
	str := res.Hash.String()
	txh.m.Lock()
	txh.trans = append(txh.trans, str)
	txh.m.Unlock()
	fmt.Println("tran hash:", str)
}

func doRecover() {
	if r := recover(); r != nil {
		fmt.Println("recoved from", r)
		debug.PrintStack()
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

func save_txhash() {
	if len(hashReturned.trans) > 0 {
		path := filepath.Join(*csvPath, "txhash.csv")
		csvFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}
		defer csvFile.Close()
		writer := bufio.NewWriter(csvFile)
		for _, tran := range hashReturned.trans {
			_, err = writer.WriteString(tran + "\n")
			if err != nil {
				continue
			}
		}
		writer.Flush()
	}
}

func save_hextx() {
	path := filepath.Join(*csvPath, "trans.csv")
	csvFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()
	writer := bufio.NewWriter(csvFile)
	expr := "_(" + *userPrefix + "[\\d]+)$"
	res, err := regexp.Compile(expr)
	if err != nil {
		panic(err)
	}
	files, err := ioutil.ReadDir(*createPath)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		matched := res.FindStringSubmatch(file.Name())
		if matched != nil {
			ip, _ := accToIp[matched[1]]
			txBytes, err := ioutil.ReadFile(filepath.Join(*createPath, file.Name()))
			if err != nil {
				panic(err)
			}
			hexBytes := make([]byte, len(txBytes)*2)
			hex.Encode(hexBytes, txBytes)
			line := fmt.Sprintf("%s|%s|%s\n", accToAdd[matched[1]], ip, hexBytes)
			_, err = writer.WriteString(line)
			if err != nil {
				continue
			}
		}
	}
	writer.Flush()
}
