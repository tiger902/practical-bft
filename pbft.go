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
	FORWARD_COMMAND = 0
	PRE_PREPARE		= 1
	PREPARE	     	= 2
	COMMIT 			= 3
	REPLY_TO_CLIENT	= 4
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
	serverLog 		map[int]logEntry	//!< to keep track of all the commands that have been seen regard less
										//!< of the stage
	prePreparedLog 	map[int]prePrepare	//!< to 
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
	Digest 			int //!< Digest of the message, which can is an int TODO: check the type of this
	SenderIndex 	int 		//!< Id of the server that sends the prepare message
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
	SenderIndex 	int //!< Id of the server that sends the prepare message
}

//!struct used as argument to multicast command
type CommitArg struct {
	View         	int        	//!< leader’s term
	SequenceNumber 	int			//!< Sequence number of the messsage
	Digest 			int 		//!< Digest of the message, which can is an int TODO: check the type of this
	SenderIndex 	int 		//!< Id of the server that sends the prepare message
}

//!struct used to store the log entry
type logEntry struct {
	message 		interface{}		//!< data for the operation that the client needs to the operation to be done
	prepareCount 	int
	commitCount 	int 
	prepared 		bool
	commandDigest 	interface{}		//!< data for the operation that the client needs to the operation to be done
	view 			int
	clientReplySent		int
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
	pbft.serverLock.Lock()
	leader := pbft.view % pbft.numberOfServer

	if (pbft.serverID != leader) {
		pbft.serverLock.Unlock()
		pbft.SendRPCs(command, FORWARD_COMMAND)
		return
	}
	
	prePrepareCommandArgs := PrePrepareCommandArg {
								View  : bft.view,
								SequenceNumber : bft.sequenceNumber++,
								Digest : interface{},
								Message : command
							}
	pbft.serverLock.Unlock()

	// process the recived command accordingly
	pbft.processPrePrepare(prePrepareCommandArgs)

	// multicast to all the other servers
	pbft.SendRPCs(prePrepareCommandArgs, PRE_PREPARE)

}

// General function send RPCs to everyone
func (pbft *PBFT) SendRPCs(command interface{}, phase int) {

	var rpcHandlerName string
	
	pbft.serverLock.Lock()
	leader := pbft.view % pbft.numberOfServer
	serverCount := pbft.numberOfServer
	pbft.serverLock.Unlock()
	
	switch phase {

	//Use RPC to the leader if command was sent to a non-leader
	case FORWARD_COMMAND:
		rpcHandlerName = "PBFT.ReceiveForwardedCommand"
		ok := pbft.peers[leader].Call(rpcHandlerName, command)
		if !ok {
			ok = pbft.peers[leader].Call(rpcHandlerName, command)
		}
		return ok

	//Use RPC to send preprepare messages to everyone
	case PRE_PREPARE:
		rpcHandlerName = "PBFT.HandlePrePrepareRPC"

	//Use RPC to send prepare messages to everyone
	case PREPARE:
		rpcHandlerName = "PBFT.HandlePrepareRPC"

	case COMMIT:
		rpcHandlerName = "PBFT.HandleCommitRPC"

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

//  HandlePrePrepareRPC receives and processes preprepare commands
//
// \param args 
func (pbft *PBFT) HandlePrePrepareRPC(args interface{}) {
	pbft.processPrePrepare(PrePrepareCommandArg(args))
}

//  processPrePrepare helper method to process and preprepare command 
//
// \param args 
func (pbft *PBFT) processPrePrepare(arg PrePrepareCommandArg) {

	pbft.serverLock.Lock()
	view := pbft.view
	serverCount := len(pbft.peers)
	isLeader := ((view % serverCount) == pbft.serverID) ? true : false
	pbft.serverLock.Unlock()

	f := (serverCount - 1) / 3
	majority := 2*f + 1

	// TODO: do nothing if not coming from leader, this is going to be done using crypo

	// check the view
	if (view != args.View) {
		return
	}

	// check if view and sequence number have not been seen. Reply to the client if the entry has been
	// processed already. If it has not been processed and the entry at the same sequence number and view
	// does not match the given entry, drop the packet
	preInPrepareLog := false
	if logEntryItem, ok1 := pbft.serverLog[args.sequenceNumber]; ok1 {
		
		// if a request has been processed already, just reply to the client
		if (logEntryItem.commitCount == majority){
			pbft.replyToClient()
			return
		}

		if (logEntryItem.view == args.View) {
			preInPrepareLog = true	// already in the prePrepareLog

			match := true; // TODO: change this to the right check using crypto
 
			// do nothing if the digest does not match the item at this entry
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
												view: args.View,
												clientReplySent: false
											}
		pbft.serverLock.Unlock()
	}

	// broadcast prepare messages to everyone if you are not the leader
	if !isLeader {
		prepareCommand := PrepareCommandArg { 
							View: args.View, 
							SequenceNumber: args.SequenceNumber,
							Digest: args.Digest,
							SenderIndex: bft.serverID
						}
		pbft.SendRPCs(prepareCommand, PREPARE)
	}
}

//!< Handle the RPC to prepare messages
func (pbft *PBFT) HandlePrepareRPC(args interface{}) {
	
	args := PrepareCommandArg(arg)
	
	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()
	
	logEntryItem, ok := pbft.serverLog[args.sequenceNumber]

	// A replica (including the primary) accepts prepare messages and adds them to its log
	// provided their signatures are correct, their view number equals the replica’s current view,
	// and their sequence number is between h and H.
	
	// do nothing if we did not receive a preprepare
	if !ok1 {
		return
	}

	// do not accept of different views, or different signatures
	if (arg.View != bft.view) || (args.Digest != logEntryItem.Digest) {
		return
	}

	// TODO: check for the sequenceNumber ranges for the watermarks


	pbft.serverLog[args.sequenceNumber].prepareCount++

	// return if already prepared
	if logEntryItem.prepared {
		return
	}

	// We define the predicate prepared to be true iff replica has inserted in its 
	// log: the request, a pre-prepare for in view with sequence number, and 2f
	// prepares from different backups that match the pre-prepare. The replicas verify 
	// whether the prepares match the pre-prepare by checking that they have the
	// same view, sequence number, and digest

	// go into the commit phase for this command after 2F + 1 replies
	f := (pbft.serverCount - 1) / 3
	majority := 2*f + 1

	if (logEntryItem.prepareCount >= majority) {
		logEntryItem.prepared = true
		pbft.serverLog[args.sequenceNumber].commitCount++
 	   commitArgs := CommitArg{
						View: bft.view,
						SequenceNumber: args.sequenceNumber,
						Digest: args.Digest,
						SenderIndex; bft.serverID 
					}
		go bft.SendRPCs(commitArgs, COMMIT)
	}
}

// Handles the commit RPC 
func (pbft *PBFT) HandleCommitRPC(arg interface{}) {

	args := CommitArg(arg)
	
	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()


	// Replicas accept commit messages and insert them in their log provided 
	// they are properly signed, the view number in the message is equal to the replica’s 
	// current view, and the sequence number is between h and H

	// TODO: implement the h and H stuff

	if args.View != bft.view {
		return
	}

	logEntryItem, ok := pbft.serverLog[args.sequenceNumber]
	
	// do nothing if we did not receive a preprepare
	if !ok1 {
		return
	}

	// do nothing if this entry has not been prepared
	// TODO: is this the behavior we want? 
	if !logEntryItem.prepared {
		return
	}

	// check the signatures of the commit messages
	// TODO: add the signatures checks

	// do nothing if the reply has been sent already
	if logEntryItem.clientReplySent {
		return
	}
	
	// go into the commit phase for this command after 2F + 1 replies
	f := (pbft.serverCount - 1) / 3
	majority := 2*f + 1

	if (logEntryItem.commitCount >= majority) {
		logEntryItem.clientReplySent = true
		clientCommand := Command(logEntryItem.Message)
 	   	clientCommandReply := CommandReply {
							CurrentView : bft.view,
							RequestTimestamp: clientCommand.Timestamp, 
							ClientID: clientCommand.ClientID,
							ServerID: bft.serverID,
							ResultData: interface{}
						}
		go bft.replyToClient(clientCommandReply)
	}

}


// reply to the client
// TODO: implement this
func (pbft *PBFT) replyToClient(clientCommandReply CommandReply) {
	
	// TODO: update the timestamp of the last sent reply
}


// Handles the RPC from a other servers that send start request to the leader
func (pbft *PBFT) ReceiveForwardedCommand(command interface{}) {
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

