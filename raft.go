import (
	"encoding/gob"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

//! A Go object implementing a single Raft peer.
type Raft struct {
}

func (rf *Raft) GetState(args interface{}, reply *GetStateReply) error {

	// code removed since it should not be made public
	return nil
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(clientCommand Command, reply *int) error {

	// code removed since it should not be made public
	return nil
}

//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
/*func (rf *Raft) Kill() {
	// Your code here, if desired.
}*/

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func (rf *Raft) Make(args *MakeArgs, reply *int) error {
	// code removed since it should not be made public
	return nil
}

func startRaft() {
	log.Print("Entering main function\n")

	gob.Register(ApplyMsg{})
	gob.Register(replyAppendEntries{})
	gob.Register(RequestVoteReply{})
	gob.Register(RequestVoteArgs{})
	gob.Register(AppendEntriesReply{})
	gob.Register(AppendEntriesArg{})
	gob.Register(LogEntry{})

	log.Print("Entering server\n")
	rf := &Raft{}
	rpc.Register(rf)
	log.Print("Registering server\n")

	rpc.HandleHTTP()
	log.Print("Handle HTTP\n")

	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}

	log.Print("About to serve\n")
	http.Serve(l, nil)
	log.Print("Served them!\n")
}