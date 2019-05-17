package blockqueue

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/sirupsen/logrus"
	"phantom/pkg/phantom"
	"sync"
	"time"
)

type HashCount struct {
	Count uint
	Time time.Time
}

type BlockQueue struct {
	queue *phantom.Queue
	Threshold uint
	orphans map[string]*HashCount
	mutex sync.Mutex
}

var instance *BlockQueue
var once sync.Once

//TODO RENAME WHEN IN PROPER PACKAGES
func GetInstance() *BlockQueue {
	once.Do(func() {
		instance = &BlockQueue{}
		instance.queue = phantom.NewQueue(12)
		instance.orphans = make(map[string]*HashCount)
		instance.Threshold = 5

		//set the queue to clean itself up
		go instance.cleanMap()
	})
	return instance
}

func (b *BlockQueue) ForceHash(hash chainhash.Hash) {
	b.queue.Push(&hash)
}

func (b *BlockQueue) AddHash(hash chainhash.Hash) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if value, ok := b.orphans[hash.String()]; ok {

		//found a hash, update it.
		value.Count++

		log.WithFields(log.Fields{
			"hash":hash,
			"hash_count:":value.Count,
		}, ).Debug("Hash found in queue.")

		//if we're over the threshold push the hash to the queue
		//remove from the orphan table
		if value.Count >= b.Threshold {
			log.WithField("hash", hash).Debug("Hash over threshold, pushing.")
			b.queue.Push(&hash)
			//don't delete because other peers might still chime in, let the clean up func() handle it.
			//delete(b.orphans, hash.String())

			for b.queue.Len() > 12 { //clear the queue until we're at 12 entries
				b.queue.Pop()
				//log.Println("Removing hash from queue: ", popped.String(), "(", queue.count, ")")
			}

			return
		}

		//only update time if we're not over the threshold
		value.Time = time.Now()

	} else {
		log.Debug("New hash added: ", hash)
		//new hash, save it
		b.orphans[hash.String()] = &HashCount{Count:1, Time: time.Now()}
	}
}

func (b *BlockQueue) GetTop() *chainhash.Hash {
	return b.queue.Peek()
}

func (b *BlockQueue) cleanMap() {
	for {
		//check every 5 minutes
		time.Sleep(time.Minute * 5)
		log.WithField("time", time.Now()).Info("Cleaning orphan blocks.")

		b.mutex.Lock()
		currentTime := time.Now()
		for k, v := range b.orphans {
			if currentTime.Sub(v.Time) > time.Minute*30 {
				delete(b.orphans, k)
			}
		}
		b.mutex.Unlock()
	}
}