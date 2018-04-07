package pbft

//
// Filename pbft.go
//
// pbft = Make(...)
//   create a new PBFT server.
// rf.Start(command interface{}) ()
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a PBFT for its current term, and whether it thinks it is leader
//

//! Define constants to be used in this class
//! const name = something


//! A Go object implementing a single PBFT peer.
type PBFT struct {
	peers     []*labrpc.ClientEnd //!< Array of all the other server sockets for RPC
	persister *Persister          //!< Persister to be used to store data for this server in permanent storage
	me        int                 //!< Index into peers[] for this server
}

//! returns whether this server believes it is the leader.
func (rf *PBFT) GetState() (int, bool) {

	return false;
}

// Write the relavant state of PBFT to persistent storage
func (rf *PBFT) persist() {
	
}

//! Restore previously persisted state.
func (rf *PBFT) readPersist(data []byte) {
	
}

// starts a new command
// return the 
func (rf *PBFT) Start(command interface{}) {
}

// stops the server
func (rf *PBFT) Kill() {
	// Your code here, if desired.
}

//
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int, persister *Persister, applyCh chan ApplyMsg) *PBFT {

	rf := &PBFT{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// add more code and return the new server
	return PBFT
}


