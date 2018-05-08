package pbft

//
// support for PBFT to save persistent
//

import ("sync"
		"github.com/boltdb/bolt"
		"log"
)


type Persister struct {
	mu        sync.Mutex
	pbftstate []byte
	snapshot  []byte
}

func MakePersister() *Persister {
	return &Persister{}
}

func (ps *Persister) Copy() *Persister {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	np := MakePersister()
	np.pbftstate = ps.pbftstate
	np.snapshot = ps.snapshot
	return np
}

func (ps *Persister) SavePBFTState(data []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("storage"))
		err := b.Put([]byte("state"), data)
		return err
	})

	ps.pbftstate = data
}

func (ps *Persister) ReadPBFTState() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	state := []byte{}

	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("storage"))
		state = b.Get([]byte("state"))
		return nil
	})

	return state
}

func (ps *Persister) PBFTStateSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.pbftstate)
}

func (ps *Persister) SaveSnapshot(snapshot []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("storage"))
		err := b.Put([]byte("snapshot"), snapshot)
		return err
	})

	ps.snapshot = snapshot
}

func (ps *Persister) ReadSnapshot() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	snapshot := []byte{}

	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("storage"))
		snapshot = b.Get([]byte("snapshot"))
		return nil
	})

	return snapshot
}
