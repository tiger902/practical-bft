package pbft

import (
	"time"
	"errors"
	"crypto/rand"
	"math/big"
)

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

// Helper method for making a view change
// should be called by someone who holds a lock
func (pbft *PBFT) makeViewChangeArguments() CommandArgs {

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
	pbft.sendRPCs(viewChangeArgs, VIEW_CHANGE)
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

	pbft.sendRPCs(pbft.makeArguments(checkPointArgs), CHECK_POINT)

}


// General function send RPCs to everyone
func (pbft *PBFT) sendRPCs(command CommandArgs, phase int) {

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
func verifyDigests(arg interface{}, digest uint64) bool {

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
func (pbft *PBFT) verifySignatures(args *verifySignatureArg, r_firstSig *big.Int, s_secondSig *big.Int, peerID int) bool {

	// hash the received command so that we can use the hash for verification of the signatures 
	hash, err := Hash(args, nil)
	if err != nil {
		panic(err)
	}

	// check that the signature of the prepare command match
	if !Verify(&pbft.publicKey[peerID], hash, r_firstSig, s_secondSig) {
		return false
	}
	return true
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

// helper function to remove checkpoints that are no longer important to us
// this function should be called by someone who owns the serverLock
func (pbft *PBFT) removeStallCheckpoints(largestStableSequenceNumber int) {
	for _, sequenceNumber := range bft.CheckPointInfo { 
		if sequenceNumber < largestStableSequenceNumber {
			delete bft.CheckPointInfo[sequenceNumber]
		}
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

// Handles the RPC from a other servers that send start request to the leader
func (pbft *PBFT) ReceiveForwardedCommand(command CommandArgs) {
	pbft.Start(command)
}


// Write the relavant state of PBFT to persistent storage
func (pbft *PBFT) persist() {
	
}

//! Restore previously persisted state.
func (pbft *PBFT) readPersist(data []byte) {
	
}

func (pbft *PBFT) makeArguments(specificArgument interface{}) CommandArgs {
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