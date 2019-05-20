package broadcaststore

import (
	"phantom/pkg/socket/wire"
	"strconv"
	"sync"
	"time"
)

type broadcastStore struct {
	broadcastSet map[string]*wire.MsgMNB
	mutex sync.Mutex
}

var instance *broadcastStore
var once sync.Once

func GetInstance() *broadcastStore {
	once.Do(func() {
		instance = &broadcastStore{}
		instance.broadcastSet = make(map[string]*wire.MsgMNB)

		go func() {
			for {
				time.Sleep(time.Minute * 15)
				instance.CleanUpBroadcasts()
			}
		}()
	})
	return instance
}

func (b *broadcastStore) StoreBroadcast(mnb *wire.MsgMNB) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.broadcastSet[mnb.Vin.PreviousOutPoint.Hash.String() +
		":" + strconv.Itoa(int(mnb.Vin.PreviousOutPoint.Index))] = mnb
}

func (b *broadcastStore) GetBroadcast(key string) *wire.MsgMNB {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	broadcast, ok := b.broadcastSet[key]
	//provide the template
	if !ok {
		return nil
	}
	return broadcast
}

func (b *broadcastStore) DeleteBroadcast(key string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	delete(b.broadcastSet, key)
}

func (b *broadcastStore) CleanUpBroadcasts() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for key, broadcast := range b.broadcastSet {
		sigTime := time.Unix(int64(broadcast.SigTime), 0)
		if sigTime.Add(time.Hour * 24).Before(time.Now().UTC()) {
			b.DeleteBroadcast(key)
		}
	}
}