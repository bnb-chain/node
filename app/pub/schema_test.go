package pub

import (
	"encoding/json"
	"fmt"
	"github.com/linkedin/goavro"
	"os"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"

	"github.com/binance-chain/node/app/config"
	"github.com/binance-chain/node/common/log"
	orderPkg "github.com/binance-chain/node/plugins/dex/order"
)

// This test ensures schema or AvroOrJsonMsg change are consistent and prevent marshal error in runtime
var testBlock = `
{"ChainID":"bnbchain-1000","CryptoBlock":{"BlockHash":"b42e1f89b9986c441a2de425e3c7ce90859276899f7900a2be5b7a24d2123b7a","ParentHash":"dd444e38f1874993ba92b0bb420b0f534e6333a893066207467ea3b6117dabee","BlockHeight":580,"Timestamp":"2019-09-17T07:00:02.678369Z","TxTotal":28,"NativeBlockMeta":{"LastCommitHash":"5ef145920e6714acc4f9bfbca8b51fb11dd82e9704e9a62cc83f6630d5c8a9a2","DataHash":"06381b06d80d6462899656d14cc7898878bbd1bcc3458fc592aa97947396307a","ValidatorsHash":"a8209794d6638a7cd09cfcd8381eb5568ed9b20cf5e0332bec60629409da9f2f","NextValidatorsHash":"a8209794d6638a7cd09cfcd8381eb5568ed9b20cf5e0332bec60629409da9f2f","ConsensusHash":"294d8fbd0b94b767a7eba9840f299a3586da7fe6b5dead3b7eecba193c400f93","AppHash":"15f60c1ca6b5ef6bcac93a7969424ff1e3e330804127586adc7e18cefbc3e33d","LastResultsHash":"","EvidenceHash":"","ProposerAddress":"bca1ry8p38u46jkrfjfmqdg5kx3ymktshyqrqdp3rz"},"Transactions":[{"TxHash":"A495179A39D033ABC3A0BB95526EDCFFC6256D3EBAE62CB79E09774853774DE6","Fee":"37500BNB","Timestamp":"2019-09-17T07:00:02.678369Z","Inputs":[{"Address":"bnb1lag5vw33q99jp73rs4murl35terycjxay07eyg","Coins":null}],"Outputs":null,"NativeTransaction":{"Source":0,"TxType":"HTLT","TxAsset":"","OrderId":"","Code":0,"Data":"{\"from\":\"bnb1lag5vw33q99jp73rs4murl35terycjxay07eyg\",\"to\":\"bnb16unm97grz9m3snejn9nv80th7eu24d02ux6z5g\",\"recipient_other_chain\":\"\",\"sender_other_chain\":\"\",\"random_number_hash\":\"8e740d3d7c2b9450a311bda08dc53225a791f4993544603e02a6949b8bb7afdb\",\"timestamp\":1568703602,\"amount\":[{\"denom\":\"BNB\",\"amount\":100000000}],\"expected_income\":\"10000:ETH-746\",\"height_span\":500,\"cross_chain\":false}"}}]}}
`

func TestMain(m *testing.M) {
	Logger = log.With("module", "pub")
	Cfg = &config.PublicationConfig{KafkaVersion: "2.1.0"}
	os.Exit(m.Run())
}

func TestExecutionResultsMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "", false)
	trades := trades{
		NumOfMsgs: 1,
		Trades: []*Trade{{
			Id: "42-0", Symbol: "NNB_BNB", Price: 100, Qty: 100,
			Sid: "s-1", Bid: "b-1", TickType: 1,
			Sfee: "BNB:8;ETH:1", Bfee: "BNB:10;BTC:1", SSingleFee: "BNB:8;ETH:1", BSingleFee: "BNB:10;BTC:1",
			SAddr: "s", BAddr: "b", SSrc: 0, BSrc: 0}},
	}
	orders := Orders{
		NumOfMsgs: 3,
		Orders: []*Order{
			{"NNB_BNB", orderPkg.Ack, "b-1", "", "b", orderPkg.Side.BUY, orderPkg.OrderType.LIMIT, 100, 100, 0, 0, 0, "", 100, 100, orderPkg.TimeInForce.GTE, orderPkg.NEW, "", ""},
			{"NNB_BNB", orderPkg.FullyFill, "b-1", "42-0", "b", orderPkg.Side.BUY, orderPkg.OrderType.LIMIT, 100, 100, 100, 100, 100, "BNB:10;BTC:1", 100, 100, orderPkg.TimeInForce.GTE, orderPkg.NEW, "", "BNB:10;BTC:1"},
			{"NNB_BNB", orderPkg.FullyFill, "s-1", "42-0", "s", orderPkg.Side.SELL, orderPkg.OrderType.LIMIT, 100, 100, 100, 100, 100, "BNB:8;ETH:1", 99, 99, orderPkg.TimeInForce.GTE, orderPkg.NEW, "", "BNB:8;ETH:1"},
		},
	}
	proposals := Proposals{
		NumOfMsgs: 3,
		Proposals: []*Proposal{
			{1, Succeed},
			{2, Succeed},
			{3, Failed},
		},
	}

	valAddr, _ := sdk.ValAddressFromBech32("bva1e2y8w2rz957lahwy0y5h3w53sm8d78qexkn3rh")
	delAddr, _ := sdk.AccAddressFromBech32("bnb1e2y8w2rz957lahwy0y5h3w53sm8d78qex2jpan")
	stakeUpdates := StakeUpdates{
		NumOfMsgs: 1,
		CompletedUnbondingDelegations: []*CompletedUnbondingDelegation{
			{
				Validator: valAddr,
				Delegator: delAddr,
				Amount:    Coin{"BNB", 100000000000},
			},
		},
	}
	msg := ExecutionResults{
		Height:       42,
		Timestamp:    100,
		NumOfMsgs:    8,
		Trades:       trades,
		Orders:       orders,
		Proposals:    proposals,
		StakeUpdates: stakeUpdates,
	}
	_, err := publisher.marshal(&msg, executionResultTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBooksMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "", false)
	book := OrderBookDelta{"NNB_BNB", []PriceLevel{{100, 100}}, []PriceLevel{{100, 100}}}
	msg := Books{42, 100, 1, []OrderBookDelta{book}}
	_, err := publisher.marshal(&msg, booksTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAccountsMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "", false)
	accs := []Account{{"b-1", "BNB:1000;BTC:10", 0, []*AssetBalance{{Asset: "BNB", Free: 100}}}}
	msg := Accounts{42, 2, accs}
	_, err := publisher.marshal(&msg, accountsTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBlockFeeMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "", false)
	msg := BlockFee{1, "BNB:1000;BTC:10", []string{"bnc1", "bnc2", "bnc3"}}
	_, err := publisher.marshal(&msg, blockFeeTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTransferMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "", false)
	msg := Transfers{42, 20, 1000, []Transfer{{TxHash: "123456ABCDE", Memo: "1234", From: "", To: []Receiver{{"bnc1", []Coin{{"BNB", 100}, {"BTC", 100}}}, {"bnc2", []Coin{{"BNB", 200}, {"BTC", 200}}}}}}}
	_, err := publisher.marshal(&msg, transferTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBlockMarsha(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "", false)
	var msg Block
	err := json.Unmarshal([]byte(testBlock), &msg)
	assert.NoError(t, err)
	_, err = publisher.marshal(&msg, blockTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCrossTransferMarsha(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "", false)
	msg := CrossTransfers{
		Height:    10,
		Num:       2,
		Timestamp: time.Now().Unix(),
		Transfers: []CrossTransfer{
			{TxHash: "xxxx", ChainId: "rialto", Type: "xx", From: "xxxx", RelayerFee: 1, Denom: "BNB", To: []CrossReceiver{{Addr: "xxxx", Amount: 100}}},
			{TxHash: "xxxx", ChainId: "rialto", Type: "xx", From: "xxxx", RelayerFee: 0, Denom: "BNB", To: []CrossReceiver{{Addr: "xxxx", Amount: 100}}},
		},
	}
	_, err := publisher.marshal(&msg, crossTransferTpe)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSideProposalMarsha(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "", false)
	msg := SideProposals{
		Height:    10,
		NumOfMsgs: 2,
		Timestamp: time.Now().Unix(),
		Proposals: []*SideProposal{
			{Id: 100, ChainId: "rialto", Status: Succeed},
			{Id: 101, ChainId: "rialto", Status: Failed},
		},
	}
	_, err := publisher.marshal(&msg, sideProposalType)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStakingMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "", false)
	valAddr, _ := sdk.ValAddressFromBech32("bva1e2y8w2rz957lahwy0y5h3w53sm8d78qexkn3rh")
	delAddr, _ := sdk.AccAddressFromBech32("bnb1e2y8w2rz957lahwy0y5h3w53sm8d78qex2jpan")

	dels := make(map[string][]*Delegation)
	dels["chain-id-1"] = []*Delegation{{
		DelegatorAddr: delAddr,
		ValidatorAddr: valAddr,
		Shares:        sdk.NewDecWithoutFra(1),
	}}

	removedVals := make(map[string][]sdk.ValAddress)
	removedVals["chain-id-1"] = []sdk.ValAddress{sdk.ValAddress(valAddr)}

	msg := StakingMsg{
		NumOfMsgs: 42, Height: 20, Timestamp: 1000,
		Validators: []*Validator{{
			FeeAddr:         delAddr,
			OperatorAddr:    valAddr,
			Status:          1,
			DelegatorShares: sdk.NewDecWithoutFra(10000),
		}},
		RemovedValidators: removedVals,
		Delegations:       dels,
		DelegateEvents:    map[string][]*DelegateEvent{"chain-id-1": {&DelegateEvent{delAddr, valAddr, Coin{Denom: "BNB", Amount: 99999999}, "0xadkjgege"}}},
		ElectedValidators: map[string][]*Validator{"chain-id-1": {&Validator{
			FeeAddr:         delAddr,
			OperatorAddr:    valAddr,
			Status:          1,
			DelegatorShares: sdk.NewDecWithoutFra(10000),
		}}},
	}
	bz, err := publisher.marshal(&msg, stakingTpe)
	if err != nil {
		t.Fatal(err)
	}

	codec, err := goavro.NewCodec(stakingSchema)
	native, _, err := codec.NativeFromBinary(bz)
	fmt.Printf("%v", native)
}

func TestSlashMarshaling(t *testing.T) {
	publisher := NewKafkaMarketDataPublisher(Logger, "", false)
	valAddr, _ := sdk.ValAddressFromBech32("bva1e2y8w2rz957lahwy0y5h3w53sm8d78qexkn3rh")
	submitterAddr, _ := sdk.AccAddressFromBech32("bnb1e2y8w2rz957lahwy0y5h3w53sm8d78qex2jpan")
	slash := make(map[string][]*Slash)
	slashItem := &Slash{
		Validator:        valAddr,
		InfractionType:   1,
		InfractionHeight: 100,
		JailUtil:         100000,
		SlashAmount:      100,
		ToFeePool:        10,
		Submitter:        submitterAddr,
		SubmitterReward:  80,
		ValidatorsCompensation: []*AllocatedAmt{{
			Address: submitterAddr.String(),
			Amount:  10,
		}},
	}
	slash["chain-id-1"] = []*Slash{slashItem}

	msg := SlashMsg{
		NumOfMsgs: 1,
		Height:    100,
		Timestamp: 100000,
		SlashData: slash,
	}
	bz, err := publisher.marshal(&msg, slashingTpe)
	if err != nil {
		t.Fatal(err)
	}

	codec, err := goavro.NewCodec(slashingSchema)
	native, _, err := codec.NativeFromBinary(bz)
	fmt.Printf("%v", native)
}
