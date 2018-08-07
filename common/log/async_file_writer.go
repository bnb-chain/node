package log

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type HourTicker struct {
	stop chan struct{}
	C    <-chan time.Time
}

func NewHourTicker() *HourTicker {
	ht := &HourTicker{
		stop: make(chan struct{}),
	}
	ht.C = ht.Ticker()
	return ht
}

func (ht *HourTicker) Stop() {
	ht.stop <- struct{}{}
}

func (ht *HourTicker) Ticker() <-chan time.Time {
	ch := make(chan time.Time)
	go func() {
		hour := time.Now().Hour()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case t := <-ticker.C:
				if t.Hour() != hour {
					ch <- t
					hour = t.Hour()
				}
			case <-ht.stop:
				return
			}
		}
	}()
	return ch
}

type AsyncFileWriter struct {
	filename string
	fd       *os.File

	wg         sync.WaitGroup
	started    int32
	rotate     bool
	buf        chan []byte
	stop       chan struct{}
	hourTicker *HourTicker
}

func NewAsyncFileWriter(filename string, bufSize int64) *AsyncFileWriter {
	return &AsyncFileWriter{
		filename:   filename,
		buf:        make(chan []byte, bufSize),
		stop:       make(chan struct{}),
		hourTicker: NewHourTicker(),
	}
}

func (w *AsyncFileWriter) initLogFile() error {
	var (
		fd  *os.File
		err error
	)
	realFile, err := w.timeFilename()
	if err != nil {
		return err
	}

	fd, err = os.OpenFile(realFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	w.fd = fd
	_, err = os.Lstat(w.filename)
	if err == nil || os.IsExist(err) {
		err = os.Remove(w.filename)
		if err != nil {
			return err
		}
	}

	err = os.Symlink("./"+filepath.Base(realFile), w.filename)
	if err != nil {
		return err
	}

	return nil
}

func (w *AsyncFileWriter) Start() error {
	if !atomic.CompareAndSwapInt32(&w.started, 0, 1) {
		return errors.New("logger has already been started")
	}

	err := w.initLogFile()
	if err != nil {
		return err
	}

	w.wg.Add(1)
	go func() {
		defer func() {
			atomic.StoreInt32(&w.started, 0)

			w.flushBuffer()
			w.flushAndClose()

			w.wg.Done()
		}()

		for {
			select {
			case msg, ok := <-w.buf:
				if !ok {
					fmt.Fprintln(os.Stderr, "buf channel has been closed.")
					return
				}
				w.SyncWrite(msg)
			case <-w.stop:
				return
			}
		}
	}()
	return nil
}

func (w *AsyncFileWriter) flushBuffer() {
	for msg := range w.buf {
		w.SyncWrite(msg)
	}
}

func (w *AsyncFileWriter) SyncWrite(msg []byte) {
	w.rotateFile()
	if w.fd != nil {
		w.fd.Write(msg)
	}
}

func (w *AsyncFileWriter) rotateFile() {
	select {
	case <-w.hourTicker.C:
		w.flushAndClose()
		w.initLogFile()
	default:
	}
}

func (w *AsyncFileWriter) Stop() {
	w.stop <- struct{}{}

	w.hourTicker.Stop()
}

func (w *AsyncFileWriter) Write(msg []byte) (n int, err error) {
	// TODO(wuzhenxing): for the underlying array may change, is there a better way to avoid copying slice?
	buf := make([]byte, len(msg))
	copy(buf, msg)

	select {
	case w.buf <- buf:
	default:
	}
	return 0, nil
}

func (w *AsyncFileWriter) Flush() error {
	if w.fd == nil {
		return nil
	}
	return w.fd.Sync()
}

func (w *AsyncFileWriter) flushAndClose() error {
	if w.fd == nil {
		return nil
	}

	err := w.fd.Sync()
	if err != nil {
		return err
	}

	return w.fd.Close()
}

func (w *AsyncFileWriter) timeFilename() (string, error) {
	absPath, err := filepath.Abs(w.filename)
	if err != nil {
		return "", err
	}
	return absPath + "." + time.Now().Format("2006-01-02_15"), nil
}
