package storage

import(
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
)

func InitialiseDB(path string) (*bolt.DB, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		peerBucket, err := tx.CreateBucketIfNotExists([]byte("peers"))
		if err != nil {
			return fmt.Errorf("Could not create peers bucket: %v", err)
		} else {
			log.Println("Bucket created:", peerBucket)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	log.Println("Database successfully initialised...")

	return db, nil
}

func CachePeerToDB(db *bolt.DB, peer string) error {
	entry := peer
	entryBytes, err := json.Marshal(entry)

	if err != nil {
		return err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("peers")).Put([]byte("peer"), (entryBytes))
		if err != nil {
			return err
		}

		return nil
	})

	log.Println("Peer added to cache", entry)
	return err
}