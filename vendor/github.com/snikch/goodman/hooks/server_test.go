package hooks

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"os/exec"
	"testing"
	"time"

	r "github.com/snikch/goodman/rpc"
	trans "github.com/snikch/goodman/transaction"
)

var run = r.DummyRunner{}

func TestServerRPC(t *testing.T) {
	hooksServerPort := 61322
	var addr = fmt.Sprintf(":%d", hooksServerPort)
	if os.Getenv("RUN_HOOKS") == "1" {
		server := NewServer(&run)
		fmt.Println("Running the server")
		server.Serve()
		defer server.Listener.Close()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestServerRPC", fmt.Sprintf("-port=%d", hooksServerPort))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "RUN_HOOKS=1")

	go func() {
		err := cmd.Run()
		fmt.Println("Command exited with error " + err.Error())
	}()

	time.Sleep(2000 * time.Millisecond)
	client, err := rpc.DialHTTPPath("tcp", addr, "/")
	defer cmd.Process.Kill()
	defer client.Close()
	if err != nil {
		log.Fatal("dialing:", err)
	}

	testCases := []struct {
		Method string
		args   interface{}
		reply  interface{}
	}{
		{
			Method: "RunBeforeEach",
			args:   trans.Transaction{Name: "Test"},
			reply:  trans.Transaction{},
		},
		{
			Method: "RunBefore",
			args:   trans.Transaction{Name: "Test"},
			reply:  trans.Transaction{},
		},
		{
			Method: "RunBeforeValidation",
			args:   trans.Transaction{Name: "Test"},
			reply:  trans.Transaction{},
		},
		{
			Method: "RunBeforeEachValidation",
			args:   trans.Transaction{Name: "Test"},
			reply:  trans.Transaction{},
		},
		{
			Method: "RunAfterEach",
			args:   trans.Transaction{Name: "Test"},
			reply:  trans.Transaction{},
		},
		{
			Method: "RunAfter",
			args:   trans.Transaction{Name: "Test"},
			reply:  trans.Transaction{},
		},
	}

	for _, test := range testCases {
		args := test.args.(trans.Transaction)
		reply := test.reply.(trans.Transaction)
		method := test.Method
		err = client.Call("DummyRunner."+method, args, &reply)
		if err != nil {
			t.Errorf("rpc client failed to connect to server: %s", err.Error())
		}

		// DummyRunner will set the transaction to the value of the args variable.
		// See rpc.go for more detail.
		if reply.Name != "Test" {
			t.Errorf("RPC method %s was never invoked", method)
		}
	}

	// Testing for RunBeforeAll and RunAfter All
	allCases := []struct {
		Method string
		args   []*trans.Transaction
	}{
		{
			Method: "RunBeforeAll",
			args:   []*trans.Transaction{&trans.Transaction{Name: "Test"}},
		},
		{
			Method: "RunAfterAll",
			args:   []*trans.Transaction{&trans.Transaction{Name: "Test"}},
		},
	}

	for _, test := range allCases {
		args := test.args
		var reply []*trans.Transaction
		method := test.Method
		err = client.Call("DummyRunner."+method, args, &reply)
		if err != nil {
			t.Errorf("rpc client failed to connect to server: %s", err.Error())
		}

		// DummyRunner will set the transaction to the value of the args variable.
		// See rpc.go for more detail.
		if reply[0].Name != "Test" {
			t.Errorf("RPC method %s was never invoked", method)
		}
	}
}
