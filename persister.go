package pbft

//
// support for PBFT to save persistent
//

import "sync"

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
	ps.pbftstate = data
}

func (ps *Persister) ReadPBFTState() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.pbftstate
}

func (ps *Persister) PBFTStateSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.pbftstate)
}

func (ps *Persister) SaveSnapshot(snapshot []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.snapshot = snapshot
}

func (ps *Persister) ReadSnapshot() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.snapshot
}
