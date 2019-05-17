package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"github.com/etcd-io/bbolt"
	log "github.com/sirupsen/logrus"
	"time"
)

type Peer struct {
	Address string
	Port uint32
	LastSeen time.Time
}

type peerdb struct {
	Database *bbolt.DB
	mutex sync.Mutex
}

var instance *peerdb
var once sync.Once

func GetInstance() *peerdb {
	once.Do(func() {
		instance = &peerdb{}
		instance.SetupDB()

		go func() {
			for {
				time.Sleep(time.Minute * 15)
				instance.RemoveStalePeers()
			}
		}()
	})
	return instance
}

func (p *peerdb) SetupDB() {
	db, err := bbolt.Open("peer.db", 0600, nil)
	if err != nil {
		log.Error("could not open db, %v", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists([]byte("PHANTOM"))
		if err != nil {
			log.Error("could not create phantom bucket: %v\n", err)
		}
		_, err = root.CreateBucketIfNotExists([]byte("PEERS"))
		if err != nil {
			log.Errorf("could not create peer bucket: %v\n", err)
		}
		return nil
	})
	if err != nil {
		log.Errorf("could not set up buckets: %v\n", err)

	}
	log.Info("Peer database loaded")

	p.Database = db
}

func (p *peerdb) GetRandomPeer() Peer {
	return p.GetRandomPeers(1)[0]
}

func (p *peerdb) GetRandomPeers(numberOfPeers uint) []Peer {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var results []Peer

	_ = p.Database.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("PHANTOM")).Bucket([]byte("PEERS"))
		stats := b.Stats()

		num := stats.KeyN

		if int(numberOfPeers) > num {
			numberOfPeers = uint(num)
		}

		var picked []int
		for i := 0; len(picked) < int(numberOfPeers); i++ {
			pick := rand.Intn(num)
			if contains(picked, pick) {
				i--
				continue
			}
			picked = append(picked, pick)
		}

		i := 0
		b.ForEach(func(k, v []byte) error {
			if contains(picked, i) {
				result := Peer{}
				err := json.Unmarshal(v, &result)
				if err != nil {
					fmt.Println(err)
				}
				results = append(results, result)
			}
			i++

			//terminate early if possible
			if len(results) == int(numberOfPeers) {
				return errors.New("Finished early - no error here.")
			}

			return nil
		})
		return nil
	})

	return results
}

func (p *peerdb) StorePeers(peers []Peer) error {
	for _, peer := range peers {
		err := p.StorePeer(peer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *peerdb) StorePeer(peer Peer) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	entryBytes, err := json.Marshal(peer)
	if err != nil {
		return fmt.Errorf("could not marshal entry json: %v", err)
	}
	err = p.Database.Update(func(tx *bbolt.Tx) error {
		err := tx.Bucket([]byte("PHANTOM")).Bucket([]byte("PEERS")).Put([]byte(GetPeerString(peer)), entryBytes)
		if err != nil {
			return fmt.Errorf("could not insert entry: %v", err)
		}
		return nil
	})
	//fmt.Println("Added Entry")
	return err
}

func GetPeerString(peer Peer) string {
	return net.JoinHostPort(peer.Address,strconv.Itoa(int(peer.Port)))
}

func (p *peerdb) RemoveStalePeers() {
	log.Info("Removing stale peers.")

	p.mutex.Lock()
	defer p.mutex.Unlock()

	_ = p.Database.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("PHANTOM")).Bucket([]byte("PEERS"))

		b.ForEach(func(k, v []byte) error {
			result := Peer{}
			err := json.Unmarshal(v, &result)
			if err != nil {
				fmt.Println(err)
			}

			if time.Now().Sub(result.LastSeen) > (time.Hour * 12) {
				//delete the peer
				b.Delete(k)
			}
			return nil
		})
		return nil
	})
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}


