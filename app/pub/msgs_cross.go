package pub

import "fmt"

type CrossReceiver struct {
	Addr   string
	Amount int64
}

func (msg CrossReceiver) String() string {
	return fmt.Sprintf("Transfer receiver %s get coin %d", msg.Addr, msg.Amount)
}

func (msg CrossReceiver) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["addr"] = msg.Addr
	native["amount"] = msg.Amount
	return native
}

type CrossTransfer struct {
	TxHash  string
	ChainId string
	RelayerFee int64
	Type    string
	From    string
	Denom   string
	Contract string
	Decimals  int
	To      []CrossReceiver
}

func (msg CrossTransfer) String() string {
	return fmt.Sprintf("CrossTransfer: txHash: %s, from: %s, to: %v", msg.TxHash, msg.From, msg.To)
}

func (msg CrossTransfer) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["txhash"] = msg.TxHash
	native["type"] = msg.Type
	native["chainid"] = msg.ChainId
	native["from"] = msg.From
	native["denom"] = msg.Denom
	native["contract"] = msg.Contract
	native["decimals"] = msg.Decimals
	native["relayerFee"] =  msg.RelayerFee
	to := make([]map[string]interface{}, len(msg.To), len(msg.To))
	for idx, t := range msg.To {
		to[idx] = t.ToNativeMap()
	}
	native["to"] = to
	return native
}

// deliberated not implemented Ess
type CrossTransfers struct {
	Height    int64
	Num       int
	Timestamp int64
	Transfers []CrossTransfer
}

func (msg CrossTransfers) String() string {
	return fmt.Sprintf("CrossTransfers in block %d, num: %d", msg.Height, msg.Num)
}

func (msg CrossTransfers) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.Height
	transfers := make([]map[string]interface{}, len(msg.Transfers), len(msg.Transfers))
	for idx, t := range msg.Transfers {
		transfers[idx] = t.ToNativeMap()
	}
	native["timestamp"] = msg.Timestamp
	native["num"] = msg.Num
	native["transfers"] = transfers
	return native
}
