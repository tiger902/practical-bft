package main

//!< Consts with the phases of the protocol
const (
	FORWARD_COMMAND = 0
	PRE_PREPARE     = 1
	PREPARE         = 2
	COMMIT          = 3
	REPLY_TO_CLIENT = 4
	VIEW_CHANGE     = 5
	NEW_VIEW        = 6
	CHECK_POINT     = 7
)

//General constants
const (
	numServers = 4
)

const (
	IDLE               = 0
	PROCESSING_COMMAND = 1
	CHANGING_VIEW      = 2
)

// protocol constants
const (
	CHECK_POINT_INTERVAL = 100
	VIEW_CHANGE_INTERVAL = 150 // time for initiating view change
)
