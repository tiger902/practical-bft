package pbft

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


	go pbft.sendRPCs(pbft.makeArguments(prepareCommand), PREPARE)
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
	
		go bft.sendRPCs(pbft.makeArguments(commitArgs), COMMIT)
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

		go pbft.sendRPCs(pbft.makeArguments(newView), NEW_VIEW)
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

		go pbft.sendRPCs(pbft.makeArguments(prepareCommand), PREPARE)
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