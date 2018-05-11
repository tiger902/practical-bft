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

/*
*
* https://golang.org/pkg/net/rpc/
* Not too difficult to implement but I'm not sure of a better solution than hard coding IP addresses.
* We'll need to have everything implemented locally first (the hard part) then transitioning to this will be easy
* See https://www.cs.princeton.edu/courses/archive/fall17/cos418/docs/P3-go-rpc.zip
*/

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
	newLeader := (pbft.view + 1) % pbft.numberOfServer
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
		ok := pbft.peers[newLeader].Call(rpcHandlerName, command)
		if !ok {
			ok = pbft.peers[newLeader].Call(rpcHandlerName, command)
		}
		return

	case NEW_VIEW:
		rpcHandlerName = "PBFT.HandleNewViewRPC"

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
											Message: args.Message,
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
	// TODO: verify signatures of this thing

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
		
		// perform the checkpointing if this is the right moment
		if (commitArgs.sequenceNumber / CHECK_POINT_INTERVAL) == 0 {
			checkPointInfo := CheckPointInfo {
				LargestSequenceNumber: commitArgs.sequenceNumber,
				CheckPointState: bft.storedState,
				ConfirmedServers: make(map[int]bool)
			}
			bft.checkPoints[commitArgs.sequenceNumber] = checkPointInfo
			bft.lastCheckPointSeqNumber = commitArgs.sequenceNumber
			go bft.makeCheckpoint(checkPointInfo) 
		}

		go bft.replyToClient(clientCommandReply)
	}
}


// Handles the view change RPC 
func (pbft *PBFT) HandleViewChangeRPC(args CommandArgs) {

	viewChange := ViewChange(args.SpecificArguments)

	// do nothing is this is not for this server
	if ((viewChange.NextView % bft.serverCount) != bft.serverID) {
		return
	}
	
	// check that the signature of the prepare command match
	if !bft.verifySignatures(&viewChange, &args.R_firstSig, &args.S_secondSig, viewChange.SenderIndex) {
		return
	}

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()

	// return if this server's view is more than that of the sender
	// of this server is not supposed to be a leader
	if ((viewChange.NextView <= bft.view) || ((viewChange.NextView % bft.serverCount)  != bft.view)) {
		return
	}

	// TODO: verify all the prepare messages that are in the viewChange argument to make
	// sure that they are all the same
	// LastCheckPointMessages and PreparedMessages needs to be verified

	// make a new map if the array of the view change messages does not exist yet
	viewChanges, ok := bft.viewChanges[viewChange.NextView] 
	if !ok {
		bft.viewChanges[viewChange.NextView] = make(map[int]CommandArgs)
	} else {
		if _, ok1 := viewChanges[viewChange.ServerID]; ok1 {
			return
		}
	}
 
	bft.viewChanges[viewChange.NextView][viewChange.ServerID] =  args

	// if we now have the majority of view change requests
	if (len(bft.viewChanges[viewChange.NextView]) >= 2 * bft.failCount) {

		bft.viewChanges[viewChange.NextView][bft.serverID] = bft.makeViewChangeArguments()

		allPreprepareMessage := make([]CommandArgs)
		createPreprepareMessages(viewChange.NextView, &allPreprepareMessage)

		// append to the log all the entries
		for _, preprepareMessage := range allPreprepareMessage {
			pbft.addLogEntry(&preprepareMessage.SpecificArguments)
		}

		bft.view = viewChange.NextView

		// send the new view to everyone
		newView := NewView {
			NextView: viewChange.NextView,
			ViewChangeMessages: bft.viewChanges[viewChange.NextView],
			PreprepareMessage: allPreprepareMessage
		}

		go pbft.SendRPCs(pbft.makeArguments(newView), NEW_VIEW)
	}

}

//!< Function to handle the new view RPC from the leader
func (pbft *PBFT) HandleNewViewRPC(args CommandArgs) {
	newView := NewView(args.SpecificArguments)

	bft.serverLock.Lock()
	defer bft.serverLock.Unlock()

	// do nothing if we are already at a higher view
	if (newView.NextView < bft.view) {
		return
	}

	// find the new leader
	var newLeader int
	for serverID := 0; serverID < bft.serverCount;  serverID++{
		if ((newView.NextView % bft.serverCount) == serverID) {
			newLeader = serverID
			break
		}
	}

	// check that the signature of the prepare command match
	if !bft.verifySignatures(&newView, &args.R_firstSig, &args.S_secondSig, newLeader) {
		return
	}

	// check the validity of view change messages
	for serverID, viewChangeArgs := range newView.ViewChangeMessages {
		viewChangeMessage := NewView(viewChangeArgs.SpecificArguments)
		if viewChangeMessage.NextView != newView.NextView {
			return
		}

		if !bft.verifySignatures(&viewChangeMessage, &viewChangeArgs.R_firstSig, &viewChangeArgs.S_secondSig, serverID) {
			return
		}
	}

	// check the validity of prepare messages and compare to those that we
	// got from the new leader
	allPreprepareMessage := make([]CommandArgs)
	createPreprepareMessages(newView.NextView, &allPreprepareMessage)

	// TODO: perform the verification but this is not needed since it's redundant
	
	// append to the log all the entries
	for _, preprepareMessage := range allPreprepareMessage {

		preprepareNoClientMessage := PreprepareWithNoClientMessage(preprepareMessage.SpecificArguments)
		pbft.addLogEntry(preprepareMessage)

		// broadcast prepare messages to everyone if you are not the leader
			prepareCommand := PrepareCommandArg { 
				View: preprepareNoClientMessage.View, 
				SequenceNumber: preprepareNoClientMessage.SequenceNumber,
				Digest: preprepareNoClientMessage.Digest,
				SenderIndex: bft.serverID
			}

		go pbft.SendRPCs(pbft.makeArguments(prepareCommand), PREPARE)
	}
}

// function calling this should have a lock
func (pbft *PBFT) createPreprepareMessages(nextVeiw int, allPreprepareMessage *[]CommandArgs) {

	// get the max and min stable sequence number in bft.viewChanges[viewChange.NextView]
	minLatestStableCheckpoint := Inf(0) // the minimum latest stable checkpoint
	maxHighestPrepareSequenceNumber := Inf(-1) // the highest prepare message seen

	for _, commandArgs := range bft.viewChanges[nextVeiw] {
		viewChange := ViewChange(commandArgs.SpecificArguments)

		if viewChange.LastCheckPointSequenceNumber < minLatestStableCheckpoint {
			minLatestStableCheckpoint = viewChange.LastCheckPointSequenceNumber

			// TODO: verify that the checkpoint is valid using the LastCheckPointMessages 
		}

		for sequenceNumber, _ := range viewChange.PreparedMessages {
			if sequenceNumber > maxHighestPrepareSequenceNumber {
				maxHighestPrepareSequenceNumber = sequenceNumber
			}
		}
	} 

	for sequenceNumber = minLatestStableCheckpoint; sequenceNumber <= maxHighestPrepareSequenceNumber; sequenceNumber++ {

		foundCommand := false 	// to show whether the command has been seen or not
		var preprepareMessage CommandArgs
		for serverID, commandArgs := range bft.viewChanges[nextVeiw] {
			viewChange := ViewChange(commandArgs.SpecificArguments)
			prepareMForViewChange, ok := viewChange.PreparedMessages[sequenceNumber]
			if ok {
				foundCommand = true
				preprepareMessage = prepareMForViewChange.PreprepareMessage
				break
			}
		}

		var preprepareWithNoClientMessage PreprepareWithNoClientMessage
		if !foundCommand {
			preprepareWithNoClientMessage = PreprepareWithNoClientMessage {
											View: nextVeiw,
											SequenceNumber: sequenceNumber,
											Digest: nil
										}
		} else {
			preprepareWithNoClientMessage = PreprepareWithNoClientMessage(preprepareMessage.SpecificArguments)
			preprepareWithNoClientMessage.View = nextVeiw
		}

		preprepareMessageToSend := pbft.makeArguments(preprepareWithNoClientMessage)
		*allPreprepareMessage = append(*allPreprepareMessage, preprepareMessageToSend)
	} 

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
	bft.CheckPointInfo[checkPointArgs.LargestSequenceNumber].ConfirmedServers[checkPointArgs.SenderIndex] = args

	// for all checkpoints, check to see if we have received a majority
	currentCheckpointInfo := bft.CheckPointInfo[checkPointArgs.LargestSequenceNumber]

	// if this checkpoint is now stable, delete all those that are before this one
	if !currentCheckpointInfo.IsStable {
		if (len(currentCheckpointInfo.ConfirmedServers) >= (2 * bft.failCount)) {
			bft.CheckPointInfo[checkPointArgs.LargestSequenceNumber].IsStable = true
			bft.removeStallCheckpoints(checkPointArgs.LargestSequenceNumber) 
		}
	}
	bft.serverLock.UnLock()
}

// helper function to remove checkpoints that are no longer important to us
// this function should be called by someone who owns the serverLock
func (pbft *PBFT) removeStallCheckpoints(largestStableSequenceNumber int) {
	for _, sequenceNumber := range bft.CheckPointInfo { 
		if sequenceNumber < largestStableSequenceNumber {
			delete bft.CheckPointInfo[sequenceNumber]
		}
	}
}


// helper function to make a checkpoint
func (pbft *PBFT) makeCheckpoint(checkPointInfo CheckPointInfo) {

	// TODO: if there are many checkpoints that are not stable before this one, make sure to ask for the most
	// stable checkpoint from the majority first

	// hash the data that is part of the checkpoint
	hash, err := Hash(checkPointInfo.CheckPointState, nil)
	if err != nil {
		panic(err)
	}
	
	checkPointArgs := CheckPointArgs {
		SequenceNumber : checkPointInfo.LargestSequenceNumber,
		Digest : hash,
		SenderIndex : bft.serverID
	}

	pbft.SendRPCs(pbft.makeArguments(checkPointArgs), CHECK_POINT)

}


// reply to the client
func (pbft *PBFT) replyToClient(clientCommandReply CommandReply) {
	// TODO: implement this
	// TODO: update the timestamp of the last sent reply
	// using the clientRegisters log
}

// helper function to change the view
func (pbft *PBFT) startViewChange() {

	pbft.serverLock.Lock()
	viewChangeArgs := bft.makeViewChangeArguments()
	pbft.serverLock.Unlock()
	pbft.SendRPCs(viewChangeArgs, VIEW_CHANGE)
}

// Helper method for making a view change
// should be called by someone who holds a lock
func (pbft *PBFT) makeViewChangeArguments() {

	// make the PrepareForViewChange by looping through all the entries in the
	// log that are prepared but not yet committed
	prepareForViewChange := make(map[int]PrepareMForViewChange)

	for sequenceNumber, logEntry := range bft.serverLog { 
		if (logEntry.prepared && !logEntry.committed)  {

			prepareMForViewChange := PrepareMForViewChange {
				PreprepareMessage:  logEntry.preprepare.PreprepareNoClientMessage,			
				PrepareMessages:    logEntry.prepareArgs
			}
			prepareForViewChange[sequenceNumber] = prepareMForViewChange
		}
	}

	viewChange := ViewChange {
			NewView: bft.view + 1, 
			LastCheckPointSequenceNumber: bft.lastCheckPointSeqNumber,
			LastCheckPointMessages: bft.CheckPointInfo[bft.lastCheckPointSeqNumber].ConfirmedServers,
			PreparedMessages: prepareForViewChange,
			ServerID: bft.serverID 
		}
	return pbft.makeArguments(viewChange)
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

