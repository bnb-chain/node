package types

import abci "github.com/tendermint/tendermint/abci/types"

// Result is the union of ResponseDeliverTx and ResponseCheckTx.
type Result struct {

	// Code is the response code, is stored back on the chain.
	Code ABCICodeType

	// Data is any data returned from the app.
	Data []byte

	// Log is just debug information. NOTE: nondeterministic.
	Log string

	// Tx fee amount and denom.
	FeeAmount int64
	FeeDenom  string

	// Tags are used for transaction indexing and pubsub.
	Tags   Tags
	Events Events
}

// TODO: In the future, more codes may be OK.
func (res Result) IsOK() bool {
	return res.Code.IsOK()
}

func (res Result) GetEvents() []abci.Event {
	events := res.Tags.ToEvents()
	if res.Events != nil {
		events = append(events, res.Events.ToABCIEvents()...)
	}
	return events
}
