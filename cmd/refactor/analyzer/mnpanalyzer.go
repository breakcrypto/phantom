package analyzer

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"phantom/pkg/socket/wire"
	"sync"
	log "github.com/sirupsen/logrus"
)

type mnpAnalyzer struct {
	UsesOutpoints    []bool
	SentinelVersions []uint32
	DaemonVersions   []uint32
	Threshold        uint
	pingsAnalyzed    uint
	analyzedHashes   []chainhash.Hash
	mutex sync.Mutex
}

var instance *mnpAnalyzer
var once sync.Once

func GetInstance() *mnpAnalyzer {
	once.Do(func() {
		instance = &mnpAnalyzer{}
	})
	return instance
}

func (m *mnpAnalyzer) AnalyzePing(ping *wire.MsgMNP) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if contains(m.analyzedHashes, ping.GetHash()) {
		log.Debug("Duplicate ping detected, ignoring analysis.")
		return false
	}

	m.analyzedHashes = append(m.analyzedHashes, ping.GetHash())

	m.UsesOutpoints = append(m.UsesOutpoints, ping.UseOutpointForm)
	m.SentinelVersions = append(m.SentinelVersions, ping.SentinelVersion)
	m.DaemonVersions = append(m.DaemonVersions, ping.DaemonVersion)

	m.pingsAnalyzed++

	if m.pingsAnalyzed > m.Threshold {
		return true
	}
	return false
}

func (m *mnpAnalyzer) GetResults() (bool, uint32, uint32) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.calculateModeForBool(m.UsesOutpoints),
		m.calculateModeForUint32(m.SentinelVersions),
		m.calculateModeForUint32(m.DaemonVersions)
}

func (m *mnpAnalyzer) calculateModeForUint32(values []uint32) (result uint32) {
	var highestScore uint = 0
	frequencies := make(map[uint32]uint)
	for _, value := range values {
		frequencies[value] = frequencies[value] + 1
		if frequencies[value] > highestScore {
			result = value
		}
	}
	return result
}

func (m *mnpAnalyzer) calculateModeForBool(values []bool) (result bool) {
	var highestScore uint = 0
	frequencies := make(map[bool]uint)
	for _, value := range values {
		frequencies[value] = frequencies[value] + 1
		if frequencies[value] > highestScore {
			result = value
		}
	}
	return result
}

func contains(hashes []chainhash.Hash, hash chainhash.Hash) bool {
	for _, a := range hashes {
		if a == hash {
			return true
		}
	}
	return false
}