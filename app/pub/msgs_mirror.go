package pub

import "fmt"

type Mirror struct {
	TxHash      string
	ChainId     string
	Type        string
	RelayerFee  int64
	Sender      string
	Contract    string
	BEP20Name   string
	BEP20Symbol string
	BEP2Symbol  string
	TotalSupply int64
	Decimals    int
	Fee         int64
}

func (msg Mirror) String() string {
	return fmt.Sprintf("Mirror: txHash: %s, sender: %s, bep2Symbol: %s", msg.TxHash, msg.Sender, msg.BEP2Symbol)
}

func (msg Mirror) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["txHash"] = msg.TxHash
	native["chainId"] = msg.ChainId
	native["type"] = msg.Type
	native["relayerFee"] = msg.RelayerFee
	native["sender"] = msg.Sender
	native["contract"] = msg.Contract
	native["bep20Name"] = msg.BEP20Name
	native["bep20Symbol"] = msg.BEP20Symbol
	native["bep2Symbol"] = msg.BEP2Symbol
	native["totalSupply"] = msg.TotalSupply
	native["totalSupply"] = msg.TotalSupply
	native["decimals"] = msg.Decimals
	native["fee"] = msg.Fee
	return native
}

// deliberated not implemented Ess
type Mirrors struct {
	Height    int64
	Num       int
	Timestamp int64
	Mirrors   []Mirror
}

func (msg Mirrors) String() string {
	return fmt.Sprintf("Mirrors in block %d, num: %d", msg.Height, msg.Num)
}

func (msg Mirrors) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.Height
	mirrors := make([]map[string]interface{}, len(msg.Mirrors), len(msg.Mirrors))
	for idx, t := range msg.Mirrors {
		mirrors[idx] = t.ToNativeMap()
	}
	native["timestamp"] = msg.Timestamp
	native["num"] = msg.Num
	native["mirrors"] = mirrors
	return native
}
