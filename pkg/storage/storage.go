package storage

import(
	"fmt"
	"log"
	"github.com/boltdb/bolt"
	"github.com/satori/go.uuid"
	//"github.com/breakcrypto/phantom/pkg/socket/wire"
)

// set up the database file
func SetupDB() (*bolt.DB, error) {
	db, err := bolt.Open("nodes.db", 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("could not open db, %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists([]byte("peers"))
		if err != nil {
			return fmt.Errorf("could not create root bucket: %v", err)
		}

		_, err = root.CreateBucketIfNotExists([]byte("nodes"))
		if err != nil {
			return fmt.Errorf("could not create peers bucket: %v", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not set up buckets, %v", err)
	}

	log.Println("Database setup complete...")
	return db, nil
}

// add a peer (in string format) to the database
func AddPeer(db *bolt.DB, peer string) error {

	// generate a uuid for the key
	id := uuid.Must(uuid.NewV4()).String()

	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("peers")).Bucket([]byte("nodes")).Put([]byte(id), []byte(peer))
		if err != nil {
			return fmt.Errorf("could not insert peer: %v", err)
		}

		return nil
	})

	fmt.Println("Added peer to database:", peer)
	return err
}

// retrieve all stored peers
func FetchPeers(db *bolt.DB) ([]string, error) {

	var list []string

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("peers")).Bucket([]byte("nodes"))
		b.ForEach(func(k, v []byte) error {
			fmt.Println("Loaded from database:", string(k), string(v))

			// load into array
			list = append(list, string(v))

			return nil
		})
		fmt.Println(list)

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	return list, nil
}