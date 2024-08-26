package example

import (
	"github.com/ameise84/lock"
	"testing"
)

func TestLock(t *testing.T) {
	lock.CheckDead(true, "aa")
	l := lock.SpinLock{}
	l.Lock()
	l.Unlock()
	l.Lock()
	l.Lock()
	l.Unlock()
}
