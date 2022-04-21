package goodman

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"

	t "github.com/snikch/goodman/transaction"
)

const (
	defaultPort             = "61321"
	defaultMessageDelimiter = "\n"
)

// Server is responsible for starting a server and running lifecycle callbacks.
type Server struct {
	Runner           []Runner
	Port             string
	MessageDelimeter []byte
	conn             net.Conn
}

// NewServer returns a new server instance with the supplied runner. If no
// runner is supplied, a new one will be created.
func NewServer(runners []Runner) *Server {
	return &Server{
		Runner:           runners,
		Port:             defaultPort,
		MessageDelimeter: []byte(defaultMessageDelimiter),
	}
}

// Run starts the server listening for events from dredd.
func (server *Server) Run() error {
	fmt.Println("Starting")
	ln, err := net.Listen("tcp", ":"+server.Port)
	if err != nil {
		return err
	}
	fmt.Println("Accepting connection")
	conn, err := ln.Accept()
	if err != nil {
		return err
	}

	defer ln.Close()
	defer conn.Close()
	server.conn = conn

	for {
		body, err := bufio.
			NewReader(conn).
			ReadString('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		body = body[:len(body)-1]
		m := &message{}
		err = json.Unmarshal([]byte(body), m)
		if err != nil {
			return err
		}
		err = server.ProcessMessage(m)
		if err != nil {
			return err
		}
	}
}

// ProcessMessage handles a single event message.
func (server *Server) ProcessMessage(m *message) error {
	switch m.Event {
	case "beforeAll":
		fallthrough
	case "afterAll":
		m.transactions = []*t.Transaction{}
		err := json.Unmarshal(m.Data, &m.transactions)
		if err != nil {
			return err
		}
	default:
		m.transaction = &t.Transaction{}
		err := json.Unmarshal(m.Data, m.transaction)
		if err != nil {
			return err
		}
	}

	switch m.Event {
	case "beforeAll":
		server.RunBeforeAll(&m.transactions)
		break
	case "beforeEach":
		// before is run after beforeEach, as no separate event is fired.
		server.RunBeforeEach(m.transaction)
		server.RunBefore(m.transaction)
		break
	case "beforeEachValidation":
		// beforeValidation is run after beforeEachValidation, as no separate event
		// is fired.
		server.RunBeforeEachValidation(m.transaction)
		server.RunBeforeValidation(m.transaction)
		break
	case "afterEach":
		// after is run before afterEach as no separate event is fired.
		server.RunAfter(m.transaction)
		server.RunAfterEach(m.transaction)
		break
	case "afterAll":
		server.RunAfterAll(&m.transactions)
		break
	default:
		return fmt.Errorf("Unknown event '%s'", m.Event)
	}

	switch m.Event {
	case "beforeAll":
		fallthrough
	case "afterAll":
		return server.sendResponse(m, m.transactions)
	default:
		return server.sendResponse(m, m.transaction)
	}
}

func (server *Server) RunBeforeAll(trans *[]*t.Transaction) {
	for _, runner := range server.Runner {
		runner.RunBeforeAll(trans)
	}
}

func (server *Server) RunBeforeEach(trans *t.Transaction) {
	for _, runner := range server.Runner {
		runner.RunBeforeEach(trans)
	}
}

func (server *Server) RunBefore(trans *t.Transaction) {
	for _, runner := range server.Runner {
		runner.RunBefore(trans)
	}
}

func (server *Server) RunBeforeEachValidation(trans *t.Transaction) {
	for _, runner := range server.Runner {
		runner.RunBeforeEachValidation(trans)
	}
}

func (server *Server) RunBeforeValidation(trans *t.Transaction) {
	for _, runner := range server.Runner {
		runner.RunBeforeValidation(trans)
	}
}

func (server *Server) RunAfterEach(trans *t.Transaction) {
	for _, runner := range server.Runner {
		runner.RunAfterEach(trans)
	}
}

func (server *Server) RunAfter(trans *t.Transaction) {
	for _, runner := range server.Runner {
		runner.RunAfter(trans)
	}
}

func (server *Server) RunAfterAll(trans *[]*t.Transaction) {
	for _, runner := range server.Runner {
		runner.RunAfterAll(trans)
	}
}

// sendResponse submits the transaction(s) back to dredd.
func (server *Server) sendResponse(m *message, dataObj interface{}) error {
	data, err := json.Marshal(dataObj)
	if err != nil {
		return err
	}

	m.Data = json.RawMessage(data)
	response, err := json.Marshal(m)
	if err != nil {
		return err
	}
	server.conn.Write(response)
	server.conn.Write(server.MessageDelimeter)
	return nil
}

// message represents a single event received over the connection.
type message struct {
	UUID  string          `json:"uuid"`
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`

	transaction  *t.Transaction
	transactions []*t.Transaction
}
