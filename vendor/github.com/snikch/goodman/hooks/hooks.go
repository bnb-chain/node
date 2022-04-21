package hooks

import trans "github.com/snikch/goodman/transaction"

type (
	// Callback is a func type that accepts a Transaction pointer.
	Callback func(*trans.Transaction)
	// AllCallback is a func type that accepts a slice of Transaction pointers.
	AllCallback func([]*trans.Transaction)
)

// Hooks is responsible for storing lifecycle callbacks.
type Hooks struct {
	beforeAll            []AllCallback
	beforeEach           []Callback
	before               map[string][]Callback
	beforeEachValidation []Callback
	beforeValidation     map[string][]Callback
	after                map[string][]Callback
	afterEach            []Callback
	afterAll             []AllCallback
}

// NewHooks returns a new Hooks instance with all callback fields initialized.
func NewHooks() *Hooks {
	return &Hooks{
		beforeAll:            []AllCallback{},
		beforeEach:           []Callback{},
		before:               map[string][]Callback{},
		beforeEachValidation: []Callback{},
		beforeValidation:     map[string][]Callback{},
		after:                map[string][]Callback{},
		afterEach:            []Callback{},
		afterAll:             []AllCallback{},
	}
}

// BeforeAll adds a callback function to be called before the entire test suite.
func (h *Hooks) BeforeAll(fn AllCallback) {
	h.beforeAll = append(h.beforeAll, fn)
}

// BeforeEach adds a callback function to be called before each transaction.
func (h *Hooks) BeforeEach(fn Callback) {
	h.beforeEach = append(h.beforeEach, fn)
}

// Before adds a callback function to be called before a named transaction.
func (h *Hooks) Before(name string, fn Callback) {
	if _, ok := h.before[name]; !ok {
		h.before[name] = []Callback{}
	}
	h.before[name] = append(h.before[name], fn)
}

// BeforeEachValidation adds a callback function to be called before each transaction.
func (h *Hooks) BeforeEachValidation(fn Callback) {
	h.beforeEachValidation = append(h.beforeEachValidation, fn)
}

// BeforeValidation adds a callback function to be called before a named transaction.
func (h *Hooks) BeforeValidation(name string, fn Callback) {
	if _, ok := h.beforeValidation[name]; !ok {
		h.beforeValidation[name] = []Callback{}
	}
	h.beforeValidation[name] = append(h.beforeValidation[name], fn)
}

// After adds a callback function to be called before a named transaction.
func (h *Hooks) After(name string, fn Callback) {
	if _, ok := h.after[name]; !ok {
		h.after[name] = []Callback{}
	}
	h.after[name] = append(h.after[name], fn)
}

// AfterEach adds a callback function to be called before each transaction.
func (h *Hooks) AfterEach(fn Callback) {
	h.afterEach = append(h.afterEach, fn)
}

// AfterAll adds a callback function to be called before the entire test suite.
func (h *Hooks) AfterAll(fn AllCallback) {
	h.afterAll = append(h.afterAll, fn)
}

// Hooks is responsible for running lifecycle callbacks.
type HooksRunner struct {
	hooks *Hooks
}

// NewHooksRunner returns a new HooksRunner instance with a given hooks structure.
func NewHooksRunner(h *Hooks) *HooksRunner {
	return &HooksRunner{
		hooks: h,
	}
}

func (h *HooksRunner) RunBeforeAll(args []*trans.Transaction, reply *[]*trans.Transaction) error {
	*reply = args
	for _, cb := range h.hooks.beforeAll {
		cb(args)
	}
	return nil
}

func (h *HooksRunner) RunBeforeEach(args trans.Transaction, reply *trans.Transaction) error {
	*reply = args
	for _, cb := range h.hooks.beforeEach {
		cb(reply)
	}
	return nil
}
func (h *HooksRunner) RunBefore(args trans.Transaction, reply *trans.Transaction) error {
	name := args.Name
	*reply = args
	for _, cb := range h.hooks.before[name] {
		cb(reply)
	}
	return nil
}

func (h *HooksRunner) RunBeforeEachValidation(args trans.Transaction, reply *trans.Transaction) error {
	*reply = args
	for _, cb := range h.hooks.beforeEachValidation {
		cb(reply)
	}
	return nil
}
func (h *HooksRunner) RunBeforeValidation(args trans.Transaction, reply *trans.Transaction) error {
	name := args.Name
	*reply = args
	for _, cb := range h.hooks.beforeValidation[name] {
		cb(reply)
	}
	return nil
}

func (h *HooksRunner) RunAfter(args trans.Transaction, reply *trans.Transaction) error {
	name := args.Name
	*reply = args
	for _, cb := range h.hooks.after[name] {
		cb(reply)
	}
	return nil
}

func (h *HooksRunner) RunAfterEach(args trans.Transaction, reply *trans.Transaction) error {
	*reply = args
	for _, cb := range h.hooks.afterEach {
		cb(reply)
	}
	return nil
}

func (h *HooksRunner) RunAfterAll(args []*trans.Transaction, reply *[]*trans.Transaction) error {
	*reply = args
	for _, cb := range h.hooks.afterAll {
		cb(args)
	}
	return nil
}
