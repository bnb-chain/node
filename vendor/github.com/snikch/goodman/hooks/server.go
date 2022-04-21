package hooks

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	trans "github.com/snikch/goodman/transaction"
)

type Server struct {
	Listener net.Listener
}

func NewServer(run RunnerRPC) *Server {
	serv := rpc.NewServer()
	serv.Register(run)
	serv.HandleHTTP("/", "/debug")

	if *port == 0 {
		panic("-port flag was not given to hook server")
	}
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	server := &Server{}
	server.Listener = l
	return server
}

func (s *Server) Serve() {
	http.Serve(s.Listener, nil)
}

type RunnerRPC interface {
	RunBeforeAll(args []*trans.Transaction, reply *[]*trans.Transaction) error
	RunBeforeEach(args trans.Transaction, reply *trans.Transaction) error
	RunBefore(args trans.Transaction, reply *trans.Transaction) error
	RunBeforeEachValidation(args trans.Transaction, reply *trans.Transaction) error
	RunBeforeValidation(args trans.Transaction, reply *trans.Transaction) error
	RunAfter(args trans.Transaction, reply *trans.Transaction) error
	RunAfterEach(args trans.Transaction, reply *trans.Transaction) error
	RunAfterAll(args []*trans.Transaction, reply *[]*trans.Transaction) error
}

var port *int

func init() {
	port = flag.Int("port", 0, "The port that the hooks server will run on")
	flag.Parse()
}
