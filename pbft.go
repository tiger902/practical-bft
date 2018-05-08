package pbft

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/rsa"
	"crypto/ecdsa"
	"encoding/gob"
	"errors"
	"labrpc"
	"log"
	"hashstructure"
	"math/big"
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
	VIEW_CHANGE 	= 5
	CHECK_POINT 	= 6
)

const (
	IDLE 				= 0
	PROCESSING_COMMAND	= 1
	CHANGING_VIEW	    = 2
)

// protocol constants
const (
	CHECK_POINT_INTERVAL = 100
)


//! A Go object implementing a single PBFT peer.
type PBFT struct {
	privateKey  PrivateKey 			//!< Private key for this server
	publicKeys     []PublicKey 		//!< Array of publick keys for all servers
	peers     []*labrpc.ClientEnd 	//!< Array of all the other server sockets for RPC
	numberOfServers int 	
	persister *Persister          	//!< Persister to be used to store data for this server in permanent storage
	serverID        int           	//!< Index into peers[] for this server
	sequenceNumber  int				//!< last sequence number 
	commandsChannel chan int		//!< channel for commands
	uncommittedCommands int			//!< store the number of commands that we are waiting for
	state 				int 		//!< the state of the server
	lastCheckPointSeqNumber int 	//!< the sequence number of the last checkpoint
	view			int 				//!< the current view of the system
	serverLog 		map[int]logEntry	//!< to keep track of all the commands that have been seen regard less of the stage
	serverLock      sync.Mutex 			//!< Lock to be when changing the sequenceNumber,
	clientRegisters map[int]time.Time	// to keep track of the last reply that has been sent to this client
	storedState		interface{}			// this is the state that the application is keeping for this application
	checkPoints 	map[int]CheckPointInfo
	//lastCheckPointDigest uint64			//!< Digest of the last checkpoint
}

//!struct used for preprepare command arguments before they have been signed
type PrePrepareCommandArg struct {
	View         	int        	//!< leader’s term
	SequenceNumber 	int			//!< Sequence number of the messsage
	Digest 			uint64 		//!< Digest of the message, which can is an uint64 TODO: change type when we switch to SHA256
	Message 		interface{} //!< Message for the command TODO: decouple this
}

//!struct used as argument to multicast command
type PrepareCommandArg struct {
	View         	int        	//!< leader’s term
	SequenceNumber 	int			//!< Sequence number of the messsage
	Digest 			uint64 //!< Digest of the message, which can is an int TODO: check the type of this
	SenderIndex 	int 		//!< Id of the server that sends the prepare message
}

//!struct used as argument to multicast command
type CommitArg struct {
	View         	int        	//!< leader’s term
	SequenceNumber 	int			//!< Sequence number of the messsage
	Digest 			uint64 		//!< Digest of the message, which can is an int TODO: check the type of this
	SenderIndex 	int 		//!< Id of the server that sends the prepare message
}

//!struct used as argument to multicast command
type CheckPointArgs struct {
	SequenceNumber int			//!< the last sequence number that is reflected in the checkpoint
	Digest 		   uint64		//!< digest of the state of the server that is being checkpoint-ed
	SenderIndex	   int			//!< ID of the server
}

//! struct for the view change arguments
type ViewChange struct {
	NewView int 
	LastCheckPointSequenceNumber int 
	LastCheckPointMessages interface{}
	PreparedMessages interface{}
	ServerID int 
}

//!struct used as argument to multicast command
type CommandArgs struct {
	SpecificArguments 	interface{}
	R_firstSig			Int
	S_secondSig 		Int
}

//!struct used to store the log entry
type logEntry struct {
	message 		interface{}
	prepareCount 	int
	commitCount 	int 
	prepared 		bool
	commandDigest 	uint64		   
	view 			int
	clientReplySent	int
}

//! struct that keeps the checkpoints of the server
type CheckPointInfo {
	LargestSequenceNumber 	int				//!< The largest sequence number for the checkpoint
	CheckPointState 	 	interface{}		//!< The state of the server that is part of the checkpoint
	ConfirmedServers	  	map[int]bool 	//!< The servers that have accepted and verified the checkpoint
}

//! returns whether this server believes it is the leader.
func (pbft *PBFT) GetState() (int, bool) {

	return false
}

// Write the relavant state of PBFT to persistent storage
func (pbft *PBFT) persist() {
	
}

//! Restore previously persisted state.
func (pbft *PBFT) readPersist(data []byte) {
	
}

func (pbft *PBFT) makeArguments(specificArgument interface{}) {
	// make the hash to be used for making the signatures, and then sign the message
	hash, err = Hash(specificArgument, nil)
	if err != nil {
		panic(err)
	}

	r, s, err1 := Sign(rand.Reader, pbft.privateKey, hash)
	if err1 != nil {
		panic(err1)
	}

	return CommandArgs {
			SpecificArguments: specificArgument,
			R_firstSig: r,
			S_secondSig: s
		}
}

// starts a new command
// return the 
func (pbft *PBFT) Start(clientCommand Command) {

	// TODO: verify that the messages have not been changed on their way from the client


	// if not primary, send the request to the primary
	pbft.serverLock.Lock()
	leader := pbft.view % pbft.numberOfServer

	if (pbft.serverID != leader) {
		pbft.serverLock.Unlock()
		pbft.SendRPCs(command, FORWARD_COMMAND)
		return
	}

	// do nothing if the time is before what we have sent already
	if lastReplyTimestamp, ok := bft.clientRegisters[clientCommand.ClientID]; ok {
		if  clientCommand.Timestamp.Before(lastReplyTimestamp) {
			pbft.serverLock.Unlock()
			return
		}
	}
	
	view := bft.view
	sequenceNumber := bft.sequenceNumber++
	pbft.serverLock.Unlock()
	
	// make a digest for the command from the clint
	hash, err := Hash(clientCommand, nil)
	if err != nil {
		panic(err)
	}

	prePrepareCommandArgs :=  PrePrepareCommandArg {
								View  : view,
								SequenceNumber : sequenceNumber,
								Digest : hash,
								Message : clientCommand
							}

	// multicast to all the other servers
	go pbft.SendRPCs(pbft.makeArguments(prePrepareCommandArgs), PRE_PREPARE)

	// process the recived command accordingly
	pbft.addLogEntry(&prePrepareCommandArgs)
}

// General function send RPCs to everyone
func (pbft *PBFT) SendRPCs(command CommandArgs, phase int) {

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
		return

	//Use RPC to send preprepare messages to everyone
	case PRE_PREPARE:
		rpcHandlerName = "PBFT.HandlePrePrepareRPC"

	//Use RPC to send prepare messages to everyone
	case PREPARE:
		rpcHandlerName = "PBFT.HandlePrepareRPC"

	case COMMIT:
		rpcHandlerName = "PBFT.HandleCommitRPC"

	case VIEW_CHANGE:
		rpcHandlerName = "PBFT.HandleViewChangeRPC"

	case CHECK_POINT:
		rpcHandlerName = "PBFT.HandleCheckPointRPC"

	case REPLY_TO_CLIENT:
		return
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

// helper function for checking that the Digest match
inline func verifyDigests(arg interface{}, digest uint64) {

	messageDigest, errDigest := Hash(arg, nil)
	if errDigest != nil {
		panic(errDigest)
	}
	if (messageDigest != digest) {
		return false
	}

	return true
}


// helper function for checking that the Digest match
func (pbft *PBFT) verifySignatures(args *interface{}, r_firstSig *Int, s_secondSig *Int, peerID int) {

	// hash the received command so that we can use the hash for verification of the signatures 
	hash, err := Hash(args, nil)
	if err != nil {
		panic(err)
	}

	// check that the signature of the prepare command match
	if !Verify(&bft.publicKey[peerID], hash, r_firstSig, s_secondSig) {
		return false
	}
	return true
}

//  HandlePrePrepareRPC receives and processes preprepare commands
//
// \param args 
func (pbft *PBFT) HandlePrePrepareRPC(args CommandArgs) {

	preprepareCommandArgs := PrePrepareCommandArg(args.SpecificArguments)

	// return of the message digest does not match with the one that will be calculated here
	if !verifyDigests(preprepareCommandArgs.Message, preprepareCommandArgs.Digest) {
		return
	}
	
	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()
	view := pbft.view
	serverCount := len(pbft.peers)
	leader := pbft.view % serverCount
	isLeader := (leader == pbft.serverID) ? true : false
	
	// check that the signature of the prepare command match
	if !bft.verifySignatures(&preprepareCommandArgs, &args.R_firstSig, &args.S_secondSig, leader) {
		return
	}

	// check the view
	if (view != preprepareCommandArgs.View) {
		return
	}

	// add entry to the log
	bft.addLogEntry(&preprepareCommandArgs)

	// broadcast prepare messages to everyone if you are not the leader
	prepareCommand := PrepareCommandArg { 
						View: preprepareCommandArgs.View, 
						SequenceNumber: preprepareCommandArgs.SequenceNumber,
						Digest: preprepareCommandArgs.Digest,
						SenderIndex: bft.serverID
					}


	go pbft.SendRPCs(pbft.makeArguments(prepareCommand), PREPARE)
}


// helper function to add a command to the log
func (pbft *PBFT) addLogEntry(args *PrePrepareCommandArg) {

	// TODO: check the range of the sequence numbers
	
	f := (pbft.numberOfServers - 1) / 3
	majority := 2*f + 1

	// check if view and sequence number have not been seen. Reply to the client if the entry has been
	// processed already. If it has not been processed and the entry at the same sequence number and view
	// does not match the given entry, drop the packet
	if logEntryItem, ok1 := pbft.serverLog[args.sequenceNumber]; ok1 {
		
		// if a request has been processed already, just reply to the client
		if (logEntryItem.commitCount == majority){
			go pbft.replyToClient()
			return
		}

		if (logEntryItem.view == args.View) {
			return
		}
	}

	// if not preInPrepareLog add to the log
	bft.serverLog[args.sequenceNumber] = logEntry {
											Message: args.Message
											prepareCount: 1,
											commitCount:  0, 
											prepared: true,	
											commandDigest: args.Digest,
											view: args.View,
											clientReplySent: false
										}
}


//!< Handle the RPC to prepare messages
func (pbft *PBFT) HandlePrepareRPC(args CommandArgs) {

	prepareArgs := PrepareCommandArg(args.SpecificArguments)
	
	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()
	
	// check that the signature of the prepare command match
	if !bft.verifySignatures(&prepareArgs, &args.R_firstSig, &args.S_secondSig, prepareArgs.SenderIndex) {
		return
	}

	logEntryItem, ok := pbft.serverLog[prepareArgs.sequenceNumber]

	// A replica (including the primary) accepts prepare messages and adds them to its log
	// provided their signatures are correct, their view number equals the replica’s current view,
	// and their sequence number is between h and H.
	
	// do nothing if we did not receive a preprepare
	if !ok1 {
		return
	}

	// do not accept of different views, or different signatures
	if (prepareArgs.View != bft.view) || (prepareArgs.Digest != logEntryItem.Digest) {
		return
	}

	// TODO: check for the sequenceNumber ranges for the watermarks


	pbft.serverLog[prepareArgs.sequenceNumber].prepareCount++

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
		pbft.serverLog[prepareArgs.sequenceNumber].commitCount++
 	   commitArgs := CommitArg{
						View: bft.view,
						SequenceNumber: prepareArgs.sequenceNumber,
						Digest: prepareArgs.Digest,
						SenderIndex; bft.serverID 
					}
	
		go bft.SendRPCs(pbft.makeArguments(commitArgs), COMMIT)
	}
}

// Handles the commit RPC 
func (pbft *PBFT) HandleCommitRPC(args CommandArgs) {

	commitArgs := PrepareCommandArg(args.SpecificArguments)
	
	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()


	// Replicas accept commit messages and insert them in their log provided 
	// they are properly signed, the view number in the message is equal to the replica’s 
	// current view, and the sequence number is between h and H

	// TODO: implement the h and H stuff

	if commitArgs.View != bft.view {
		return
	}

	logEntryItem, ok := pbft.serverLog[commitArgs.sequenceNumber]
	
	// do nothing if we did not receive a preprepare
	if !ok1 {
		return
	}

	// do nothing if this entry has not been prepared
	// TODO: is this the behavior we want? 
	if !logEntryItem.prepared {
		return
	}

	// check that the signature of the prepare command match
	if !bft.verifySignatures(&commitArgs, &args.R_firstSig, &args.S_secondSig, commitArgs.SenderIndex) {
		return
	}

	// do nothing if the reply has been sent already
	if logEntryItem.clientReplySent {
		return
	}
	
	// go into the commit phase for this command after 2F + 1 replies
	f := (pbft.serverCount - 1) / 3
	majority := 2*f + 1

	logEntryItem.commitCount++

	if ((logEntryItem.commitCount >= majority) && !logEntryItem.clientReplySent) {
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


// Handles the view change RPC 
func (pbft *PBFT) HandleViewChangeRPC(args CommandArgs) {

	viewChange := ViewChange(args.SpecificArguments)
	
	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()
}

func (pbft *PBFT) HandleCheckPointRPC(args CommandArgs) {
	checkPointArgs := CheckPointArgs(args.SpecificArguments)

	// check that the signature of the prepare command match
	if !bft.verifySignatures(&checkPointArgs, &args.R_firstSig, &args.S_secondSig, checkPointArgs.SenderIndex) {
		return
	}

	// return if you do not have this checkpoint started, otherwise check the digests
	checkPointInfo, ok := bft.CheckPointInfo[checkPointArgs.LargestSequenceNumber];
	if !ok {
		return 
	}
	if !verifyDigests(checkPointInfo, checkPointArgs.Digest) {
		return
	}

	bft.serverLock.Lock()
	bft.CheckPointInfo[checkPointArgs.LargestSequenceNumber].ConfirmedServers[checkPointArgs.SenderIndex] = true
	bft.serverLock.UnLock()

	// TODO: clean up the other checkpoints and the logs if this checkpoint is stable

}


// helper function to make a checkpoint
func (pbft *PBFT) makeCheckpoint(checkPointInfo *CheckPointInfo) {


	// TODO: the person who calls this must create the pointer
	/*pbft.serverLock.Lock()
	checkPointInfo := CheckPointInfo {
		LargestSequenceNumber: sequenceNumber,
		CheckPointState: bft.storedState,
		ConfirmedServers: make(map[int]bool)
	}
	
	bft.checkPoints[sequenceNumber] = checkPointInfo
	pbft.serverLock.Unlock()*/

	// hash the data that is part of the checkpoint
	hash, err := Hash(checkPointInfo.CheckPointState, nil)
	if err != nil {
		panic(err)
	}
	
	checkPointArgs := CheckPointArgs {
		SequenceNumber : checkPointInfo.LargestSequenceNumber,
		Digest : hash
		SenderIndex : bft.serverID
	}

	pbft.SendRPCs(pbft.makeArguments(checkPointArgs), CHECK_POINT)

}


// reply to the client
// TODO: implement this
func (pbft *PBFT) replyToClient(clientCommandReply CommandReply) {
	
	// TODO: update the timestamp of the last sent reply
	// using the clientRegisters log
}

// helper function to change the view
func (pbft *PBFT) startViewChange() {
	pbft.serverLock.Lock()

	viewChange := ViewChange {
			NewView: bft.view + 1, 
			LastCheckPointSequenceNumber: lastCheckPointSeqNumber,
			LastCheckPointMessages: interface{},
			PreparedMessages: interface{},
			ServerID: bft.serverID 
		}
	pbft.serverLock.Unlock()
	pbft.SendRPCs(pbft.makeArguments(viewChange), VIEW_CHANGE)
}

// Handles the RPC from a other servers that send start request to the leader
func (pbft *PBFT) ReceiveForwardedCommand(command CommandArgs) {
	pbft.Start(command)
}

// Processes commands in a loop as they come from commandsChannel
func (pbft *PBFT) runningState() {

	const T = 150	//TODO: this need to be changed
	timer := &Timer{}

	bft.state := IDLE

	for {

		select {
		case <-timer.C:
			if !timer.Stop() {
				<-timer.C
			}
			if (bft.state == PROCESSING_COMMAND) {
				bft.state = CHANGING_VIEW
				bft.startViewChange()
			}

		// increment the number of commands we are waiting for and 
		// reset the timer if we are in the IDLE state
		case <-pbft.newValidCommad:
			bft.uncommittedCommands++
			if (bft.state == IDLE) {
				bft.state = PROCESSING_COMMAND
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(time.Millisecond * time.Duration(T))
			}

		// decrement the number of commands we are waiting for and 
		// stop the timer if there is no more commands to run, otherwise restart
		// the timer
		case <-pbft.commandExecuted:
			bft.uncommittedCommands--
			if (bft.uncommittedCommands < 0) {
				panic(errors.New("uncommittedCommands is 0"))
			}
			if (bft.state == PROCESSING_COMMAND) {
				if !timer.Stop() {
					<-timer.C
				}
				
				if (bft.uncommittedCommands == 0) {
					bft.state = IDLE
				} else {
					timer.Reset(time.Millisecond * time.Duration(T))
				}
			}

		// TODO: this means that we will need to bft.uncommittedCommands to zero when we are
		// done with the view change
		case <-pbft.viewChangeComplete:
			if (bft.uncommittedCommands == 0) {
				bft.state = IDLE
			} else {
				bft.state = PROCESSING_COMMAND
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
func Make(privateKey PrivateKey, publicKeys []PublicKey, peers []*labrpc.ClientEnd, serverID int,
		  persister *Persister, applyCh chan ApplyMsg) *PBFT {

	pbft := &PBFT{}
	pbft.peers = peers
	pbft.persister = persister
	pbft.serverID = serverID
	pbft.sequenceNumber = 0
	pbft.numberOfServers = length(peers)

	// TODO: make sure to update the variable from persist

	// start a go routine to make sure that we receive heartbeats from the primary
	// this will essentially be a time out that then intiates a view change

	// add more code and return the new server

	// start a go routine for handling command
	// TODO: change the size dynamically
	pbft.commandsChannel = make(chan interface{}, 2*len(peers))
	go runningState()

	return PBFT
}

