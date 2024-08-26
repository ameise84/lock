//go:build debug

package lock

import (
	"io"
	"log"
	"os"
	"time"
)

func CheckDead(ok bool, where string) {
	_gLogOut = ok
	if ok {
		file, err := os.OpenFile("./deadlock-"+where+"-"+time.Now().Format("20060102150405.999")+".log", os.O_CREATE|os.O_WRONLY|os.O_SYNC|os.O_APPEND, 0666)
		if err != nil {
			_gLogOut = false
			return
		}
		_gLogger = log.New(io.MultiWriter(file, os.Stdout), "", 0)
	}
}
