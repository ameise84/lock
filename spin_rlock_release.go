//go:build !debug

package lock

import (
	"runtime"
	"sync/atomic"
	"time"
)

type ReinLock struct {
	mu    SpinLock
	gid   atomic.Value
	count uint64
}

func (l *ReinLock) Lock() {
	gid := getGID()
	const maxBlock = 64
	block := 1
	if l.tryReinLock(gid) {
		return
	}
	for {
		if l.trySpinLock(gid) {
			break
		}

		for i := 0; i < block; i++ {
			runtime.Gosched()
		}
		if block < maxBlock {
			block <<= 1
		}
	}
}

func (l *ReinLock) TryLock() bool {
	gid := getGID()
	return l.tryReinLock(gid)
}

func (l *ReinLock) TryLockInTime(dur time.Duration) bool {
	gid := getGID()
	const maxBlock = 64
	block := 1
	tr := time.NewTimer(dur)
	isLocked := false
loopLock:
	for {
		if l.tryReinLock(gid) {
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
	return isLocked
}

func (l *ReinLock) tryReinLock(gid int) bool {
	if l.gid.Load() == gid {
		l.count += 1
		return true
	}

	return l.trySpinLock(gid)
}

func (l *ReinLock) trySpinLock(gid int) bool {
	if l.mu.TryLock() {
		l.gid.Store(gid)
		l.count = 1
		return true
	}
	return false
}

func (l *ReinLock) Unlock() {
	l.count -= 1
	if l.count == 0 {
		l.gid.Store(-1)
		l.mu.Unlock()
	}
}
