package order

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	amino "github.com/tendermint/go-amino"
	cs "github.com/tendermint/tendermint/consensus"
	cmn "github.com/tendermint/tendermint/libs/common"
)

type WALEncoder = cs.WALEncoder
type WALDecoder = cs.WALDecoder
type TimedWALMessage = cs.TimedWALMessage

var NewWALEncoder = cs.NewWALEncoder
var NewWALDeccoder = cs.NewWALDecoder

const (
	// must be greater than 4K orders
	maxMsgSizeBytes = 4 * 1024 * 1024 // 4MB
	walFileName     = "orderbook_wal"
)

type WALMessage interface{}

func RegisterWALMessages(cdc *amino.Codec) {
	cdc.RegisterInterface((*WALMessage)(nil), nil)
}

//--------------------------------------------------------
// Simple write-ahead logger

// WAL is an interface for any write-ahead logger.
type WAL interface {
	Write(WALMessage)
	WriteSync(WALMessage)
	File() *os.File
	Start() error
	Stop() error
	Wait()
}

// Write ahead logger writes order book snapshot and changes to disk after block execution.
// Can be used for crash-recovery and deterministic replay

type orderbookWAL struct {
	cmn.BaseService

	file *os.File

	enc *WALEncoder
}

func locateLatestFile(dirPath string) (string, error) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("failed to list directory[%s] for WAL", dirPath))
	}
	latestName := ""
	latestDate, _ := time.Parse("20060102", "20000101") //impossible old date
	for _, f := range files {
		name := f.Name()
		l := len(name)
		if l < 9+len(walFileName) { //$walFileName.20060102
			continue
		}
		dateStr := name[l-8:]
		d, err := time.Parse("20060102", dateStr)
		if err != nil {
			continue
		}
		if d.After(latestDate) {
			latestName = name
			latestDate = d
		}
	}
	if latestName != "" { // locate one
		return filepath.Join(dirPath, latestName), nil
	} else {
		return filepath.Join(dirPath, walFileName, time.Now().Format("20060102")), nil
	}
}

func NewWAL(walPath string) (*orderbookWAL, error) {
	dirPath := filepath.Dir(walPath)
	err := cmn.EnsureDir(dirPath, 0700)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ensure WAL directory is in place")
	}

	filePath, err := locateLatestFile(dirPath)

	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	wal := &orderbookWAL{
		file: file,
		enc:  NewWALEncoder(file),
	}
	wal.BaseService = *cmn.NewBaseService(nil, "orderbookWAL", wal)
	return wal, nil
}

func (wal *orderbookWAL) Group() *os.File {
	return wal.file
}

func (wal *orderbookWAL) OnStart() error {
	//load existing order book.
	return nil
}

func (wal *orderbookWAL) OnStop() {
	wal.file.Close()
}

// Write is called in newStep and for each receive on the
// peerMsgQueue and the timeoutTicker.
// NOTE: does not call fsync()
func (wal *orderbookWAL) Write(msg WALMessage) {
	if wal == nil {
		return
	}

	// Write the wal message
	if err := wal.enc.Encode(&TimedWALMessage{Time: time.Now(), Msg: msg}); err != nil {
		panic(cmn.Fmt("Error writing msg to consensus wal: %v \n\nMessage: %v", err, msg))
	}
}

// WriteSync is called when we receive a msg from ourselves
// so that we write to disk before sending signed messages.
// NOTE: calls fsync()
func (wal *orderbookWAL) WriteSync(msg WALMessage) {
	if wal == nil {
		return
	}

	wal.Write(msg)
	if err := wal.file.Sync(); err != nil {
		panic(cmn.Fmt("Error flushing consensus wal buf to file. Error: %v \n", err))
	}
}

// WALSearchOptions are optional arguments to SearchForEndHeight.
type WALSearchOptions struct {
	// IgnoreDataCorruptionErrors set to true will result in skipping data corruption errors.
	IgnoreDataCorruptionErrors bool
}

type nilWAL struct{}

func (nilWAL) Write(m WALMessage)     {}
func (nilWAL) WriteSync(m WALMessage) {}
func (nilWAL) File() *os.File         { return nil }

func (nilWAL) Start() error { return nil }
func (nilWAL) Stop() error  { return nil }
func (nilWAL) Wait()        {}
