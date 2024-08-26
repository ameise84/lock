//go:build !debug

package lock

import (
	"runtime"
	"sync/atomic"
	"time"
)

type SpinLock struct {
	isLocked atomic.Int64
}

func (l *SpinLock) Lock() {
	const maxBlock = 64
	block := 1
	for {
		if l.TryLock() {
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

func (l *SpinLock) TryLock() bool {
	return l.isLocked.CompareAndSwap(unlocked, locked)
}

func (l *SpinLock) TryLockInTime(dur time.Duration) bool {
	const maxBlock = 64
	block := 1
	tr := time.NewTimer(dur)
	isLocked := false
loopLock:
	for {
		if l.TryLock() {
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

func (l *SpinLock) Unlock() {
	l.isLocked.Store(unlocked)
}
