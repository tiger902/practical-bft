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
	peers     []*labrpc.ClientEnd 	//!< Array of all the other server sockets for RPC
	numberOfServer int 	
	persister *Persister          	//!< Persister to be used to store data for this server in permanent storage
	serverID        int           	//!< Index into peers[] for this server
	sequenceNumber  int				//!< last sequence number 
	commandsChannel chan int		//!< channel for commands

	// add log of operations that have not been received responses


	view			int 				//!< the current view of the system
	serverLog 		map[int]interface{}	//!< commands
	prepreparedCounter 	map[int]map[int]interface{}	//!< commands
	serverLock      sync.Mutex //!< Lock to be when changing the sequenceNumber,
}

//!struct used as argument to multicast command
type PrePrepareCommandArg struct {
	View         	int        	//!< leader’s term
	SequenceNumber 	int			//!< Sequence number of the messsage
	Digest 			int 		//!< Digest of the message, which can is an int TODO: check the type of this
	Message 		interface{} //!< Message for the command TODO: decouple this
}

//!struct used as argument to multicast command
type MulticastCommandArg struct {
	View         int        //!< leader’s term
}

//!struct used as reply to multicast command
type MulticastCommandReply struct {
}


//!struct used to store the log entry
type logEntry struct {
	newCommand 	interface{}		//!< data for the operation that the client needs to the operation to be done
	prepareCount 	int
	commit 			int 
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
	leader := pbft.view % pbft.numberOfServer
	if (pbft.serverID != leader) {
		pbft.SendCommandToPrimary(command)
	}

	// if primary, multicast to all the other servers
	pbft.MulticastCommand(command)
	
	// add to the log
	
	// send to everyone else 

	// if a request has been processed already, just reply to the client

}

// Use RPC to send command to the primary when the server receives a command but it is not a primary
func (pbft *PBFT) SendCommandToPrimary(command interface{}) {
	leader := pbft.view % pbft.numberOfServer
	ok := pbft.peers[leader].Call("PBFT.ReceiveSentCommandFromServer", command)
	return ok
}

// Handles the RPC from a other servers that send start request to the leader
func (pbft *PBFT) ReceiveSentCommandFromServer(command interface{}) {
	pbft.Start(command)
}

// Processes commands in a loop as they come from commandsChannel
func (pbft *PBFT) processsCommands() {

}


// MulticastCommand sends a new command to all the followers
func (pbft *PBFT) MulticastCommand(command interface{}) {
	
	// add sequence number 
	newLogEntry := logEntry{newCommand: command, prepareCount: 1, commit: 1}
	
	pbft.serverLock.Lock() // grab lock
	sequenceNumber = pbft.sequenceNumber
	view := pbft.view
	serverCount := len(pbft.peers)
	pbft.sequenceNumber += 1
	pbft.serverLog[sequenceNumber] = newLogEntry	// TODO: do we need to lock
	pbft.serverLock.Unlock() // unlock
	
	prePrepareCommandArg := PrePrepareCommandArg{View: view, 
												SequenceNumber: sequenceNumber,
												Digest: 0 /* dummy */,
												Message: interface{} /* dummy */}
	
	for server := 0; server < serverCount; server++ {
		if server != pbft.serverID {
			ok := pbft.peers[server].Call("PBFT.MulticastCommandRPCHandler", prePrepareCommandArg)

			if !ok {
				pbft.peers[server].Call("PBFT.MulticastCommandRPCHandler", prePrepareCommandArg)
			}
		}
	}	
}

// MulticastCommand RPC handler.
//
// \param args Arguments for command multicast
// \param reply Reply for command multicast
func (pbft *PBFT) MulticastCommandRPCHandler(args MulticastCommandArg, reply *MulticastCommandReply) {

	newLogEntry := logEntry{newCommand: command, prepareCount: 1, commit: 1}

	pbft.serverLock.Lock()
	view := pbft.view
	serverCount := len(pbft.peers)
	//pbft.serverLog[sequenceNumber] = newLogEntry	// TODO: do we need to lock
	pbft.serverLock.Unlock()

	// check the view
	if (view != pbft.view) {
		return
	}

	// check if view and sequence number have not been seen
	if sequenceMap, ok1 := pbft.prepreparedCounter[view]; ok1 {
		
		if digest, ok2 := sequenceMap[args.SequenceNumber]; ok2 {
			
			match := false; // TODO: change this to the right check
 
			// check that the digests match
			if (!match) {
				return
			}
		}

	}


	// TODO: using crypto do nothing if not coming from leader 

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
	pbft.sequenceNumber = 0
	pbft.numberOfServer = length(peers)

	// TODO: make sure to update the variable from persist

	// start a go routine to make sure that we receive heartbeats from the primary
	// this will essentially be a time out that then intiates a view change

	// add more code and return the new server

	// start a go routine for handling command
	// TODO: change the size dynamically
	pbft.commandsChannel = make(chan interface{}, 2*len(peers))
	go processsCommands()

	return PBFT
}

