package lock

import (
	"log"
	"os"
)

const (
	layout     = "2006-01-02 15:04:05.000"
	warnFormat = "\u001B[93;1m[%s WARN] %v\u001B[0m\n"
)

var (
	_gLogOut        bool
	_gDefaultLogger = log.New(os.Stdout, "", log.LstdFlags)
	_gLogger        = _gDefaultLogger
)
