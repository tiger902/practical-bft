package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"
	"encoding/gob"
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
func (pbft *PBFT) GetState(args interface{}, reply *GetStateReply) error {

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()
	reply.isLeader = (pbft.view%len(pbft.peers) == pbft.serverID)

	return nil
}

// starts a new command
// return the
func (pbft *PBFT) Start(clientCommand Command, reply *int) error {
	log.Printf("Start being called\n")
	//pbft.replyToClient(CommandReply{}, "18.206.100.184")

	pbft.serverLock.Lock()
	defer pbft.serverLock.Unlock()

	// if not primary, send the request to the primary
	leader := pbft.view % len(pbft.peers)
	if pbft.serverID != leader {
		commandArg := pbft.makeArguments(clientCommand)
		go pbft.sendRPCs(commandArg, FORWARD_COMMAND)
		return nil
	}

	// do nothing if the time is before what we have sent already, and update the time otherwise
	if lastReplyTimestamp, ok := pbft.clientRegisters[clientCommand.ClientID]; ok {
		if clientCommand.Timestamp.Before(lastReplyTimestamp) {
			return nil
		}
	}

	// make a digest for the command from the client
	hash, err := Hash(clientCommand, nil)
	log.Print(hash)
	if err != nil {
		panic(err)
	}

	prePrepareCommandArgsNoMessage := PreprepareWithNoClientMessage{
		View:           pbft.view,
		SequenceNumber: pbft.sequenceNumber,
		Digest:         hash,
		Timestamp:                  time.Now(),
	}

	signedPreprepareNoMessage := pbft.makeArguments(prePrepareCommandArgsNoMessage)

	prePrepareCommandArgs := PrePrepareCommandArg{
		PreprepareNoClientMessage: signedPreprepareNoMessage,
		Message:                   clientCommand,
		Timestamp:                  time.Now(),
	}

	// multicast to all the other servers
	go pbft.sendRPCs(pbft.makeArguments(prePrepareCommandArgs), PRE_PREPARE)

	// process the recived command accordingly
	pbft.addLogEntry(&prePrepareCommandArgs)

	return nil
}

// stops the server
/*func (pbft *PBFT) Kill() {
	// Your code here, if desired.
}*/

//
// Make() must return quickly, so it should start goroutines
// for any long-running work.
// Arguments: privateKey ecdsa.PrivateKey, publicKeys []ecdsa.PublicKey, peers []*ClientEnd, serverID int
//
func (pbft *PBFT) Make(args *MakeArgs, reply *int) error {
	log.Print("Make being called!\n")

	//persister := &Persister{mu: sync.Mutex{}}

	peers := []string{}

	for i := 0; i < len(args.IpAddrs); i++ {
		peers = append(peers, args.IpAddrs[i])
	}

	pbft.serverLock.Lock()
	/*pbft.privateKey = nil
	pbft.publicKeys = nil*/
	pbft.peers = peers
	//pbft.persister = persister
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

	// TODO: uncomment this
	//pbft.readPersist(persister.ReadPBFTState())
	pbft.serverLock.Unlock()

	// start a go routine for handling command
	// TODO: remove this later when we fix everything
	//go pbft.runningState()

	return nil
}

func main() {

	runType := os.Args[1]
	log.Print("Entering main function\n")

	gob.Register(RPCReply{})
	gob.Register(CommandReply{})
	gob.Register(Command{})
	gob.Register(GetStateReply{})

	gob.Register(MakeArgs{})
	gob.Register(PreprepareWithNoClientMessage{})
	gob.Register(PrePrepareCommandArg{})
	gob.Register(PrepareCommandArg{})
	gob.Register(CommitArg{})
	gob.Register(CheckPointArgs{})
	gob.Register(PrepareMForViewChange{})
	gob.Register(ViewChange{})
	gob.Register(NewView{})
	gob.Register(logEntry{})
	gob.Register(CheckPointInfo{})
	gob.Register(CommandArgs{})

	//fmt.Println(runType)
	//fmt.Println(os.Args[5])

	if runType == "server" {
		log.Print("Entering server\n")
		pbft := &PBFT{}
		rpc.Register(pbft)
		log.Print("Registering server\n")

		rpc.HandleHTTP()
		log.Print("Handle HTTP\n")

		l, e := net.Listen("tcp", ":1234")
		if e != nil {
			log.Fatal("listen error:", e)
		}

		log.Print("About to serve\n")
		http.Serve(l, nil)
		log.Print("Served them!\n")
	} else if runType == "raft"  
		startRaft()
	}else {
		log.Print("Entering client\n")
		Bootstrap()
	}


}
