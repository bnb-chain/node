package hot

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/p2p"
)

func makeTestPeerId(num int) []p2p.ID {
	if num <= 0 {
		return []p2p.ID{}
	}
	pids := make([]p2p.ID, 0, num)
	for i := 0; i < num; i++ {
		pids = append(pids, p2p.ID(fmt.Sprintf("test id %v", i)))
	}
	return pids
}
func randomEvent() eventType {
	if rand.Int()%2 == 0 {
		return Good
	} else {
		return Bad
	}
}
func TestPeerMetricsBasic(t *testing.T) {
	pm := newPeerMetrics(rand.New(rand.NewSource(time.Now().Unix())))
	seq := pm.sampleSequence
	sample := rand.Int63()
	pm.addSample(sample)

	assert.Equal(t, pm.average, sample)
	assert.Equal(t, pm.samples.Len(), 1)
	assert.Equal(t, pm.sampleSequence, seq+1)
}

func TestPeerMetricsNoOverflow(t *testing.T) {
	pm := newPeerMetrics(rand.New(rand.NewSource(time.Now().Unix())))
	for i := 0; i < 10000; i++ {
		// make sure it will not cause math overflow.
		sample := rand.Int63n(math.MaxInt64 / maxMetricsSampleSize)
		pm.addSample(sample)
	}
	assert.Equal(t, pm.samples.Len(), maxMetricsSampleSize)
}

func TestPeerMetricsParamResonableInMillisecondLevel(t *testing.T) {
	var sum int64
	pm := newPeerMetrics(rand.New(rand.NewSource(time.Now().Unix())))
	for i := 0; i < recalculateInterval-1; i++ {
		sample := rand.Int63n(recalculateInterval-1) * time.Millisecond.Nanoseconds()
		sum += sample
		pm.addSample(sample)
	}
	average := sum / (recalculateInterval - 1)
	// the diff is not too much
	assert.True(t, (average-pm.average) < time.Millisecond.Nanoseconds()/10 && (average-pm.average) < time.Millisecond.Nanoseconds()/10)
	// the average is not too small
	assert.True(t, average > 100)
}

func TestPeerMetricsNoErrorAccumulate(t *testing.T) {
	pm := newPeerMetrics(rand.New(rand.NewSource(time.Now().Unix())))
	for i := 0; i < 10000; i++ {
		sample := rand.Int63n(math.MaxInt64 / maxMetricsSampleSize)
		pm.addSample(sample)
	}
	var sum int64
	// make sure it will recalculate when finish
	pm.sampleSequence = 0
	for i := 0; i < maxMetricsSampleSize; i++ {
		sample := rand.Int63n(math.MaxInt64 / maxMetricsSampleSize)
		sum += sample
		pm.addSample(sample)
	}
	average := sum / maxMetricsSampleSize
	assert.Equal(t, pm.average, average)
}

func TestCandidatePoolBasic(t *testing.T) {
	sampleStream := make(chan metricsEvent)
	candidatePool := NewCandidatePool(sampleStream)
	candidatePool.Start()
	defer candidatePool.Stop()

	// control the pick decay logic
	candidatePool.pickSequence = 0
	totalPidNum := 85
	goodPidNum := 21

	testPids := makeTestPeerId(totalPidNum)
	for _, p := range testPids {
		candidatePool.addPeer(p)
	}
	// peers stay in fresh set until an event come in
	for i := 0; i < 2; i++ {
		candidates := candidatePool.PickCandidates()
		assert.Equal(t, len(candidates), totalPidNum)
	}

	for i := 0; i < goodPidNum; i++ {
		sampleStream <- metricsEvent{Good, testPids[i], int64(i) * time.Millisecond.Nanoseconds()}
	}
	for i := goodPidNum; i < totalPidNum; i++ {
		sampleStream <- metricsEvent{Bad, testPids[i], 0}
	}
	//wait for pool to handle
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < 2; i++ {
		candidates := candidatePool.PickCandidates()
		// only one peer is selected
		assert.Equal(t, len(candidates), 1)
	}
	assert.Equal(t, len(candidatePool.permanentSet), maxPermanentSetSize)
	for i := 0; i < maxPermanentSetSize; i++ {
		_, exist := candidatePool.permanentSet[testPids[i]]
		assert.True(t, exist)
	}
	assert.Equal(t, len(candidatePool.decayedSet), totalPidNum-maxPermanentSetSize)
	assert.Equal(t, len(candidatePool.freshSet), 0)
}

func TestCandidatePoolPickDecayPeriodically(t *testing.T) {
	sampleStream := make(chan metricsEvent)
	candidatePool := NewCandidatePool(sampleStream)
	candidatePool.Start()
	defer candidatePool.Stop()
	testPids := makeTestPeerId(2)
	candidatePool.addPeer(testPids[0])
	candidatePool.addPeer(testPids[1])
	sampleStream <- metricsEvent{Good, testPids[0], 1 * time.Millisecond.Nanoseconds()}
	sampleStream <- metricsEvent{Bad, testPids[1], 0}

	//wait for pool to handle
	time.Sleep(10 * time.Millisecond)
	// control the pick decay logic
	candidatePool.pickSequence = 0

	for i := 0; i < pickDecayPeerInterval-1; i++ {
		peers := candidatePool.PickCandidates()
		assert.Equal(t, len(peers), 1)
	}
	peers := candidatePool.PickCandidates()
	assert.Equal(t, len(peers), 2)
}

func TestCandidatePoolNoDuplicatePeer(t *testing.T) {
	sampleStream := make(chan metricsEvent)
	candidatePool := NewCandidatePool(sampleStream)
	candidatePool.Start()
	defer candidatePool.Stop()
	totalPidNum := 1000
	testPids := makeTestPeerId(totalPidNum)
	for _, p := range testPids {
		candidatePool.addPeer(p)
	}
	for i := 0; i < 100000; i++ {
		dur := rand.Int63()
		et := randomEvent()
		peer := testPids[rand.Intn(totalPidNum)]
		sampleStream <- metricsEvent{et, peer, dur}
	}
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, len(candidatePool.freshSet)+len(candidatePool.decayedSet)+len(candidatePool.permanentSet), totalPidNum)
}

func TestCandidatePoolPickInScore(t *testing.T) {
	sampleStream := make(chan metricsEvent)
	candidatePool := NewCandidatePool(sampleStream)
	candidatePool.Start()
	defer candidatePool.Stop()
	totalPidNum := maxPermanentSetSize
	testPids := makeTestPeerId(totalPidNum)
	for i, p := range testPids {
		candidatePool.addPeer(p)
		sampleStream <- metricsEvent{Good, p, int64(i+1) * time.Millisecond.Nanoseconds()}
	}
	time.Sleep(10 * time.Millisecond)
	candidatePool.PickCandidates()
	pickRate := make(map[p2p.ID]int)
	for i := 0; i < 100000; i++ {
		peers := candidatePool.PickCandidates()
		assert.Equal(t, 1, len(peers))
		peer := *peers[0]
		pickRate[peer] = pickRate[peer] + 1
	}
	for i := 0; i < maxPermanentSetSize; i++ {
		fmt.Printf("index %d rate is %v\n", i, float64(pickRate[testPids[i]])/float64(100000))
	}
	for i := 0; i < maxPermanentSetSize-1; i++ {
		assert.True(t, pickRate[testPids[i]] > pickRate[testPids[i+1]])
	}
}

func TestPickFromFreshSet(t *testing.T) {
	sampleStream := make(chan metricsEvent)
	candidatePool := NewCandidatePool(sampleStream)
	candidatePool.Start()
	defer candidatePool.Stop()
	testPids := makeTestPeerId(100)
	for _, p := range testPids {
		candidatePool.addPeer(p)
	}
	candidates := candidatePool.pickFromFreshSet()
	for _, c := range candidates {
		for idx, tpid := range testPids {
			if *c == tpid {
				if len(testPids) > 1 {
					testPids = append(testPids[:idx], testPids[idx+1:]...)
					break
				} else {
					testPids = nil
					break
				}
			}
		}
	}
	assert.Nil(t, testPids)
}

func TestPickFromDecayedSet(t *testing.T) {
	sampleStream := make(chan metricsEvent)
	candidatePool := NewCandidatePool(sampleStream)
	candidatePool.Start()
	defer candidatePool.Stop()
	testPids := makeTestPeerId(100)
	for _, p := range testPids {
		candidatePool.addPeer(p)
		sampleStream <- metricsEvent{Bad, p, 0}
	}
	candidates := candidatePool.pickFromDecayedSet(true)
	total:=make(map[p2p.ID]bool)
	for _, c := range candidates {
		for _, tpid := range testPids {
			if *c == tpid {
				total[tpid] = true
			}
		}
	}
	assert.Equal(t, len(total),100)
}
