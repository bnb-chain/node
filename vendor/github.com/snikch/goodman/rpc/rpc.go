package rpc

import trans "github.com/snikch/goodman/transaction"

// DummyRunner is an implementation of the hooks.RunnerRPC interface
// it is strictly used for testing to ensure that the hooks.server
// serves its rpc correctly.
type DummyRunner struct{}

func (run *DummyRunner) RunBeforeAll(args []*trans.Transaction, reply *[]*trans.Transaction) error {
	*reply = args
	return nil
}

func (run *DummyRunner) RunBeforeEach(args trans.Transaction, reply *trans.Transaction) error {
	*reply = args
	return nil
}

func (run *DummyRunner) RunBefore(args trans.Transaction, reply *trans.Transaction) error {
	*reply = args
	return nil
}

func (run *DummyRunner) RunBeforeEachValidation(args trans.Transaction, reply *trans.Transaction) error {
	*reply = args
	return nil
}

func (run *DummyRunner) RunBeforeValidation(args trans.Transaction, reply *trans.Transaction) error {
	*reply = args
	return nil
}

func (run *DummyRunner) RunAfter(args trans.Transaction, reply *trans.Transaction) error {
	*reply = args
	return nil
}

func (run *DummyRunner) RunAfterEach(args trans.Transaction, reply *trans.Transaction) error {
	*reply = args
	return nil
}

func (run *DummyRunner) RunAfterAll(args []*trans.Transaction, reply *[]*trans.Transaction) error {
	*reply = args
	return nil
}
