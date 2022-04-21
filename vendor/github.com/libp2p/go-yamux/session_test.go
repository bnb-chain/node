package yamux

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type logCapture struct{ bytes.Buffer }

func (l *logCapture) logs() []string {
	return strings.Split(strings.TrimSpace(l.String()), "\n")
}

func (l *logCapture) match(expect []string) bool {
	return reflect.DeepEqual(l.logs(), expect)
}

func captureLogs(s *Session) *logCapture {
	buf := new(logCapture)
	s.logger = log.New(buf, "", 0)
	return buf
}

type pipeConn struct {
	net.Conn
	writeDeadline pipeDeadline
	writeBlocker  chan struct{}
	closeCh       chan struct{}
}

func (p *pipeConn) SetDeadline(t time.Time) error {
	p.writeDeadline.set(t)
	return p.Conn.SetDeadline(t)
}

func (p *pipeConn) SetWriteDeadline(t time.Time) error {
	p.writeDeadline.set(t)
	return p.Conn.SetWriteDeadline(t)
}

func (p *pipeConn) Write(b []byte) (int, error) {
	select {
	case p.writeBlocker <- struct{}{}:
	case <-p.writeDeadline.wait():
		return 0, ErrTimeout
	case <-p.closeCh:
		return 0, io.ErrClosedPipe
	}
	n, err := p.Conn.Write(b)
	<-p.writeBlocker
	return n, err
}

func (p *pipeConn) Close() error {
	p.writeDeadline.set(time.Time{})
	err := p.Conn.Close()
	close(p.closeCh)
	return err
}

func (p *pipeConn) BlockWrites() {
	p.writeBlocker <- struct{}{}
}

func (p *pipeConn) UnblockWrites() {
	<-p.writeBlocker
}

func testConn() (conn1, conn2 net.Conn) {
	c1, c2 := net.Pipe()
	conn1 = &pipeConn{
		Conn:          c1,
		writeDeadline: makePipeDeadline(),
		writeBlocker:  make(chan struct{}, 1),
		closeCh:       make(chan struct{}, 1),
	}
	conn2 = &pipeConn{
		Conn:          c2,
		writeDeadline: makePipeDeadline(),
		writeBlocker:  make(chan struct{}, 1),
		closeCh:       make(chan struct{}, 1),
	}
	return conn1, conn2
}

func testConf() *Config {
	conf := DefaultConfig()
	conf.AcceptBacklog = 64
	conf.KeepAliveInterval = 100 * time.Millisecond
	conf.ConnectionWriteTimeout = 350 * time.Millisecond
	return conf
}

func testConfNoKeepAlive() *Config {
	conf := testConf()
	conf.EnableKeepAlive = false
	return conf
}

func testClientServer() (*Session, *Session) {
	return testClientServerConfig(testConf())
}

func testClientServerConfig(conf *Config) (*Session, *Session) {
	conn1, conn2 := testConn()
	client, _ := Client(conn1, conf)
	server, _ := Server(conn2, conf)
	return client, server
}

func TestClientClient(t *testing.T) {
	conf := testConf()
	conn1, conn2 := testConn()
	client1, _ := Client(conn1, conf)
	client2, _ := Client(conn2, conf)
	defer client1.Close()
	defer client2.Close()

	client1.OpenStream()
	_, err := client2.AcceptStream()
	if err == nil {
		t.Fatalf("should have failed to open a stream with two clients")
	}
	client2.OpenStream()
	_, err = client1.AcceptStream()
	if err == nil {
		t.Fatalf("should have failed to open a stream with two clients")
	}
	if !client1.IsClosed() || !client2.IsClosed() {
		t.Fatalf("sessions should have been closed by errors")
	}
}

func TestServerServer(t *testing.T) {
	conf := testConf()
	conn1, conn2 := testConn()
	server1, _ := Server(conn1, conf)
	server2, _ := Server(conn2, conf)
	defer server1.Close()
	defer server2.Close()

	server1.OpenStream()
	_, err := server2.AcceptStream()
	if err == nil {
		t.Fatalf("should have failed to open a stream with two servers")
	}
	server2.OpenStream()
	_, err = server1.AcceptStream()
	if err == nil {
		t.Fatalf("should have failed to open a stream with two servers")
	}
	if !server1.IsClosed() || !server2.IsClosed() {
		t.Fatalf("sessions should have been closed by errors")
	}
}

func TestStreamAfterShutdown(t *testing.T) {
	do := func(cb func(s *Stream)) {
		var wg sync.WaitGroup
		client, server := testClientServer()
		wg.Add(2)

		go func() {
			defer wg.Done()
			s, err := client.OpenStream()
			if err == nil {
				cb(s)
				s.Reset()
			}
			client.Close()
		}()
		go func() {
			defer wg.Done()
			server.Close()
		}()
		wg.Wait()
	}
	// test reset
	for i := 0; i < 100; i++ {
		do(func(s *Stream) {})
	}
	// test close
	for i := 0; i < 100; i++ {
		do(func(s *Stream) {
			s.Close()
		})
	}

	// test write
	for i := 0; i < 100; i++ {
		do(func(s *Stream) {
			s.Write([]byte{10})
		})
	}

	// test read
	for i := 0; i < 100; i++ {
		do(func(s *Stream) {
			s.Read([]byte{10})
		})
	}
}

func TestPing(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	rtt, err := client.Ping()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rtt == 0 {
		t.Fatalf("bad: %v", rtt)
	}

	rtt, err = server.Ping()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rtt == 0 {
		t.Fatalf("bad: %v", rtt)
	}
}

func TestCloseBeforeAck(t *testing.T) {
	cfg := testConf()
	cfg.AcceptBacklog = 8
	client, server := testClientServerConfig(cfg)

	defer client.Close()
	defer server.Close()

	for i := 0; i < 8; i++ {
		s, err := client.OpenStream()
		if err != nil {
			t.Fatal(err)
		}
		s.Close()
	}

	for i := 0; i < 8; i++ {
		s, err := server.AcceptStream()
		if err != nil {
			t.Fatal(err)
		}
		s.Close()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		s, err := client.OpenStream()
		if err != nil {
			t.Fatal(err)
		}
		s.Close()
	}()

	select {
	case <-done:
	case <-time.After(time.Second * 5):
		t.Fatal("timed out trying to open stream")
	}
}

func TestAccept(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	if client.NumStreams() != 0 {
		t.Fatalf("bad")
	}
	if server.NumStreams() != 0 {
		t.Fatalf("bad")
	}

	wg := &sync.WaitGroup{}
	wg.Add(4)

	go func() {
		defer wg.Done()
		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if id := stream.StreamID(); id != 1 {
			t.Fatalf("bad: %v", id)
		}
		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		stream, err := client.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if id := stream.StreamID(); id != 2 {
			t.Fatalf("bad: %v", id)
		}
		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		stream, err := server.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if id := stream.StreamID(); id != 2 {
			t.Fatalf("bad: %v", id)
		}
		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		stream, err := client.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if id := stream.StreamID(); id != 1 {
			t.Fatalf("bad: %v", id)
		}
		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
	case <-time.After(time.Second):
		panic("timeout")
	}
}

func TestNonNilInterface(t *testing.T) {
	_, server := testClientServer()
	server.Close()

	conn, err := server.Accept()
	if err != nil && conn != nil {
		t.Error("bad: accept should return a connection of nil value")
	}

	conn, err = server.Open()
	if err != nil && conn != nil {
		t.Error("bad: open should return a connection of nil value")
	}
}

func TestSendData_Small(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if server.NumStreams() != 1 {
			t.Fatalf("bad")
		}

		buf := make([]byte, 4)
		for i := 0; i < 1000; i++ {
			n, err := stream.Read(buf)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("short read: %d", n)
			}
			if string(buf) != "test" {
				t.Fatalf("bad: %s", buf)
			}
		}

		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
		n, err := stream.Read([]byte{0})
		if n != 0 || err != io.EOF {
			t.Errorf("err: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		if client.NumStreams() != 1 {
			t.Fatalf("bad")
		}

		for i := 0; i < 1000; i++ {
			n, err := stream.Write([]byte("test"))
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("short write %d", n)
			}
		}

		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
		n, err := stream.Read([]byte{0})
		if n != 0 || err != io.EOF {
			t.Errorf("err: %v", err)
		}
	}()

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-time.After(time.Second):
		panic("timeout")
	}

	if client.NumStreams() != 0 {
		t.Fatalf("bad")
	}
	if server.NumStreams() != 0 {
		t.Fatalf("bad")
	}
}

func TestSendData_Large(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	const (
		sendSize = 250 * 1024 * 1024
		recvSize = 4 * 1024
	)

	data := make([]byte, sendSize)
	for idx := range data {
		data[idx] = byte(idx % 256)
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		var sz int
		buf := make([]byte, recvSize)
		for i := 0; i < sendSize/recvSize; i++ {
			n, err := io.ReadFull(stream, buf)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != recvSize {
				t.Fatalf("short read: %d", n)
			}
			sz += n
			for idx := range buf {
				if buf[idx] != byte(idx%256) {
					t.Fatalf("bad: %v %v %v", i, idx, buf[idx])
				}
			}
		}

		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}

		t.Logf("cap=%d, n=%d\n", stream.recvBuf.Cap(), sz)
	}()

	go func() {
		defer wg.Done()
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		n, err := stream.Write(data)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if n != len(data) {
			t.Fatalf("short write %d", n)
		}

		if err := stream.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		panic("timeout")
	}
}

func TestGoAway(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	if err := server.GoAway(); err != nil {
		t.Fatalf("err: %v", err)
	}

	for i := 0; i < 100; i++ {
		s, err := client.Open()
		switch err {
		case nil:
			s.Close()
		case ErrRemoteGoAway:
			return
		default:
			t.Fatalf("err: %v", err)
		}
	}
	t.Fatalf("expected GoAway error")
}

func TestManyStreams(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	wg := &sync.WaitGroup{}

	acceptor := func(i int) {
		defer wg.Done()
		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		buf := make([]byte, 512)
		for {
			n, err := stream.Read(buf)
			if err == io.EOF {
				return
			}
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n == 0 {
				t.Fatalf("err: %v", err)
			}
		}
	}
	sender := func(i int) {
		defer wg.Done()
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		msg := fmt.Sprintf("%08d", i)
		for i := 0; i < 1000; i++ {
			n, err := stream.Write([]byte(msg))
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != len(msg) {
				t.Fatalf("short write %d", n)
			}
		}
	}

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go acceptor(i)
		go sender(i)
	}

	wg.Wait()
}

func TestManyStreams_PingPong(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	wg := &sync.WaitGroup{}

	ping := []byte("ping")
	pong := []byte("pong")

	acceptor := func(i int) {
		defer wg.Done()
		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		buf := make([]byte, 4)
		for {
			// Read the 'ping'
			n, err := io.ReadFull(stream, buf)
			if err == io.EOF {
				return
			}
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("err: %v", err)
			}
			if !bytes.Equal(buf, ping) {
				t.Fatalf("bad: %s", buf)
			}

			// Shrink the internal buffer!
			stream.Shrink()

			// Write out the 'pong'
			n, err = stream.Write(pong)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("err: %v", err)
			}
		}
	}
	sender := func(i int) {
		defer wg.Done()
		stream, err := client.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		buf := make([]byte, 4)
		for i := 0; i < 1000; i++ {
			// Send the 'ping'
			n, err := stream.Write(ping)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("short write %d", n)
			}

			// Read the 'pong'
			n, err = io.ReadFull(stream, buf)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != 4 {
				t.Fatalf("err: %v", err)
			}
			if !bytes.Equal(buf, pong) {
				t.Fatalf("bad: %s", buf)
			}

			// Shrink the buffer
			stream.Shrink()
		}
	}

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go acceptor(i)
		go sender(i)
	}

	wg.Wait()
}

func TestHalfClose(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, err = stream.Write([]byte("a")); err != nil {
		t.Fatalf("err: %v", err)
	}

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	stream2.Close() // Half close

	buf := make([]byte, 4)
	n, err := stream2.Read(buf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 1 {
		t.Fatalf("bad: %v", n)
	}

	// Send more
	if _, err = stream.Write([]byte("bcd")); err != nil {
		t.Fatalf("err: %v", err)
	}
	stream.Close()

	// Read after close
	n, err = stream2.Read(buf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 3 {
		t.Fatalf("bad: %v", n)
	}

	// EOF after close
	n, err = stream2.Read(buf)
	if err != io.EOF {
		t.Fatalf("err: %v", err)
	}
	if n != 0 {
		t.Fatalf("bad: %v", n)
	}
}

func TestReadDeadline(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	if err := stream.SetReadDeadline(time.Now().Add(5 * time.Millisecond)); err != nil {
		t.Fatalf("err: %v", err)
	}

	buf := make([]byte, 4)
	if _, err := stream.Read(buf); err != ErrTimeout {
		t.Fatalf("err: %v", err)
	}
}

func TestWriteDeadlineWindowFull(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	if err := stream.SetWriteDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatalf("err: %v", err)
	}

	buf := make([]byte, 512)
	for i := 0; i < int(initialStreamWindow); i++ {
		_, err := stream.Write(buf)
		if err != nil && err == ErrTimeout {
			return
		} else if err != nil {
			t.Fatalf("err: %v", err)
		}
	}
	t.Fatalf("Expected timeout")
}

func TestBacklogExceeded(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	// Fill the backlog
	max := client.config.AcceptBacklog
	for i := 0; i < max; i++ {
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		if _, err := stream.Write([]byte("foo")); err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	// Attempt to open a new stream
	errCh := make(chan error, 1)
	go func() {
		_, err := client.Open()
		errCh <- err
	}()

	// Shutdown the server
	go func() {
		time.Sleep(10 * time.Millisecond)
		server.Close()
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatalf("open should fail")
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}

func TestKeepAlive(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	time.Sleep(200 * time.Millisecond)

	// Ping value should increase
	client.pingLock.Lock()
	defer client.pingLock.Unlock()
	if client.pingID == 0 {
		t.Fatalf("should ping")
	}

	server.pingLock.Lock()
	defer server.pingLock.Unlock()
	if server.pingID == 0 {
		t.Fatalf("should ping")
	}
}

func TestKeepAlive_Timeout(t *testing.T) {
	conn1, conn2 := testConn()

	clientConf := testConf()
	clientConf.ConnectionWriteTimeout = time.Hour // We're testing keep alives, not connection writes
	clientConf.EnableKeepAlive = false            // Just test one direction, so it's deterministic who hangs up on whom
	client, _ := Client(conn1, clientConf)
	defer client.Close()

	server, _ := Server(conn2, testConf())
	defer server.Close()

	_ = captureLogs(client) // Client logs aren't part of the test
	serverLogs := captureLogs(server)

	errCh := make(chan error, 1)
	go func() {
		_, err := server.Accept() // Wait until server closes
		errCh <- err
	}()

	// Prevent the client from responding
	clientConn := client.conn.(*pipeConn)
	clientConn.BlockWrites()

	select {
	case err := <-errCh:
		if err != ErrKeepAliveTimeout {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for timeout")
	}

	if !server.IsClosed() {
		t.Fatalf("server should have closed")
	}

	if !serverLogs.match([]string{"[ERR] yamux: keepalive failed: i/o deadline reached"}) {
		t.Fatalf("server log incorect: %v", serverLogs.logs())
	}
}

func TestLargeWindow(t *testing.T) {
	conf := DefaultConfig()
	conf.MaxStreamWindowSize *= 2

	client, server := testClientServerConfig(conf)
	defer client.Close()
	defer server.Close()

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	stream.SetWriteDeadline(time.Now().Add(10 * time.Millisecond))
	buf := make([]byte, conf.MaxStreamWindowSize)
	n, err := stream.Write(buf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != len(buf) {
		t.Fatalf("short write: %d", n)
	}
}

type UnlimitedReader struct{}

func (u *UnlimitedReader) Read(p []byte) (int, error) {
	runtime.Gosched()
	return len(p), nil
}

func TestSendData_VeryLarge(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	var n int64 = 1 * 1024 * 1024 * 1024
	var workers int = 16

	wg := &sync.WaitGroup{}
	wg.Add(workers * 2)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			stream, err := server.AcceptStream()
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			defer stream.Close()

			buf := make([]byte, 4)
			_, err = io.ReadFull(stream, buf)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if !bytes.Equal(buf, []byte{0, 1, 2, 3}) {
				t.Fatalf("bad header")
			}

			recv, err := io.Copy(ioutil.Discard, stream)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if recv != n {
				t.Fatalf("bad: %v", recv)
			}
		}()
	}
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			stream, err := client.Open()
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			defer stream.Close()

			_, err = stream.Write([]byte{0, 1, 2, 3})
			if err != nil {
				t.Fatalf("err: %v", err)
			}

			unlimited := &UnlimitedReader{}
			sent, err := io.Copy(stream, io.LimitReader(unlimited, n))
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if sent != n {
				t.Fatalf("bad: %v", sent)
			}
		}()
	}

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-time.After(20 * time.Second):
		panic("timeout")
	}
}

func TestBacklogExceeded_Accept(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	max := 5 * client.config.AcceptBacklog
	go func() {
		for i := 0; i < max; i++ {
			stream, err := server.Accept()
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			defer stream.Close()
		}
	}()

	// Fill the backlog
	for i := 0; i < max; i++ {
		stream, err := client.Open()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		if _, err := stream.Write([]byte("foo")); err != nil {
			t.Fatalf("err: %v", err)
		}
	}
}

func TestSession_WindowUpdateWriteDuringRead(t *testing.T) {
	client, server := testClientServerConfig(testConfNoKeepAlive())
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// Choose a huge flood size that we know will result in a window update.
	flood := int64(client.config.MaxStreamWindowSize) + 1

	sync := make(chan struct{})

	// The server will accept a new stream and then flood data to it.
	go func() {
		defer wg.Done()

		stream, err := server.AcceptStream()
		if err != nil {
			t.Errorf("err: %v", err)
			server.Close()
			return
		}
		defer stream.Close()

		<-sync
		sync <- struct{}{}

		_, err = stream.Write(make([]byte, flood))
		if err == nil {
			t.Errorf("expected write to fail due to no window update")
			return
		}
	}()

	// The client will open a stream, block outbound writes, and then
	// listen to the flood from the server, which should time out since
	// it won't be able to send the window update.
	go func() {
		defer wg.Done()

		stream, err := client.OpenStream()
		if err != nil {
			t.Errorf("err: %v", err)
			server.Close()
			return
		}
		defer stream.Close()

		sync <- struct{}{}
		conn := client.conn.(*pipeConn)
		conn.BlockWrites()
		<-sync

		_, err = io.ReadFull(stream, make([]byte, flood))
		if err == nil {
			t.Errorf("expected read to fail")
		}
	}()

	wg.Wait()
}

func TestSession_PartialReadWindowUpdate(t *testing.T) {
	client, server := testClientServerConfig(testConfNoKeepAlive())
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	// Choose a huge flood size that we know will result in a window update.
	flood := int64(client.config.MaxStreamWindowSize)
	var wr *Stream

	// The server will accept a new stream and then flood data to it.
	go func() {
		defer wg.Done()

		var err error
		wr, err = server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer wr.Close()

		sendWindow := atomic.LoadUint32(&wr.sendWindow)
		if sendWindow != client.config.MaxStreamWindowSize {
			t.Fatalf("sendWindow: exp=%d, got=%d", client.config.MaxStreamWindowSize, sendWindow)
		}

		n, err := wr.Write(make([]byte, flood))
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if int64(n) != flood {
			t.Fatalf("short write: %d", n)
		}
		sendWindow = atomic.LoadUint32(&wr.sendWindow)
		if sendWindow != 0 {
			t.Fatalf("sendWindow: exp=%d, got=%d", 0, sendWindow)
		}
	}()

	stream, err := client.OpenStream()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	wg.Wait()

	_, err = io.ReadFull(stream, make([]byte, flood/2))
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Millisecond)

	sendWindow := atomic.LoadUint32(&wr.sendWindow)
	if exp := uint32(flood / 2); sendWindow != exp {
		t.Errorf("sendWindow: exp=%d, got=%d", exp, sendWindow)
	}
}

func TestSession_sendMsg_Timeout(t *testing.T) {
	client, server := testClientServerConfig(testConfNoKeepAlive())
	defer client.Close()
	defer server.Close()

	conn := client.conn.(*pipeConn)
	conn.BlockWrites()

	hdr := encode(typePing, flagACK, 0, 0)
	for {
		err := client.sendMsg(hdr, nil, nil)
		if err == nil {
			continue
		} else if err == ErrConnectionWriteTimeout {
			break
		} else {
			t.Fatalf("err: %v", err)
		}
	}
}

func TestSession_PingOfDeath(t *testing.T) {
	client, server := testClientServerConfig(testConfNoKeepAlive())
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	begin := make(chan struct{})
	for i := 0; i < 10000; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-begin
			if _, err := server.Ping(); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer wg.Done()
			<-begin
			if _, err := client.Ping(); err != nil {
				t.Error(err)
			}
		}()
	}
	close(begin)
	wg.Wait()
}

func TestSession_ConnectionWriteTimeout(t *testing.T) {
	client, server := testClientServerConfig(testConfNoKeepAlive())
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	sync := make(chan struct{})

	go func() {
		defer wg.Done()

		stream, err := server.AcceptStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		<-sync
		sync <- struct{}{}

		defer stream.Close()
	}()

	// The client will open the stream and then block outbound writes, we'll
	// tee up a write and make sure it eventually times out.
	go func() {
		defer wg.Done()

		stream, err := client.OpenStream()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		defer stream.Close()

		sync <- struct{}{}
		conn := client.conn.(*pipeConn)
		conn.BlockWrites()
		<-sync

		// Fill up the write queue and wait for the write to timeout.
		for {
			_, err := stream.Write([]byte("hello"))
			if err == nil {
				continue
			} else if err == ErrConnectionWriteTimeout {
				break
			} else {
				t.Fatalf("err: %v", err)
			}
		}
	}()

	wg.Wait()
}

func TestStreamResetWrite(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	wait := make(chan struct{})
	go func() {
		defer close(wait)
		stream, err := server.AcceptStream()
		if err != nil {
			t.Errorf("err: %v", err)
		}

		time.Sleep(time.Millisecond * 50)

		_, err = stream.Write([]byte("foo"))
		if err == nil {
			t.Errorf("should have failed to write")
		}
	}()

	stream, err := client.OpenStream()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	stream.Reset()
	<-wait
}

// Because reads should succeed after closing the stream.
func TestStreamHalfClose2(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	wait := make(chan struct{})

	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			t.Error(err)
		}
		<-wait
		_, err = stream.Write([]byte("asdf"))
		if err != nil {
			t.Error(err)
		}
		stream.Close()
		wait <- struct{}{}
	}()

	stream, err := client.OpenStream()
	if err != nil {
		t.Error(err)
	}

	stream.Close()
	wait <- struct{}{}

	buf, err := ioutil.ReadAll(stream)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(buf, []byte("asdf")) {
		t.Fatalf("didn't get expected data")
	}
	<-wait
}

func TestStreamResetRead(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	wc := new(sync.WaitGroup)
	wc.Add(2)
	go func() {
		defer wc.Done()
		stream, err := server.AcceptStream()
		if err != nil {
			t.Error(err)
		}

		_, err = ioutil.ReadAll(stream)
		if err == nil {
			t.Errorf("expected reset")
		}
	}()

	stream, err := client.OpenStream()
	if err != nil {
		t.Error(err)
	}

	go func() {
		defer wc.Done()

		_, err := ioutil.ReadAll(stream)
		if err == nil {
			t.Errorf("expected reset")
		}
	}()

	time.Sleep(1 * time.Second)
	stream.Reset()
	wc.Wait()
}

func TestLotsOfWritesWithStreamDeadline(t *testing.T) {
	config := testConf()
	config.EnableKeepAlive = false

	client, server := testClientServerConfig(config)
	defer client.Close()
	defer server.Close()

	waitCh := make(chan struct{})
	doneCh := make(chan struct{})

	// Server side accepts two streams. The first one is the clogger.
	go func() {
		defer close(doneCh)
		_, err := server.AcceptStream()
		if err != nil {
			t.Error(err)
			return
		}

		stream2, err := server.AcceptStream()
		if err != nil {
			t.Error(err)
			return
		}

		waitCh <- struct{}{}

		// Wait until all writes have timed out on the client.
		<-waitCh

		// stream2 should've received no messages, as they all expired in the buffer.
		stream2.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		if b, err := ioutil.ReadAll(stream2); len(b) != 0 || err != ErrTimeout {
			t.Errorf("writes from the client should've expired; got: %v, bytes: %v", err, b)
			return
		}
	}()

	// stream1 is the clogger.
	stream1, err := client.OpenStream()
	if err != nil {
		t.Fatal(err)
	}

	// all writes on stream2 will time out.
	stream2, err := client.OpenStream()
	if err != nil {
		t.Fatal(err)
	}
	defer stream2.Reset()

	// wait for the server to accept the streams.
	<-waitCh

	clientConn := client.conn.(*pipeConn)
	clientConn.BlockWrites()

	// Send a clogging write on stream1.
	go func() {
		stream1.SetWriteDeadline(time.Now().Add(5 * time.Second))
		stream1.Write([]byte{100})
	}()

	// Keep writing till we fill the buffer and timeout.
	var wg sync.WaitGroup
	stream2.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
	for {
		_, err := stream2.Write([]byte("foobar"))
		if err == nil {
			continue
		} else if err == ErrTimeout {
			break
		} else {
			t.Errorf("expected stream timeout error, got: %v", err)
			break
		}
	}

	// All writes completed and timed out; notify the server.
	wg.Wait()
	select {
	case waitCh <- struct{}{}:
	default:
	}
	<-doneCh
}

func TestReadDeadlineInterrupt(t *testing.T) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	stream, err := client.Open()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream.Close()

	stream2, err := server.Accept()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer stream2.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4)
		if _, err := stream.Read(buf); err != ErrTimeout {
			t.Fatalf("err: %v", err)
		}
	}()

	select {
	case <-done:
		t.Fatal("read shouldn't have finished")
	case <-time.After(5 * time.Millisecond):
	}

	if err := stream.SetReadDeadline(time.Now().Add(5 * time.Millisecond)); err != nil {
		t.Fatalf("err: %v", err)
	}

	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("read should have finished")
	}

	for i := 0; i < 5; i++ {
		buf := make([]byte, 4)
		if _, err := stream.Read(buf); err != ErrTimeout {
			t.Fatalf("err: %v", err)
		}
	}
}
