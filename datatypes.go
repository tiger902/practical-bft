package main

import (
	"crypto/ecdsa"
	"sync"
	"time"
)

//! A Go object implementing a single PBFT peer.
type PBFT struct {
	privateKey ecdsa.PrivateKey  //!< Private key for this server
	publicKeys []ecdsa.PublicKey //!< Array of publick keys for all servers
	peers      []string     //!< Array of all the other server sockets for RPC
	//persister               *Persister        //!< Persister to be used to store data for this server in permanent storage
	serverID                int      //!< Index into peers[] for this server
	sequenceNumber          int      //!< last sequence number
	commandsChannel         chan int //!< channel for commands
	uncommittedCommands     int      //!< store the number of commands that we are waiting for
	state                   int      //!< the state of the server
	serverStateLock         sync.Mutex
	lastCheckPointSeqNumber int                    //!< the sequence number of the last checkpoint
	view                    int                    //!< the current view of the system
	serverLog               map[int]logEntry       //!< to keep track of all the commands that have been seen regard less of the stage
	serverLock              sync.Mutex             //!< Lock to be when changing the sequenceNumber,
	clientRegisters         map[int]time.Time      // to keep track of the last reply that has been sent to this client
	storedState             map[string]interface{} // this is the state that the application is keeping for this application
	checkPoints             map[int]CheckPointInfo
	viewChanges             map[int]map[int]CommandArgs // only used when we are performing a view change. The first index is
	// is for the new view, and the second is for the serverI
	// is for the new view, and the second is for the serverID
	newValidCommad     chan bool
	commandExecuted    chan bool
	commandRecieved    chan bool
	viewChangeComplete chan int
}

//Dummy reply
type RPCReply struct{}

type MakeArgs struct {
	IpAddrs    [numServers]string
	ServerID   int
}

//!struct used as argument to multicast command
type CommandArgs struct {
	SpecificArguments interface{}
	R_firstSig        int
	S_secondSig       int
}

type PreprepareWithNoClientMessage struct {
	View           int    //!< leader’s term
	SequenceNumber int    //!< Sequence number of the messsage
	Digest         uint64 //!< Digest of the message, which can is an uint64 TODO: change type when we switch to SHA256
}

type verifySignatureArg struct {
	generic interface{}
}

//!struct used for preprepare command arguments before they have been signed
// TODO: change this to reflect that in places we use it
type PrePrepareCommandArg struct {
	PreprepareNoClientMessage CommandArgs
	Message                   interface{} //!< Message for the command TODO: decouple this
}

//!struct used as argument to multicast command
type PrepareCommandArg struct {
	View           int    //!< leader’s term
	SequenceNumber int    //!< Sequence number of the messsage
	Digest         uint64 //!< Digest of the message, which can is an int TODO: check the type of this
	SenderIndex    int    //!< Id of the server that sends the prepare message
}

//!struct used as argument to multicast command
type CommitArg struct {
	View           int    //!< leader’s term
	SequenceNumber int    //!< Sequence number of the messsage
	Digest         uint64 //!< Digest of the message, which can is an int TODO: check the type of this
	SenderIndex    int    //!< Id of the server that sends the prepare message
}

//!struct used as argument to multicast command
type CheckPointArgs struct {
	SequenceNumber int    //!< the last sequence number that is reflected in the checkpoint
	Digest         uint64 //!< digest of the state of the server that is being checkpoint-ed
	SenderIndex    int    //!< ID of the server
}

//! struct that is used to mak
type PrepareMForViewChange struct {
	PreprepareMessage CommandArgs         //!< valid preprepare message (without the corresponding client message)
	PrepareMessages   map[int]CommandArgs //!< 2f matching, valid prepare messages signed by different backups
	//!< with the same view, sequence number, and the digest of client message
}

//! struct for the view change arguments
type ViewChange struct {
	NextView                     int
	LastCheckPointSequenceNumber int
	LastCheckPointMessages       map[int]CommandArgs
	PreparedMessages             map[int]PrepareMForViewChange
	SenderID                     int
}

//! struct for the view change arguments
type NewView struct {
	NextView           int                 //!< The next view of the system
	ViewChangeMessages map[int]CommandArgs //!< Map with all the valid view change messages from 2f + this server
	PreprepareMessage  []CommandArgs
}

//!struct used to store the log entry
type logEntry struct {
	message         interface{}
	preprepare      PrePrepareCommandArg
	prepareArgs     map[int]CommandArgs //!< to keep track of the prepare messages
	commitArgs      map[int]CommandArgs //!< to keep track of the commit messages
	commandDigest   uint64
	view            int
	clientReplySent bool
}

//! struct that keeps the checkpoints of the server
type CheckPointInfo struct {
	LargestSequenceNumber int                 //!< The largest sequence number for the checkpoint
	CheckPointState       interface{}         //!< The state of the server that is part of the checkpoint
	ConfirmedServers      map[int]CommandArgs //!< The servers that have accepted and verified the checkpoint
	IsStable              bool
}
