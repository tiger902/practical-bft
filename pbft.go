package pbft

import (
	"crypto/ecdsa"
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
// pbft.GetState() (isLeader)
//   ask a PBFT for its current term, and whether it thinks it is leader
//

//! returns whether this server believes it is the leader.
func (pbft *PBFT) GetState() bool {

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()
	return (pbft.view%len(pbft.peers) == pbft.serverID)
}

// starts a new command
// return the
func (pbft *PBFT) Start(clientCommand Command) {

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()

	// if not primary, send the request to the primary
	leader := pbft.view % len(pbft.peers)
	if pbft.serverID != leader {
		commandArg := pbft.makeArguments(clientCommand)
		go pbft.sendRPCs(commandArg, FORWARD_COMMAND)
		return
	}

	// do nothing if the time is before what we have sent already, and update the time otherwise
	if lastReplyTimestamp, ok := pbft.clientRegisters[clientCommand.ClientID]; ok {
		if clientCommand.Timestamp.Before(lastReplyTimestamp) {
			return
		}
	}
	pbft.clientRegisters[clientCommand.ClientID] = clientCommand.Timestamp

	// make a digest for the command from the client
	hash, err := Hash(clientCommand, nil)
	if err != nil {
		panic(err)
	}

	prePrepareCommandArgsNoMessage := PreprepareWithNoClientMessage{
		View:           pbft.view,
		SequenceNumber: pbft.sequenceNumber,
		Digest:         hash,
	}

	signedPreprepareNoMessage := pbft.makeArguments(prePrepareCommandArgsNoMessage)

	prePrepareCommandArgs := PrePrepareCommandArg{
		PreprepareNoClientMessage: signedPreprepareNoMessage,
		Message:                   clientCommand,
	}

	// multicast to all the other servers
	go pbft.sendRPCs(pbft.makeArguments(prePrepareCommandArgs), PRE_PREPARE)

	// process the recived command accordingly
	pbft.addLogEntry(&prePrepareCommandArgs)
}

// stops the server
func (pbft *PBFT) Kill() {
	// Your code here, if desired.
}

//
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(privateKey ecdsa.PrivateKey, publicKeys []ecdsa.PublicKey, peers []*ClientEnd, serverID int,
	persister *Persister) *PBFT {

	pbft := &PBFT{}

	pbft.serverLock.Lock()
	pbft.privateKey = privateKey
	pbft.publicKeys = publicKeys
	pbft.peers = peers
	pbft.persister = persister
	pbft.serverID = 0
	pbft.sequenceNumber = 0
	pbft.commandsChannel = make(chan int, 10)
	pbft.uncommittedCommands = 0
	pbft.state = IDLE
	pbft.lastCheckPointSeqNumber = 0
	pbft.view = 0
	pbft.serverLog = make(map[int]logEntry)
	pbft.clientRegisters = make(map[int]time.Time)
	pbft.storedState = make(map[string]interface{})
	pbft.checkPoints = make(map[int]CheckPointInfo)
	pbft.viewChanges = make(map[int]map[int]CommandArgs)
	pbft.newValidCommad = make(chan bool, 10)
	pbft.commandExecuted = make(chan bool, 10)
	pbft.commandRecieved = make(chan bool, 10)
	pbft.viewChangeComplete = make(chan int, 10)

	pbft.readPersist(persister.ReadPBFTState())
	pbft.serverLock.Unlock()

	// start a go routine for handling command
	go pbft.runningState()

	return pbft
}
