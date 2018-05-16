package main

import (
	"encoding/gob"
	"log"
	"os"
)

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

	if runType == "server" {
		runpbft()
	} else if runType == "raft" {
		runRaft()
	} else {
		log.Print("Entering client\n")
		runClient()
	}

}
