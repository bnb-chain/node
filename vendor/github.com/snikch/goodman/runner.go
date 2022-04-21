package goodman

import (
	"fmt"
	"net/rpc"

	"github.com/snikch/goodman/transaction"
)

func NewRunner(rpcService string, port int) (*Run, error) {
	client, err := rpc.DialHTTPPath("tcp", fmt.Sprintf(":%d", port), "/")

	if err != nil {
		return nil, err
	}
	return &Run{
		client:     client,
		rpcService: rpcService,
	}, nil
}

type Run struct {
	client     *rpc.Client
	rpcService string
}

func (r *Run) RunBeforeAll(t *[]*transaction.Transaction) {
	var reply []*transaction.Transaction
	err := r.client.Call(r.rpcService+".RunBeforeAll", t, &reply)

	if err != nil {
		panic("RPC client threw error " + err.Error())
	}
	*t = reply
}

func (r *Run) RunBeforeEach(t *transaction.Transaction) {
	var reply transaction.Transaction
	err := r.client.Call(r.rpcService+".RunBeforeEach", *t, &reply)

	if err != nil {
		panic("RPC client threw error " + err.Error())
	}
	*t = reply
}

func (r *Run) RunBefore(t *transaction.Transaction) {
	var reply transaction.Transaction
	err := r.client.Call(r.rpcService+".RunBefore", *t, &reply)

	if err != nil {
		panic("RPC client threw error " + err.Error())
	}
	*t = reply
}

func (r *Run) RunBeforeEachValidation(t *transaction.Transaction) {
	var reply transaction.Transaction
	err := r.client.Call(r.rpcService+".RunBeforeEachValidation", *t, &reply)

	if err != nil {
		panic("RPC client threw error " + err.Error())
	}
	*t = reply
}

func (r *Run) RunBeforeValidation(t *transaction.Transaction) {
	var reply transaction.Transaction
	err := r.client.Call(r.rpcService+".RunBeforeValidation", *t, &reply)

	if err != nil {
		panic("RPC client threw error " + err.Error())
	}
	*t = reply
}

func (r *Run) RunAfterAll(t *[]*transaction.Transaction) {
	var reply []*transaction.Transaction
	err := r.client.Call(r.rpcService+".RunAfterAll", t, &reply)

	if err != nil {
		panic("RPC client threw error " + err.Error())
	}
	*t = reply
}

func (r *Run) RunAfterEach(t *transaction.Transaction) {
	var reply transaction.Transaction
	err := r.client.Call(r.rpcService+".RunAfterEach", *t, &reply)

	if err != nil {
		panic("RPC client threw error " + err.Error())
	}
	*t = reply
}

func (r *Run) RunAfter(t *transaction.Transaction) {
	var reply transaction.Transaction
	err := r.client.Call(r.rpcService+".RunAfter", *t, &reply)

	if err != nil {
		panic("RPC client threw error " + err.Error())
	}
	*t = reply
}

func (r *Run) Close() {
	if err := r.client.Close(); err != nil {
		panic("RPC client threw error on Close() " + err.Error())
	}
}

type Runner interface {
	RunBeforeAll(t *[]*transaction.Transaction)
	RunBeforeEach(t *transaction.Transaction)
	RunBefore(t *transaction.Transaction)
	RunBeforeEachValidation(t *transaction.Transaction)
	RunBeforeValidation(t *transaction.Transaction)
	RunAfterAll(t *[]*transaction.Transaction)
	RunAfterEach(t *transaction.Transaction)
	RunAfter(t *transaction.Transaction)
	Close()
}

type DummyRunner struct{}

func (r *DummyRunner) RunBeforeAll(t *[]*transaction.Transaction) {}

func (r *DummyRunner) RunBeforeEach(t *transaction.Transaction) {}

func (r *DummyRunner) RunBefore(t *transaction.Transaction) {}

func (r *DummyRunner) RunBeforeEachValidation(t *transaction.Transaction) {}

func (r *DummyRunner) RunBeforeValidation(t *transaction.Transaction) {}

func (r *DummyRunner) RunAfterAll(t *[]*transaction.Transaction) {}

func (r *DummyRunner) RunAfterEach(t *transaction.Transaction) {}

func (r *DummyRunner) RunAfter(t *transaction.Transaction) {}

func (r *DummyRunner) Close() {}
