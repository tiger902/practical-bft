package main

import "log"

//  HandlePrePrepareRPC receives and processes preprepare commands
func (pbft *PBFT) HandlePrePrepareRPC(args CommandArgs, reply *RPCReply) error {

	log.Print("Handle preprepare caled\n")
	if pbft.isChangingView() {
		log.Print("[HandlePrePrepareRPC] pbft.isChangingView() returned true")
		return nil
	}

	//pbft.commandRecieved <- true

	preprepareCommandArgs, ok := args.SpecificArguments.(PrePrepareCommandArg)
	if !ok {
		log.Fatal("[handlePrePrepareRPC] args.SpecificArguments.(PrePrepareCommandArg) failed")
	}

	prePrepareNoClientMessageArgs := preprepareCommandArgs.PreprepareNoClientMessage
	prePrepareNoClientMessages, ok1 := prePrepareNoClientMessageArgs.SpecificArguments.(PreprepareWithNoClientMessage)
	if !ok1 {
		log.Fatal("[handlePrePrepareRPC]  prePrepareNoClientMessageArgs.SpecificArguments.(PreprepareWithNoClientMessage) failed")
	}

	// return of the message digest does not match with the one that will be calculated here
	if !verifyDigests(preprepareCommandArgs.Message, prePrepareNoClientMessages.Digest) {
		log.Print("[handlePrePrepareRPC] verifyDigests(preprepareCommandArgs.Message, prePrepareNoClientMessages.Digest) returned false")
		return nil
	}

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()

	// check the view
	if pbft.view != prePrepareNoClientMessages.View {
		log.Print("[handlePrePrepareRPC] make sure our idea of the current view is the same")
		return nil
	}

	// verify the signature of the received commands
	signatureArg := verifySignatureArg{
		generic: preprepareCommandArgs,
	}
	if !pbft.verifySignatures(&signatureArg, args.R_firstSig, args.S_secondSig, pbft.view%len(pbft.peers)) {
		log.Print("[handlePrePrepareRPC] verify signatures failed")
		return nil
	}

	signatureArg = verifySignatureArg{
		generic: prePrepareNoClientMessages,
	}
	if !pbft.verifySignatures(&signatureArg, prePrepareNoClientMessageArgs.R_firstSig, prePrepareNoClientMessageArgs.S_secondSig, pbft.view%len(pbft.peers)) {
		log.Print("[handlePrePrepareRPC] verify signatures on prepare no client messages failed")
		return nil
	}

	// add entry to the log
	if !pbft.addLogEntry(&preprepareCommandArgs) {
		return nil
	}

	// broadcast prepare messages to everyone if you are not the leader
	prepareCommand := PrepareCommandArg{
		View:           prePrepareNoClientMessages.View,
		SequenceNumber: prePrepareNoClientMessages.SequenceNumber,
		Digest:         prePrepareNoClientMessages.Digest,
		SenderIndex:    pbft.serverID,
	}

	//pbft.newValidCommad <- true

	prepareCommandArg := pbft.makeArguments(prepareCommand)
	go pbft.sendRPCs(prepareCommandArg, PREPARE)
	pbft.serverLog[prepareCommand.SequenceNumber].prepareArgs[pbft.serverID] = prepareCommandArg
	// TODO: maybe remove this save to persist
	//pbft.persist()
	return nil
}

//!< Handle the RPC to prepare messages
func (pbft *PBFT) HandlePrepareRPC(args CommandArgs, reply *RPCReply) error {

	/*if pbft.isChangingView() {
		return nil
	}*/

	//pbft.commandRecieved <- true
	prepareArgs, ok := args.SpecificArguments.(PrepareCommandArg)
	if !ok {
		log.Fatal("[handlePrePrepareRPC] preprepare command args failed")
	}

	// verify the signatures
	signatureArg := verifySignatureArg{
		generic: prepareArgs,
	}
	if !pbft.verifySignatures(&signatureArg, args.R_firstSig, args.S_secondSig, prepareArgs.SenderIndex) {
		log.Print("[HandlePrepareRPC] signatures of the prepare command args don't match")
		return nil
	}

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()

	logEntryItem, ok1 := pbft.serverLog[prepareArgs.SequenceNumber]

	// A replica (including the primary) accepts prepare messages and adds them to its log
	// provided their signatures are correct, their view number equals the replica’s current view,
	// and their sequence number is between h and H.

	// do nothing if we did not receive a preprepare
	if !ok1 {
		log.Print("[HandlePrepareRPC] did not recieve a preprepare")
		return nil
	}

	// do not accept of different views, or different signatures
	if (prepareArgs.View != pbft.view) || (prepareArgs.Digest != logEntryItem.commandDigest) {
		log.Print("[HandlePrepareRPC] saw different view or different digest")
		return nil
	}

	// TODO: check for the sequenceNumber ranges for the watermarks

	/*if _, ok2 := logEntryItem.prepareArgs[prepareArgs.SenderIndex]; ok2 {
		log.Print("[HandlePrepareRPC] already received from this server")
		return nil
	}*/

	pbft.serverLog[prepareArgs.SequenceNumber].prepareArgs[prepareArgs.SenderIndex] = args

	// return if already prepared
	/*if len(logEntryItem.prepareArgs) > pbft.calculateMajority() {
		// TODO: maybe remove this save to persist
		//pbft.persist()
		log.Printf("[HandlePrepareRPC] already prepared, so exiting")
		return nil
	}*/

	// We define the predicate prepared to be true iff replica has inserted in its
	// log: the request, a pre-prepare for in view with sequence number, and 2f
	// prepares from different backups that match the pre-prepare. The replicas verify
	// whether the prepares match the pre-prepare by checking that they have the
	// same view, sequence number, and digest

	// go into the commit phase for this command after 2F + 1 replies
	if len(logEntryItem.prepareArgs) == pbft.calculateMajority() {
		commitArgs := CommitArg{
			View:           pbft.view,
			SequenceNumber: prepareArgs.SequenceNumber,
			Digest:         prepareArgs.Digest,
			SenderIndex:    pbft.serverID,
		}

		commitArg := pbft.makeArguments(commitArgs)
		pbft.serverLog[prepareArgs.SequenceNumber].commitArgs[pbft.serverID] = commitArg

		go pbft.sendRPCs(commitArg, COMMIT)
	}
	// TODO: maybe remove this save to persist
	//pbft.persist()

	return nil
}

// Handles the commit RPC
func (pbft *PBFT) HandleCommitRPC(args CommandArgs, reply *RPCReply) error {

	/*if pbft.isChangingView() {
		log.Print("[HandleCommitRPC] currently changing views")
		return nil
	}*/

	//pbft.commandRecieved <- true

	commitArgs, _ := args.SpecificArguments.(PrepareCommandArg)
	/*if !ok {
		log.Fatal("[handlePrePrepareRPC] preprepare command args failed")
	}*/

	// verify signatures
	/*signatureArg := verifySignatureArg{
		generic: commitArgs,
	}
	// check that the signature of the prepare command match
	/*if !pbft.verifySignatures(&signatureArg, args.R_firstSig, args.S_secondSig, commitArgs.SenderIndex) {
		log.Print("[HandleCommitRPC] signature of the preprepare does not match")
		return nil
	}*/

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()

	// Replicas accept commit messages and insert them in their log provided
	// they are properly signed, the view number in the message is equal to the replica’s
	// current view, and the sequence number is between h and H

	// TODO: implement the h and H stuff
	// TODO: verify signatures of this thing

	/*if commitArgs.View != pbft.view {
		log.Print("[HandleCommitRPC] Current view does not match request")
		return nil
	}*/

	logEntryItem, ok := pbft.serverLog[commitArgs.SequenceNumber]

	// do nothing if we did not receive a preprepare
	/*if !ok {
		log.Print("[HandleCommitRPC] did not recieve a preprepare")
		return nil
	}*/

	// do nothing if this entry has not been prepared
	/*if len(logEntryItem.prepareArgs) < pbft.calculateMajority() {
		log.Printf("[HandleCommitRPC] not prepared, so exiting")
		return nil
	}*/

	// do nothing if we already received a commit from this server
	if _, ok2 := logEntryItem.commitArgs[commitArgs.SenderIndex]; ok2 {
		log.Print("[HandlePrepareRPC] already received from this server")
		return nil
	}
	pbft.serverLog[commitArgs.SequenceNumber].commitArgs[commitArgs.SenderIndex] = args

	// do nothing if the reply has been sent already
	/*if logEntryItem.clientReplySent {
		// TODO: maybe remove this save to persist
		//pbft.persist()

		log.Print("[HandleCommitRPC] reply to client has already been sent")
		return nil
	}*/

	// go into the commit phase for this command after 2F + 1 replies +

	if len(logEntryItem.commitArgs) == pbft.calculateMajority() {
		logEntryItem.clientReplySent = true
		clientCommand := logEntryItem.message.(Command)
		clientCommandReply := CommandReply{
			CurrentView:      pbft.view,
			RequestTimestamp: clientCommand.Timestamp,
			ClientID:         clientCommand.ClientID,
			ServerID:         pbft.serverID,
		}

		// perform the checkpointing if this is the right moment
		/*if (commitArgs.SequenceNumber / CHECK_POINT_INTERVAL) == 0 {
			checkPointInfo := CheckPointInfo{
				LargestSequenceNumber: commitArgs.SequenceNumber,
				CheckPointState:       pbft.storedState,
				ConfirmedServers:      make(map[int]CommandArgs),
			}
			pbft.checkPoints[commitArgs.SequenceNumber] = checkPointInfo
			pbft.lastCheckPointSeqNumber = commitArgs.SequenceNumber

			go pbft.makeCheckpoint(checkPointInfo)
		} */

		lastReplyTimestamp := pbft.clientRegisters[clientCommand.ClientID]
		if clientCommand.Timestamp.After(lastReplyTimestamp) {
			pbft.clientRegisters[clientCommand.ClientID] = clientCommand.Timestamp
		}

		go pbft.replyToClient(clientCommandReply, clientCommand.ClientAddress)

		//pbft.commandExecuted <- true
	}

	// TODO: maybe remove this for to performance
	//pbft.persist()

	return nil
}

// Handles the view change RPC
func (pbft *PBFT) HandleViewChangeRPC(args CommandArgs, reply *RPCReply) error {

	pbft.changeState(CHANGING_VIEW)

	viewChange, ok := args.SpecificArguments.(ViewChange)
	if !ok {
		log.Fatal("[handlePrePrepareRPC] preprepare command args failed")
	}

	// do nothing is this is not for this server
	if (viewChange.NextView % len(pbft.peers)) != pbft.serverID {
		log.Print("[HandleViewChangeRPC] not intended for this server")
		return nil
	}

	signatureArg := verifySignatureArg{
		generic: viewChange,
	}
	// check that the signature of the prepare command match
	if !pbft.verifySignatures(&signatureArg, args.R_firstSig, args.S_secondSig, viewChange.SenderID) {
		log.Print("[HandleViewChangeRPC] signatures of the preprepare command don't match")
		return nil
	}

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()

	// return if this server's view is more than that of the sender
	// of this server is not supposed to be a leader
	if (viewChange.NextView <= pbft.view) || ((viewChange.NextView % len(pbft.peers)) != pbft.view) {
		log.Print("[HandleViewChangeRPC] server's view is more than the sender")
		return nil
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
			log.Print("[HandleViewChangeRPC] view change message does not exist yet")
			return nil
		}
	}

	pbft.viewChanges[viewChange.NextView][viewChange.SenderID] = args

	// if we now have the majority of view change requests
	if len(pbft.viewChanges[viewChange.NextView]) >= 2*pbft.numFailableServers() {

		pbft.viewChanges[viewChange.NextView][pbft.serverID] = pbft.makeViewChangeArguments()

		var allPreprepareMessage []CommandArgs
		pbft.createPreprepareMessages(viewChange.NextView, &allPreprepareMessage)

		// append to the log all the entries
		for _, preprepareMessage := range allPreprepareMessage {

			prePrepareCommandArg := PrePrepareCommandArg{
				PreprepareNoClientMessage: preprepareMessage,
			}
			pbft.addLogEntry(&prePrepareCommandArg)
		}

		pbft.view = viewChange.NextView

		// send the new view to everyone
		newView := NewView{
			NextView:           viewChange.NextView,
			ViewChangeMessages: pbft.viewChanges[viewChange.NextView],
			PreprepareMessage:  allPreprepareMessage,
		}

		go pbft.sendRPCs(pbft.makeArguments(newView), NEW_VIEW)
	}
	return nil
}

//!< Function to handle the new view RPC from the leader
func (pbft *PBFT) HandleNewViewRPC(args CommandArgs, reply *RPCReply) error {

	newView, ok := args.SpecificArguments.(NewView)

	if !ok {
		log.Fatal("[HandleNewViewRPC] preprepare command args failed")
	}

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()

	// do nothing if we are already at a higher view
	if newView.NextView < pbft.view {
		log.Print("[HandleNewViewRPC] already have a higher view")
		return nil
	}

	// find the new leader
	var newLeader int
	for serverID := 0; serverID < len(pbft.peers); serverID++ {
		if (newView.NextView % len(pbft.peers)) == serverID {
			newLeader = serverID
			break
		}
	}

	signatureArg := verifySignatureArg{
		generic: newView,
	}
	// check that the signature of the prepare command match
	if !pbft.verifySignatures(&signatureArg, args.R_firstSig, args.S_secondSig, newLeader) {
		log.Print("[HandleNewViewRPC] signatures of preprepare command messages don't match")
		return nil
	}

	// check the validity of view change messages
	for serverID, viewChangeArgs := range newView.ViewChangeMessages {
		viewChangeMessage := viewChangeArgs.SpecificArguments.(NewView)
		if viewChangeMessage.NextView != newView.NextView {
			log.Print("[HandleNewViewRPC] view change messages not valid")
			return nil
		}

		signatureArg = verifySignatureArg{
			generic: viewChangeMessage,
		}

		if !pbft.verifySignatures(&signatureArg, viewChangeArgs.R_firstSig, viewChangeArgs.S_secondSig, serverID) {
			log.Print("[HandleNewViewRPC] signature not valid")
			return nil
		}
	}

	// check the validity of prepare messages and compare to those that we
	// got from the new leader
	var allPreprepareMessage []CommandArgs
	pbft.createPreprepareMessages(newView.NextView, &allPreprepareMessage)

	// TODO: perform the verification but this is not needed since it's redundant

	// append to the log all the entries
	for _, preprepareMessage := range allPreprepareMessage {

		prePrepareCommandArg := PrePrepareCommandArg{
			PreprepareNoClientMessage: preprepareMessage,
		}
		pbft.addLogEntry(&prePrepareCommandArg)

		preprepareNoClientMessage := preprepareMessage.SpecificArguments.(PreprepareWithNoClientMessage)

		// broadcast prepare messages to everyone if you are not the leader
		prepareCommand := PrepareCommandArg{
			View:           preprepareNoClientMessage.View,
			SequenceNumber: preprepareNoClientMessage.SequenceNumber,
			Digest:         preprepareNoClientMessage.Digest,
			SenderIndex:    pbft.serverID,
		}

		go pbft.sendRPCs(pbft.makeArguments(prepareCommand), PREPARE)
	}
	pbft.view = newView.NextView
	//pbft.viewChangeComplete <- len(allPreprepareMessage)

	return nil
}

func (pbft *PBFT) HandleCheckPointRPC(args CommandArgs, reply *RPCReply) error {

	if pbft.isChangingView() {
		log.Print("[HandleCheckPointRPC] currently changing view")
		return nil
	}

	checkPointArgs, ok := args.SpecificArguments.(CheckPointArgs)
	if !ok {
		log.Fatal("[HandleCheckPointRPC] preprepare command args failed")
	}

	signatureArg := verifySignatureArg{
		generic: checkPointArgs,
	}
	// check that the signature of the prepare command match
	if !pbft.verifySignatures(&signatureArg, args.R_firstSig, args.S_secondSig, checkPointArgs.SenderIndex) {
		log.Fatal("[HandleCheckPointRPC] signature not valid")
		return nil
	}

	// return if you do not have this checkpoint started, otherwise check the digests
	checkPointInfo, ok := pbft.checkPoints[checkPointArgs.SequenceNumber]
	if !ok {
		log.Fatal("[HandleCheckPointRPC] checkpoint not started")
		return nil
	}
	if !verifyDigests(checkPointInfo, checkPointArgs.Digest) {
		log.Fatal("[HandleCheckPointRPC] checkpoint digest invalid")
		return nil
	}

	pbft.serverLock.Lock()
	pbft.checkPoints[checkPointArgs.SequenceNumber].ConfirmedServers[checkPointArgs.SenderIndex] = args

	// for all checkpoints, check to see if we have received a majority
	currentCheckpointInfo := pbft.checkPoints[checkPointArgs.SequenceNumber]

	// if this checkpoint is now stable, delete all those that are before this one
	if !currentCheckpointInfo.IsStable {
		if len(currentCheckpointInfo.ConfirmedServers) >= (2 * pbft.numFailableServers()) {
			currentCheckpointInfo.IsStable = true
			pbft.checkPoints[checkPointArgs.SequenceNumber] = currentCheckpointInfo
			pbft.removeStallCheckpoints(checkPointArgs.SequenceNumber)
		}
	}
	pbft.serverLock.Unlock()

	return nil
}
