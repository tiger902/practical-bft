package pbft

import (
	"time"
)

// this file has the common structures that are needed by both the client and servers
const (
	OK           = "OK"
	ErrNoKey     = "ErrNoKey"
	OpInProgress = "IndenticalOperationInProgress"
)

// the command types
// these should be int so that the comparison would be faster
const (
)

type Err string

//!struct used as reply a client from every server
type CommandReply struct {
	CurrentView int			//!< view of the system at the time of the reply
	RequestTimestamp Time	//!< timestamp at which the request was made
	ClientID	int				//!< ID of the client that sent the original request
	ServerID 	int 		//!< ID of the server that is sending the reply
	ResultData interface{}	//!< result from performing the operation
}

//!struct used by the client to issue a new command
type Command struct {
	CommandType string			//!< type of the command that neeeds to be done
	CommandData interface{}		//!< data for the operation that the client needs to the operation to be done
	Timestamp Time				//!< timestamp for the operation
	ClientID	int				//!< ID of the client
}
