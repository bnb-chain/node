package ttransport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"testing"
	"time"

	crand "crypto/rand"
	mrand "math/rand"

	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/transport"
	"github.com/libp2p/go-libp2p-testing/race"

	ma "github.com/multiformats/go-multiaddr"
)

// VerboseDebugging can be set to true to enable verbose debug logging in the
// stream stress tests.
var VerboseDebugging = false

var randomness []byte

var StressTestTimeout = 1 * time.Minute

func init() {
	// read 1MB of randomness
	randomness = make([]byte, 1<<20)
	if _, err := crand.Read(randomness); err != nil {
		panic(err)
	}

	if timeout := os.Getenv("TEST_STRESS_TIMEOUT_MS"); timeout != "" {
		if v, err := strconv.ParseInt(timeout, 10, 32); err == nil {
			StressTestTimeout = time.Duration(v) * time.Millisecond
		}
	}
}

type Options struct {
	connNum   int
	streamNum int
	msgNum    int
	msgMin    int
	msgMax    int
}

func fullClose(t *testing.T, s mux.MuxedStream) {
	if err := s.Close(); err != nil {
		t.Error(err)
		s.Reset()
		return
	}
	b, err := ioutil.ReadAll(s)
	if err != nil {
		t.Error(err)
	}
	if len(b) != 0 {
		t.Error("expected to be done reading")
	}
}

func randBuf(size int) []byte {
	n := len(randomness) - size
	if size < 1 {
		panic(fmt.Errorf("requested too large buffer (%d). max is %d", size, len(randomness)))
	}

	start := mrand.Intn(n)
	return randomness[start : start+size]
}

func checkErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		debug.PrintStack()
		// TODO: not safe to call in parallel
		t.Fatal(err)
	}
}

func debugLog(t *testing.T, s string, args ...interface{}) {
	if VerboseDebugging {
		t.Logf(s, args...)
	}
}

func echoStream(t *testing.T, s mux.MuxedStream) {
	defer s.Close()
	// echo everything
	var err error
	if VerboseDebugging {
		t.Logf("accepted stream")
		_, err = io.Copy(&logWriter{t, s}, s)
		t.Log("closing stream")
	} else {
		_, err = io.Copy(s, s) // echo everything
	}
	if err != nil {
		t.Error(err)
	}
}

type logWriter struct {
	t *testing.T
	W io.Writer
}

func (lw *logWriter) Write(buf []byte) (int, error) {
	lw.t.Logf("logwriter: writing %d bytes", len(buf))
	return lw.W.Write(buf)
}

func goServe(t *testing.T, l transport.Listener) (done func()) {
	closed := make(chan struct{}, 1)

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				select {
				case <-closed:
					return // closed naturally.
				default:
					checkErr(t, err)
				}
			}

			debugLog(t, "accepted connection")
			go func() {
				for {
					str, err := c.AcceptStream()
					if err != nil {
						break
					}
					go echoStream(t, str)
				}
			}()
		}
	}()

	return func() {
		closed <- struct{}{}
	}
}

func SubtestStress(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID, opt Options) {
	msgsize := 1 << 11
	errs := make(chan error, 0) // dont block anything.

	rateLimitN := 5000 // max of 5k funcs, because -race has 8k max.
	rateLimitChan := make(chan struct{}, rateLimitN)
	for i := 0; i < rateLimitN; i++ {
		rateLimitChan <- struct{}{}
	}

	rateLimit := func(f func()) {
		<-rateLimitChan
		f()
		rateLimitChan <- struct{}{}
	}

	writeStream := func(s mux.MuxedStream, bufs chan<- []byte) {
		debugLog(t, "writeStream %p, %d msgNum", s, opt.msgNum)

		for i := 0; i < opt.msgNum; i++ {
			buf := randBuf(msgsize)
			bufs <- buf
			debugLog(t, "%p writing %d bytes (message %d/%d #%x)", s, len(buf), i, opt.msgNum, buf[:3])
			if _, err := s.Write(buf); err != nil {
				errs <- fmt.Errorf("s.Write(buf): %s", err)
				continue
			}
		}
	}

	readStream := func(s mux.MuxedStream, bufs <-chan []byte) {
		debugLog(t, "readStream %p, %d msgNum", s, opt.msgNum)

		buf2 := make([]byte, msgsize)
		i := 0
		for buf1 := range bufs {
			i++
			debugLog(t, "%p reading %d bytes (message %d/%d #%x)", s, len(buf1), i-1, opt.msgNum, buf1[:3])

			if _, err := io.ReadFull(s, buf2); err != nil {
				errs <- fmt.Errorf("io.ReadFull(s, buf2): %s", err)
				debugLog(t, "%p failed to read %d bytes (message %d/%d #%x)", s, len(buf1), i-1, opt.msgNum, buf1[:3])
				continue
			}
			if !bytes.Equal(buf1, buf2) {
				errs <- fmt.Errorf("buffers not equal (%x != %x)", buf1[:3], buf2[:3])
			}
		}
	}

	openStreamAndRW := func(c mux.MuxedConn) {
		debugLog(t, "openStreamAndRW %p, %d opt.msgNum", c, opt.msgNum)

		s, err := c.OpenStream()
		if err != nil {
			errs <- fmt.Errorf("Failed to create NewStream: %s", err)
			return
		}

		bufs := make(chan []byte, opt.msgNum)
		go func() {
			writeStream(s, bufs)
			close(bufs)
		}()

		readStream(s, bufs)
		fullClose(t, s)
	}

	openConnAndRW := func() {
		debugLog(t, "openConnAndRW")

		l, err := ta.Listen(maddr)
		checkErr(t, err)

		done := goServe(t, l)
		defer done()

		c, err := tb.Dial(context.Background(), l.Multiaddr(), peerA)
		checkErr(t, err)

		// serve the outgoing conn, because some muxers assume
		// that we _always_ call serve. (this is an error?)
		go func() {
			debugLog(t, "serving connection")
			for {
				str, err := c.AcceptStream()
				if err != nil {
					break
				}
				go echoStream(t, str)
			}
		}()

		var wg sync.WaitGroup
		for i := 0; i < opt.streamNum; i++ {
			wg.Add(1)
			go rateLimit(func() {
				defer wg.Done()
				openStreamAndRW(c)
			})
		}
		wg.Wait()
		c.Close()
	}

	openConnsAndRW := func() {
		debugLog(t, "openConnsAndRW, %d conns", opt.connNum)

		var wg sync.WaitGroup
		for i := 0; i < opt.connNum; i++ {
			wg.Add(1)
			go rateLimit(func() {
				defer wg.Done()
				openConnAndRW()
			})
		}
		wg.Wait()
	}

	go func() {
		openConnsAndRW()
		close(errs) // done
	}()

	for err := range errs {
		t.Error(err)
	}

}

func SubtestStreamOpenStress(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	l, err := ta.Listen(maddr)
	checkErr(t, err)
	defer l.Close()

	count := 10000
	workers := 5

	if race.WithRace() {
		// the race detector can only deal with 8128 simultaneous goroutines, so let's make sure we don't go overboard.
		count = 1000
	}

	var (
		connA, connB transport.CapableConn
	)

	accepted := make(chan error, 1)
	go func() {
		var err error
		connA, err = l.Accept()
		accepted <- err
	}()
	connB, err = tb.Dial(context.Background(), l.Multiaddr(), peerA)
	checkErr(t, err)
	checkErr(t, <-accepted)

	defer func() {
		if connA != nil {
			connA.Close()
		}
		if connB != nil {
			connB.Close()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < workers; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < count; i++ {
					s, err := connA.OpenStream()
					if err != nil {
						t.Error(err)
						return
					}
					wg.Add(1)
					go func() {
						defer wg.Done()
						fullClose(t, s)
					}()
				}
			}()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < count*workers; i++ {
			str, err := connB.AcceptStream()
			if err != nil {
				break
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				fullClose(t, str)
			}()
		}
	}()

	timeout := time.After(StressTestTimeout)
	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-timeout:
		t.Fatal("timed out receiving streams")
	case <-done:
	}
}

func SubtestStreamReset(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	l, err := ta.Listen(maddr)
	checkErr(t, err)

	done := make(chan struct{}, 2)
	go func() {
		muxa, err := l.Accept()
		checkErr(t, err)

		s, err := muxa.OpenStream()
		if err != nil {
			panic(err)
		}

		// Some transports won't open the stream until we write. That's
		// fine.
		s.Write([]byte("foo"))

		time.Sleep(time.Millisecond * 50)

		_, err = s.Write([]byte("bar"))
		if err == nil {
			t.Error("should have failed to write")
		}

		s.Close()
		done <- struct{}{}
	}()

	muxb, err := tb.Dial(context.Background(), l.Multiaddr(), peerA)
	checkErr(t, err)

	go func() {
		str, err := muxb.AcceptStream()
		checkErr(t, err)
		str.Reset()
		done <- struct{}{}
	}()

	<-done
	<-done
}

func SubtestStress1Conn1Stream1Msg(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	SubtestStress(t, ta, tb, maddr, peerA, Options{
		connNum:   1,
		streamNum: 1,
		msgNum:    1,
		msgMax:    100,
		msgMin:    100,
	})
}

func SubtestStress1Conn1Stream100Msg(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	SubtestStress(t, ta, tb, maddr, peerA, Options{
		connNum:   1,
		streamNum: 1,
		msgNum:    100,
		msgMax:    100,
		msgMin:    100,
	})
}

func SubtestStress1Conn100Stream100Msg(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	SubtestStress(t, ta, tb, maddr, peerA, Options{
		connNum:   1,
		streamNum: 100,
		msgNum:    100,
		msgMax:    100,
		msgMin:    100,
	})
}

func SubtestStress50Conn10Stream50Msg(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	SubtestStress(t, ta, tb, maddr, peerA, Options{
		connNum:   50,
		streamNum: 10,
		msgNum:    50,
		msgMax:    100,
		msgMin:    100,
	})
}

func SubtestStress1Conn1000Stream10Msg(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	SubtestStress(t, ta, tb, maddr, peerA, Options{
		connNum:   1,
		streamNum: 1000,
		msgNum:    10,
		msgMax:    100,
		msgMin:    100,
	})
}

func SubtestStress1Conn100Stream100Msg10MB(t *testing.T, ta, tb transport.Transport, maddr ma.Multiaddr, peerA peer.ID) {
	SubtestStress(t, ta, tb, maddr, peerA, Options{
		connNum:   1,
		streamNum: 100,
		msgNum:    100,
		msgMax:    10000,
		msgMin:    1000,
	})
}
