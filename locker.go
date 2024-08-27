package lock

import "time"

type Locker interface {
	Lock()
	Unlock()
	TryLock() bool
	TryLockInTime(time.Duration) bool
}
