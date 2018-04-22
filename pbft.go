package pbft

import (
	"bytes"
	"encoding/gob"
	"labrpc"
	"log"
	"math/rand"
	"sync"
	"time"
)

//
// Filename pbft.go
//
// pbft = Make(...)
//   create a new PBFT server.
// pbft.Start(command interface{}) ()
//   start agreement on a new log entry
// pbft.GetState() (term, isLeader)
//   ask a PBFT for its current term, and whether it thinks it is leader
//

//! Define constants to be used in this class
//! const name = something


//! A Go object implementing a single PBFT peer.
type PBFT struct {
	peers     []*labrpc.ClientEnd //!< Array of all the other server sockets for RPC
	persister *Persister          //!< Persister to be used to store data for this server in permanent storage
	serverID        int                 //!< Index into peers[] for this server

	view				int 				//!< the current view of the system
	lastClientCommand 	map[int]interface{}	//!< last command that was sent to the client
}

//!struct used as argument to multicast command
type MulticastCommandArg struct {
	View         int        //!< leader’s term
}

//!struct used as reply to multicast command
type MulticastCommandReply struct {
}

//! returns whether this server believes it is the leader.
func (pbft *PBFT) GetState() (int, bool) {

	return false;
}

// Write the relavant state of PBFT to persistent storage
func (pbft *PBFT) persist() {
	
}

//! Restore previously persisted state.
func (pbft *PBFT) readPersist(data []byte) {
	
}

// starts a new command
// return the 
func (pbft *PBFT) Start(command interface{}) {

	// if not primary, send the request to the primary

	// if primary, multicast to all the oither servers

	// if a request has been processed already, just reply to the client

}

// MulticastCommand sends a new command to all the followers
func (pbft *PBFT) MulticastCommand(command interface{}) {
	
}

// MulticastCommand RPC handler.
//
// \param args Arguments for command multicast
// \param reply Reply for command multicast
func (pbft *PBFT) MulticastCommandRPCHandler(args MulticastCommandArg, reply *MulticastCommandReply) {

	// if a request has been processed already, just reply to the client
	
	// process the command, add the result to lastClientCommand and reply to client
	// TODO: how do we know the channel for the client? Do we need to pass in the channel? What if
	// it is over the network?
	return
}

// stops the server
func (pbft *PBFT) Kill() {
	// Your code here, if desired.
}

//
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, serverID int, persister *Persister, applyCh chan ApplyMsg) *PBFT {

	pbft := &PBFT{}
	pbft.peers = peers
	pbft.persister = persister
	pbft.serverID = serverID

	// start a go routine to make sure that we receive heartbeats from the primary
	// this will essentially be a time out that then intiates a view change

	// add more code and return the new server
	return PBFT
}


