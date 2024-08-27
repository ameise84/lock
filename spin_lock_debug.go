//go:build debug

package lock

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

type stack struct {
	file     string
	funcName string
	line     int
	gid      int
	skip     int
}

type SpinLock struct {
	isLocked atomic.Int64
	infos    []stack
}

func (l *SpinLock) LockSkip(n int) {
	const maxBlock = 64
	block := 1
	t0 := time.Now()
	t1 := t0
	for {
		if l.tryLock(n) {
			break
		}
		for i := 0; i < block; i++ {
			runtime.Gosched()
		}
		if block < maxBlock {
			block <<= 1
		}
		if _gLogOut {
			t2 := time.Now()
			if t2.Sub(t1).Seconds() > 5 {
				t1 = t2
				gid := getGID()
				funcName, file, line, _ := runtime.Caller(n + 1)
				b := strings.Builder{}
				for n, info := range l.infos {
					b.WriteString(fmt.Sprintf("\t-> [%d][gid:%d] func: %s\t%s:%d\n", n, info.gid, info.funcName, info.file, info.line))
				}
				msg := fmt.Sprintf("dead lock[%.2f s]\nlock on:\n%s\ncalling on:\n\t[gid:%d] func: %s\t%s:%d\n", t2.Sub(t0).Seconds(), b.String(), gid, runtime.FuncForPC(funcName).Name(), file, line)
				_gLogger.Printf(warnFormat, time.Now().Format(layout), msg)
			}
		}
	}
}

func (l *SpinLock) Lock() {
	l.LockSkip(1)
}

func (l *SpinLock) TryLock() bool {
	return l.tryLock(0)
}

func (l *SpinLock) TryLockInTime(dur time.Duration) bool {
	const maxBlock = 64
	block := 1
	tr := time.NewTimer(dur)
	isLocked := false
loopLock:
	for {
		if l.tryLock(0) {
			isLocked = true
			break loopLock
		}

		select {
		case <-tr.C:
			break loopLock
		default:
		}
		for i := 0; i < block; i++ {
			runtime.Gosched()
		}
		if block < maxBlock {
			block <<= 1
		}
	}

	tr.Stop()
	select {
	case <-tr.C:
	default:
	}
	return isLocked
}

func (l *SpinLock) tryLock(skip int) bool {
	ok := l.isLocked.CompareAndSwap(unlocked, locked)
	if ok {
		l.infos = l.infos[:0]
		l.recordLockIndex(getGID(), skip)
	}
	return ok
}

func (l *SpinLock) Unlock() {
	l.isLocked.Store(unlocked)
}

func (l *SpinLock) recordLockIndex(gid int, skip int) {
	funcName, file, line, _ := runtime.Caller(3 + skip)
	s := stack{file: file, funcName: runtime.FuncForPC(funcName).Name(), line: line, gid: gid, skip: skip}
	l.infos = append(l.infos, s)
}
