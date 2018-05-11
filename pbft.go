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

// starts a new command
// return the 
func (pbft *PBFT) Start(clientCommand Command) {

	// TODO: verify that the messages have not been changed on their way from the client


	// if not primary, send the request to the primary
	pbft.serverLock.Lock()
	leader := pbft.view % pbft.numberOfServer

	if (pbft.serverID != leader) {
		pbft.serverLock.Unlock()
		pbft.sendRPCs(command, FORWARD_COMMAND)
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

