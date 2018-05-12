package pbft

import "log"

//  HandlePrePrepareRPC receives and processes preprepare commands
//
// \param args 
func (pbft *PBFT) HandlePrePrepareRPC(args CommandArgs) {

	if pbft.isChangingView() {
		return
	}

	preprepareCommandArgs, ok := args.SpecificArguments.(PrePrepareCommandArg)

	if !ok {
		log.Fatal("[handlePrePrepareRPC] preprepare command args failed")
	}

	prePrepareNoClientMessageArgs := preprepareCommandArgs.PreprepareNoClientMessage

	prePrepareNoClientMessages, ok1 := prePrepareNoClientMessageArgs.SpecificArguments.(PreprepareWithNoClientMessage)
	if !ok1 {
		log.Fatal("[handlePrePrepareRPC] preprepare command args failed")
	}

	// return of the message digest does not match with the one that will be calculated here
	if !verifyDigests(preprepareCommandArgs.Message, prePrepareNoClientMessages.Digest) {
		log.Print("[handlePrePrepareRPC] preprepare command args failed")
		return
	}
	
	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()
	view := pbft.view
	serverCount := len(pbft.peers)
	leader := pbft.view % serverCount
	
	// check that the signature of the prepare command match
	signatureArg := verifySignatureArg {
		generic: preprepareCommandArgs,

	}
	if !pbft.verifySignatures(&signatureArg, &args.R_firstSig, &args.S_secondSig, leader) {
		return
	}

	signatureArg = verifySignatureArg {
		generic: prePrepareNoClientMessages,

	}
	if !pbft.verifySignatures(&signatureArg, &prePrepareNoClientMessageArgs.R_firstSig, &prePrepareNoClientMessageArgs.S_secondSig, leader) {
		return
	}


	// check the view
	if (view != prePrepareNoClientMessages.View) {
		return
	}

	// add entry to the log
	pbft.addLogEntry(&preprepareCommandArgs)

	// broadcast prepare messages to everyone if you are not the leader
	prepareCommand := PrepareCommandArg{
		View:           prePrepareNoClientMessages.View,
		SequenceNumber: prePrepareNoClientMessages.SequenceNumber,
		Digest:         prePrepareNoClientMessages.Digest,
		SenderIndex:    pbft.serverID,
	}

	pbft.newValidCommad <- true
	go pbft.sendRPCs(pbft.makeArguments(prepareCommand), PREPARE)
}


//!< Handle the RPC to prepare messages
func (pbft *PBFT) HandlePrepareRPC(args CommandArgs) {

	if pbft.isChangingView() {
		return
	}

	pbft.commandRecieved <- true
	prepareArgs, ok := args.SpecificArguments.(PrepareCommandArg)
	if !ok {
		log.Fatal("[handlePrePrepareRPC] preprepare command args failed")
	}

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()

	signatureArg := verifySignatureArg {
		generic: prepareArgs,

	}
	// check that the signature of the prepare command match
	if !pbft.verifySignatures(&signatureArg, &args.R_firstSig, &args.S_secondSig, prepareArgs.SenderIndex) {
		return
	}

	logEntryItem, ok1 := pbft.serverLog[prepareArgs.SequenceNumber]

	// A replica (including the primary) accepts prepare messages and adds them to its log
	// provided their signatures are correct, their view number equals the replica’s current view,
	// and their sequence number is between h and H.
	
	// do nothing if we did not receive a preprepare
	if !ok1 {
		return
	}

	// do not accept of different views, or different signatures
	if (prepareArgs.View != pbft.view) || (prepareArgs.Digest != logEntryItem.commandDigest) {
		return
	}

	// TODO: check for the sequenceNumber ranges for the watermarks


	pbft.serverLog[prepareArgs.SequenceNumber].prepareCount++

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
	f := (len(pbft.peers) - 1) / 3
	majority := 2*f + 1

	if (logEntryItem.prepareCount >= majority) {
		logEntryItem.prepared = true
		pbft.serverLog[prepareArgs.SequenceNumber].commitCount++
 	   commitArgs := CommitArg{
						View: pbft.view,
						SequenceNumber: prepareArgs.SequenceNumber,
						Digest: prepareArgs.Digest,
						SenderIndex: pbft.serverID,
					}
	
		go pbft.sendRPCs(pbft.makeArguments(commitArgs), COMMIT)
	}

}

// Handles the commit RPC 
func (pbft *PBFT) HandleCommitRPC(args CommandArgs) {

	if pbft.isChangingView() {
		return
	}

	pbft.commandRecieved <- true

	commitArgs, ok := args.SpecificArguments.(PrepareCommandArg)
	if !ok {
		log.Fatal("[handlePrePrepareRPC] preprepare command args failed")
	}
	
	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()


	// Replicas accept commit messages and insert them in their log provided 
	// they are properly signed, the view number in the message is equal to the replica’s 
	// current view, and the sequence number is between h and H

	// TODO: implement the h and H stuff
	// TODO: verify signatures of this thing

	if commitArgs.View != pbft.view {
		return
	}

	logEntryItem, ok := pbft.serverLog[commitArgs.SequenceNumber]
	
	// do nothing if we did not receive a preprepare
	if !ok {
		return
	}

	// do nothing if this entry has not been prepared
	// TODO: is this the behavior we want? 
	if !logEntryItem.prepared {
		return
	}

	signatureArg := verifySignatureArg {
		generic: commitArgs,

	}
	// check that the signature of the prepare command match
	if !pbft.verifySignatures(&signatureArg, &args.R_firstSig, &args.S_secondSig, commitArgs.SenderIndex) {
		return
	}

	// do nothing if the reply has been sent already
	if logEntryItem.clientReplySent {
		return
	}
	
	// go into the commit phase for this command after 2F + 1 replies
	f := (len(pbft.peers) - 1) / 3
	majority := 2*f + 1

	logEntryItem.commitCount++

	if ((logEntryItem.commitCount >= majority) && !logEntryItem.clientReplySent) {
		logEntryItem.clientReplySent = true
		clientCommand := logEntryItem.message.(Command)
 	   	clientCommandReply := CommandReply {
							CurrentView : pbft.view,
							RequestTimestamp: clientCommand.Timestamp, 
							ClientID: clientCommand.ClientID,
							ServerID: pbft.serverID,
						}
		
		// perform the checkpointing if this is the right moment
		if (commitArgs.SequenceNumber / CHECK_POINT_INTERVAL) == 0 {
			checkPointInfo := CheckPointInfo {
				LargestSequenceNumber: commitArgs.SequenceNumber,
				CheckPointState: pbft.storedState,
				ConfirmedServers: make(map[int]CommandArgs),
			}
			pbft.checkPoints[commitArgs.SequenceNumber] = checkPointInfo
			pbft.lastCheckPointSeqNumber = commitArgs.SequenceNumber
			go pbft.makeCheckpoint(checkPointInfo)
		}

		go pbft.replyToClient(clientCommandReply)

		pbft.commandExecuted <- true
	}
}


// Handles the view change RPC 
func (pbft *PBFT) HandleViewChangeRPC(args CommandArgs) {

	pbft.changeState(CHANGING_VIEW)

	viewChange, ok := args.SpecificArguments.(ViewChange)
	if !ok {
		log.Fatal("[handlePrePrepareRPC] preprepare command args failed")
	}

	// do nothing is this is not for this server
	if ((viewChange.NextView % len(pbft.peers)) != pbft.serverID) {
		return
	}

	signatureArg := verifySignatureArg {
		generic: viewChange,

	}
	// check that the signature of the prepare command match
	if !pbft.verifySignatures(&signatureArg, &args.R_firstSig, &args.S_secondSig, viewChange.SenderID) {
		return
	}

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()

	// return if this server's view is more than that of the sender
	// of this server is not supposed to be a leader
	if ((viewChange.NextView <= pbft.view) || ((viewChange.NextView % len(pbft.peers))  != pbft.view)) {
		return
	}

	// TODO: verify all the prepare messages that are in the viewChange argument to make
	// sure that they are all the same
	// LastCheckPointMessages and PreparedMessages needs to be verified

	// make a new map if the array of the view change messages does not exist yet
	viewChanges, ok := pbft.viewChanges[viewChange.NextView]
	if !ok {
		pbft.viewChanges[viewChange.NextView] = make(map[int]CommandArgs)
	} else {
		if _, ok1 := viewChanges[viewChange.SenderID]; ok1 {
			return
		}
	}
 
	pbft.viewChanges[viewChange.NextView][viewChange.SenderID] =  args

	// if we now have the majority of view change requests
	if (len(pbft.viewChanges[viewChange.NextView]) >= 2 * pbft.failCount) {

		pbft.viewChanges[viewChange.NextView][pbft.serverID] = pbft.makeViewChangeArguments()

		var allPreprepareMessage []CommandArgs
		pbft.createPreprepareMessages(viewChange.NextView, &allPreprepareMessage)

		// append to the log all the entries
		for _, preprepareMessage := range allPreprepareMessage {

			prePrepareCommandArg := PrePrepareCommandArg {
				PreprepareNoClientMessage: preprepareMessage,
			}
			pbft.addLogEntry(&prePrepareCommandArg)
		}

		pbft.view = viewChange.NextView

		// send the new view to everyone
		newView := NewView {
			NextView: viewChange.NextView,
			ViewChangeMessages: pbft.viewChanges[viewChange.NextView],
			PreprepareMessage: allPreprepareMessage,
		}

		go pbft.sendRPCs(pbft.makeArguments(newView), NEW_VIEW)
	}

}

//!< Function to handle the new view RPC from the leader
func (pbft *PBFT) HandleNewViewRPC(args CommandArgs) {


	newView, ok := args.SpecificArguments.(NewView)

	if !ok {
		log.Fatal("[handlePrePrepareRPC] preprepare command args failed")
	}

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()

	// do nothing if we are already at a higher view
	if (newView.NextView < pbft.view) {
		return
	}

	// find the new leader
	var newLeader int
	for serverID := 0; serverID < len(pbft.peers);  serverID++{
		if ((newView.NextView % len(pbft.peers)) == serverID) {
			newLeader = serverID
			break
		}
	}

	signatureArg := verifySignatureArg {
		generic: newView,

	}
	// check that the signature of the prepare command match
	if !pbft.verifySignatures(&signatureArg, &args.R_firstSig, &args.S_secondSig, newLeader) {
		return
	}

	// check the validity of view change messages
	for serverID, viewChangeArgs := range newView.ViewChangeMessages {
		viewChangeMessage := viewChangeArgs.SpecificArguments.(NewView)
		if viewChangeMessage.NextView != newView.NextView {
			return
		}

		signatureArg = verifySignatureArg {
			generic: viewChangeMessage,

		}

		if !pbft.verifySignatures(&signatureArg, &viewChangeArgs.R_firstSig, &viewChangeArgs.S_secondSig, serverID) {
			return
		}
	}

	// check the validity of prepare messages and compare to those that we
	// got from the new leader
	var allPreprepareMessage []CommandArgs
	pbft.createPreprepareMessages(newView.NextView, &allPreprepareMessage)

	// TODO: perform the verification but this is not needed since it's redundant
	
	// append to the log all the entries
	for _, preprepareMessage := range allPreprepareMessage {

		prePrepareCommandArg := PrePrepareCommandArg {
			PreprepareNoClientMessage: preprepareMessage,
		}
		pbft.addLogEntry(&prePrepareCommandArg)

		preprepareNoClientMessage := preprepareMessage.SpecificArguments.(PreprepareWithNoClientMessage)

		// broadcast prepare messages to everyone if you are not the leader
			prepareCommand := PrepareCommandArg { 
				View: preprepareNoClientMessage.View, 
				SequenceNumber: preprepareNoClientMessage.SequenceNumber,
				Digest: preprepareNoClientMessage.Digest,
				SenderIndex: pbft.serverID,
			}

		go pbft.sendRPCs(pbft.makeArguments(prepareCommand), PREPARE)
	}
	pbft.view = newView.NextView
	pbft.viewChangeComplete <- len(allPreprepareMessage)
}


func (pbft *PBFT) HandleCheckPointRPC(args CommandArgs) {

	if pbft.isChangingView() {
		return
	}

	checkPointArgs, ok := args.SpecificArguments.(CheckPointArgs)
	if !ok {
		log.Fatal("[handlePrePrepareRPC] preprepare command args failed")
	}

	signatureArg := verifySignatureArg {
		generic: checkPointArgs,

	}
	// check that the signature of the prepare command match
	if !pbft.verifySignatures(&signatureArg, &args.R_firstSig, &args.S_secondSig, checkPointArgs.SenderIndex) {
		return
	}

	// return if you do not have this checkpoint started, otherwise check the digests
	checkPointInfo, ok := pbft.checkPoints[checkPointArgs.SequenceNumber];
	if !ok {
		return 
	}
	if !verifyDigests(checkPointInfo, checkPointArgs.Digest) {
		return
	}

	pbft.serverLock.Lock()
	pbft.checkPoints[checkPointArgs.SequenceNumber].ConfirmedServers[checkPointArgs.SenderIndex] = args

	// for all checkpoints, check to see if we have received a majority
	currentCheckpointInfo := pbft.checkPoints[checkPointArgs.SequenceNumber]

	// if this checkpoint is now stable, delete all those that are before this one
	if !currentCheckpointInfo.IsStable {
		if (len(currentCheckpointInfo.ConfirmedServers) >= (2 * pbft.failCount)) {
			currentCheckpointInfo.IsStable = true
			pbft.checkPoints[checkPointArgs.SequenceNumber] = currentCheckpointInfo
			pbft.removeStallCheckpoints(checkPointArgs.SequenceNumber)
		}
	}
	pbft.serverLock.Unlock()
}