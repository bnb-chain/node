package types

// Handler defines the core of the state transition function of an application.
type Handler func(ctx Context, msg Msg) Result

// Enum mode for app.runTx
type RunTxMode uint8

const (
	// Check a transaction
	RunTxModeCheck RunTxMode = iota
	// Simulate a transaction
	RunTxModeSimulate RunTxMode = iota
	// Deliver a transaction
	RunTxModeDeliver RunTxMode = iota

	// ReCheck a transaction
	RunTxModeReCheck RunTxMode = iota

	// Check a transaction after PreCheck
	RunTxModeCheckAfterPre RunTxMode = iota

	// Deliver a transaction after PreDeliver
	RunTxModeDeliverAfterPre RunTxMode = iota
)

// AnteHandler authenticates transactions, before their internal messages are handled.
// If newCtx.IsZero(), ctx is used instead.
type AnteHandler func(ctx Context, tx Tx,
	runTxMode RunTxMode) (newCtx Context, result Result, abort bool)

type PreChecker func(ctx Context, txBytes []byte, tx Tx) Result
