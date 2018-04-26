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

//!< Consts with the phases of the protocol
const (
	PRE_PREPARE		= 0
	PREPARE	     	= 1
	COMMIT 			= 2
	REPLY_TO_CLIENT	= 3
)


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
	prePreparedLog 	map[int]prePrepare	//!< commands
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
type PrepareCommandArg struct {
	View         	int        	//!< leader’s term
	SequenceNumber 	int			//!< Sequence number of the messsage
	Digest 			int 		//!< Digest of the message, which can is an int TODO: check the type of this
	SenderIndex 	interface{} //!< Id of the server that sends the prepare message
}

//!struct used as argument to multicast command
type MulticastCommandArg struct {
	View         int        //!< leader’s term
}

//!struct used as reply to multicast command
type MulticastCommandReply struct {
}


//!struct used as argument to multicast command
type PrepareCommandArg struct {
	View         	int        	//!< leader’s term
	SequenceNumber 	int			//!< Sequence number of the messsage
	Digest 			int 		//!< Digest of the message, which can is an int TODO: check the type of this
	SenderIndex 	interface{} //!< Id of the server that sends the prepare message
}

//!struct used as argument to multicast command
type CommitArg struct {
	View         	int        	//!< leader’s term
	SequenceNumber 	int			//!< Sequence number of the messsage
	Digest 			int 		//!< Digest of the message, which can is an int TODO: check the type of this
	SenderIndex 	interface{} //!< Id of the server that sends the prepare message
}

//!struct used to store the log entry
type logEntry struct {
	Message 		interface{}		//!< data for the operation that the client needs to the operation to be done
	prepareCount 	int
	commitCount 	int 
	prepared 		bool
	commandDigest 	interface{}		//!< data for the operation that the client needs to the operation to be done
	view 			int
}

//!struct used to store in log after the prepepara call
type prePrepare struct {
	commandDigest 	interface{}		//!< data for the operation that the client needs to the operation to be done
	view 	int
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
		pbft.SendRPCs(command, PRE_PREPARE)
	}

	// if primary, multicast to all the other servers
	pbft.MulticastCommand(command)
	
	// add to the log
	
	// send to everyone else 

	// if a request has been processed already, just reply to the client

}

// General function send RPCs to everyone
func (pbft *PBFT) SendRPCs(command interface{}, phase int) {

	var rpcHandlerName string
	
	switch phase {

	//Use RPC to send preprepare messages to everyone
	case PRE_PREPARE:
		rpcHandlerName = "PBFT.ReceivePrePrepareRPC"

	//Use RPC to send prepare messages to everyone
	case PREPARE:
		rpcHandlerName = "PBFT.HandlePrepareRPC"

	case COMMIT:
		return ok

	case REPLY_TO_CLIENT:
		return ok
	}

	for server := 0; server < serverCount; server++ {
		if server != pbft.serverID {
			ok := pbft.peers[server].Call(rpcHandlerName, command)

			if !ok {
				pbft.peers[server].Call(rpcHandlerName, command)
			}
		}
	}	

}

//!< Handle the RPC to prepare messages
func (pbft *PBFT) HandlePrepareRPC(args PrepareCommandArg) {
	
	// TODO: cast this to the right data type

	// do nothing if we did not receive a prepare
	pbft.serverLock.Lock()
	if _, ok1 := pbft.serverLog[args.sequenceNumber]; !ok1 {
		pbft.serverLock.Unlock()
		return
	}

	logEntryItem, _ := pbft.serverLog[args.sequenceNumber]
	view := bft.view
	pbft.serverLock.Unlock()

	// TODO: check for the sequenceNumber ranges for the watermarks
	
	// do not accept of different views
	if (logEntryItem.view != view) || (args.Digest != logEntryItem.Digest) {
		return
	}

	
	
	startCommit := false
	pbft.serverLock.Lock()
	serverCount := len(pbft.peers)
	prepareCount := pbft.serverLog[args.sequenceNumber].prepareCount + 1
	pbft.serverLog[args.sequenceNumber].prepareCount = prepareCount
	pbft.serverLock.Unlock()

	// go into the commit phase for this command after 2F + 1 replies
	f := (serverCount - 1) / 3
	majority := 2*f + 1

	if (prepareCount >= majority) {
		
	}
	
}

// MulticastCommand RPC handler.
//
// \param args Arguments for command multicast
// \param reply Reply for command multicast
func (pbft *PBFT) HandlePrePrepareRPC(args PrePrepareCommandArg) {

	pbft.serverLock.Lock()
	view := pbft.view
	serverCount := len(pbft.peers)
	pbft.serverLock.Unlock()

	f := (serverCount - 1) / 3
	majority := 2*f + 1

	// TODO: do nothing if not coming from leader 

	// check the view
	if (view != pbft.view) {
		return
	}

	// check if view and sequence number have not been seen
	preInPrepareLog := false
	if logEntryItem, ok1 := pbft.serverLog[args.sequenceNumber]; ok1 {
		
		// if a request has been processed already, just reply to the client
		if (logEntryItem.commitCount == majority){
			pbft.replyToClient()
			return
		}

		if (logEntryItem.view == args.View) {
			preInPrepareLog = true	// already in the prePrepareLog

			match := true; // TODO: change this to the right check
 
			// check that the digests match
			if (!match) {
				return
			}
		}
	}

	// if not preInPrepareLog add to the log
	if !preInPrepareLog {
		pbft.serverLock.Lock()
		bft.serverLog[args.sequenceNumber] = logEntry {
												Message: args.Message
												prepareCount: 1,
												commitCount:  0, 
												prepared: true,	
												commandDigest: args.Digest,
												view: args.View
											}
		pbft.serverLock.Unlock()
	}

	// broadcast prepare messages to everyone
	prepareCommand := PrepareCommandArg { 
							View: args.View, 
							SequenceNumber: args.SequenceNumber,
							Digest: args.Digest,
							SenderIndex: bft.view
						}

	pbft.SendRPCs(prepareCommand, PREPARE)
}

// reply to the client
// TODO: implement this
func (pbft *PBFT) replyToClient() {
	
}

// Handles the RPC from a other servers that send start request to the leader
func (pbft *PBFT) ReceivePrePrepareRPC(command interface{}) {
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
												Message: command /* dummy */}
	
	for server := 0; server < serverCount; server++ {
		if server != pbft.serverID {
			ok := pbft.peers[server].Call("PBFT.HandlePrePrepareRPC", prePrepareCommandArg)

			if !ok {
				pbft.peers[server].Call("PBFT.HandlePrePrepareRPC", prePrepareCommandArg)
			}
		}
	}	
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

