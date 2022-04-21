package ssdp

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/koron/go-ssdp"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
)

const serviceType string = "binance:tss"

// SsdpService helps parties found others' moniker (encrypted within usn) and listen address(host:port)
// before start this service, process should listen on listenAddr
// Output of this service is PeerAddrs
type SsdpService struct {
	finished         chan bool
	listenAddrs      string              // comma separated listen address
	expectedPeers    int                 // how many listen addresses should be collected before exist
	existingMonikers map[string]struct{} // existing monikers used to filter out already known listen_addr, this used to exclude already known address during regroup
	usn              string
	monitor          *ssdp.Monitor

	PeerAddrs sync.Map // map[string]string (uuid -> connectable address)
}

// listenAddrs - comma separated listen address
func NewSsdpService(moniker, vault, listenAddrs string, expectedPeers int, existingMonikers map[string]struct{}) *SsdpService {
	s := &SsdpService{
		finished:         make(chan bool),
		listenAddrs:      listenAddrs,
		expectedPeers:    expectedPeers,
		existingMonikers: existingMonikers,
		usn:              fmt.Sprintf("unique:%s_%s", moniker, vault), // TODO: hash this combination to protect privacy
	}
	s.monitor = &ssdp.Monitor{
		Alive:  s.onAlive,
		Bye:    nil,
		Search: nil,
	}

	go s.startAdvertiser()
	return s
}

func (s *SsdpService) CollectPeerAddrs() {
	if err := s.monitor.Start(); err != nil {
		log.Fatal(err)
	}

	<-s.finished
	s.monitor.Close()
}

func (s *SsdpService) startAdvertiser() {

	ad, err := ssdp.Advertise(serviceType, s.usn, s.listenAddrs, "", 1800)
	if err != nil {
		log.Fatal(err)
	}
	// it might be fine we advertise fast,
	// because the tss process is not a daemon or long-running process
	aliveTick := time.Tick(500 * time.Millisecond)

	for range aliveTick {
		ad.Alive()
	}
	ad.Bye()
	ad.Close()
}

func (s *SsdpService) stop() {
	s.finished <- true
}

func (s *SsdpService) onAlive(m *ssdp.AliveMessage) {
	client.Logger.Debugf("ssdp onAlive: %v", m)
	if m.Type != "binance:tss" {
		return
	}
	if m.USN == s.usn ||
		!strings.HasPrefix(m.USN, "unique:") ||
		len(m.USN) == len("unique:") {
		return
	}
	if _, ok := s.PeerAddrs.Load(m.USN); !ok {
		multiAddrs := strings.Split(m.Location, ",")
		for _, multiAddr := range multiAddrs {
			_, err := common.ConvertMultiAddrStrToNormalAddr(multiAddr)
			if err != nil {
				continue
			}
			// only newly found moniker is considered as new peer
			// TODO: update old peer's listen addr
			if _, ok := s.existingMonikers[strings.Trim(m.USN, "unique:")]; !ok {
				s.PeerAddrs.Store(m.USN, multiAddr)
				client.Logger.Debugf("stored %s (%s)", m.USN, multiAddr)
			}
			break
		}
	}

	receivedAddrs := 0
	s.PeerAddrs.Range(func(_, _ interface{}) bool {
		receivedAddrs++
		return true
	})
	if receivedAddrs == s.expectedPeers {
		s.stop()
	}
}
