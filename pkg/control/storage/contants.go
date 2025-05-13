package storage

import (
	"lightiot/pkg/control/runtime"
)

const (
	cmdRaw     = "cmd-raw"
	actionRaw  = "act-raw"
	timeKey    = "_time"

	cacheSize = 128
)

var fixedCmdSelects = []string {
	runtime.CommandFieldSeq,
	runtime.CommandFieldState,
	runtime.CommandFieldCode,
	runtime.CommandFieldMsg,
	runtime.CommandFieldThingId,
}

var fixedActionSelects = []string {
	runtime.ActionFieldSeq,
	runtime.ActionFieldState,
	runtime.ActionFieldCode,
	runtime.ActionFieldMsg,
}