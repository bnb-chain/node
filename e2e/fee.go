package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/bnb-chain/go-sdk/client/rpc"
	sdkTypes "github.com/bnb-chain/go-sdk/common/types"
	"golang.org/x/xerrors"
)

var (
	//c = rpc.NewRPCClient("http://172.22.42.52:27147", sdkTypes.TestNetwork)
	c = rpc.NewRPCClient("http://127.0.0.1:26657", sdkTypes.ProdNetwork)
)

//const sideChainId = "rialto"
const sideChainId = "bsc"

type Validator struct {
	OperatorAddr  sdkTypes.ValAddress   `json:"operator_addr"`
	FeeAddr       sdkTypes.AccAddress   `json:"fee_addr"`
	HexAddr       string                `json:"hex_addr"`
	TotalShares   sdkTypes.Dec          `json:"total_shares"`
	SelfShare     sdkTypes.Dec          `json:"self_share"`
	Balance       sdkTypes.TokenBalance `json:"balance"`
	ComissionRate sdkTypes.Dec          `json:"comission_rate"`
	Status        sdkTypes.BondStatus   `json:"status"`
}

type Data struct {
	TotalTxs       int64       `json:"total_txs"`
	Validators     []Validator `json:"validators"`
	SideValidators []Validator `json:"side_validators"`
}

type SnapShot struct {
	Height int64 `json:"height"`
	NumTxs int64 `json:"num_txs"`
	Data   Data  `json:"data"`
}

func changed(a, b *SnapShot) bool {
	aJson, _ := json.Marshal(a.Data)
	bJson, _ := json.Marshal(b.Data)
	return string(aJson) != string(bJson)
}

func GetSnapshot() (s *SnapShot, err error) {
	s = &SnapShot{}
	block, err := c.Block(nil)
	if err != nil {
		return nil, xerrors.Errorf("get block failed: %v", err)
	}
	s.Height = block.Block.Height
	s.NumTxs = int64(len(block.Block.Txs))
	s.Data.TotalTxs = block.Block.Header.TotalTxs
	// get all validators
	validators, err := c.QueryTopValidators(50)
	if err != nil {
		return nil, xerrors.Errorf("get validators failed: %v", err)
	}
	//log.Printf("validators: %+v", validators)
	for _, v := range validators {
		delegation, err := c.QueryDelegation(v.FeeAddr, v.OperatorAddr)
		if err != nil {
			return nil, xerrors.Errorf("get delegation failed: %v", err)
		}
		//log.Printf("validator: %s", Pretty(v))
		//log.Printf("validator: %+v", v)
		//log.Printf("delegation: %+v", delegation)
		feeAddrBalance, err := c.GetBalance(v.FeeAddr, "BNB")
		if err != nil {
			return nil, xerrors.Errorf("get balance failed: %v", err)
		}
		//log.Printf("feeAddrBalance: %+v", feeAddrBalance)
		validator := Validator{
			OperatorAddr:  v.OperatorAddr,
			FeeAddr:       v.FeeAddr,
			TotalShares:   v.DelegatorShares,
			SelfShare:     delegation.Shares,
			Balance:       *feeAddrBalance,
			ComissionRate: v.Commission.Rate,
			Status:        v.Status,
			HexAddr:       fmt.Sprintf("%X", v.FeeAddr.Bytes()),
		}
		s.Data.Validators = append(s.Data.Validators, validator)
	}
	// get all sidechain validators
	sidechainValidators, err := c.QuerySideChainTopValidators(sideChainId, 50)
	if err != nil {
		return nil, xerrors.Errorf("get sidechain validators failed: %v", err)
	}
	for _, v := range sidechainValidators {
		delegation, err := c.QuerySideChainDelegation(sideChainId, v.FeeAddr, v.OperatorAddr)
		if err != nil {
			return nil, xerrors.Errorf("get delegation failed: %v", err)
		}
		//log.Printf("validator: %+v", v)
		//log.Printf("delegation: %+v", delegation)
		feeAddrBalance, err := c.GetBalance(v.FeeAddr, "BNB")
		if err != nil {
			return nil, xerrors.Errorf("get balance failed: %v", err)
		}
		//log.Printf("feeAddrBalance: %+v", feeAddrBalance)
		validator := Validator{
			OperatorAddr:  v.OperatorAddr,
			FeeAddr:       v.FeeAddr,
			TotalShares:   v.DelegatorShares,
			SelfShare:     delegation.Shares,
			Balance:       *feeAddrBalance,
			ComissionRate: v.Commission.Rate,
			Status:        v.Status,
			HexAddr:       fmt.Sprintf("%X", v.FeeAddr.Bytes()),
		}
		s.Data.SideValidators = append(s.Data.SideValidators, validator)
	}
	return s, nil
}

func saveSnapshot(s *SnapShot, path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return xerrors.Errorf("marshal snapshot failed: %v", err)
	}
	return ioutil.WriteFile(fmt.Sprintf("%s/%d.json", path, s.Height), data, 0644)
}

//nolint
func TestFee() error {
	latestSnap := &SnapShot{}
	for {
		snapshot, err := GetSnapshot()
		if err != nil {
			return xerrors.Errorf("get snapshot failed: %v", err)
		}
		changed := changed(latestSnap, snapshot)
		log.Printf("height: %v, TxNum: %v, changed: %v", snapshot.Height, snapshot.NumTxs, changed)
		if changed {
			log.Printf("snapshot: %s", Pretty(snapshot))
			err = saveSnapshot(snapshot, "test")
			if err != nil {
				return xerrors.Errorf("save snapshot failed: %v", err)
			}
		}
		latestSnap = snapshot
	}
}
