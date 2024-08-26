package lock

import (
	"sync/atomic"
)

type Once struct {
	done uint32
	m    SpinLock
}

func (o *Once) Reset() {
	atomic.StoreUint32(&o.done, 0)
}

func (o *Once) Do(f func()) {
	if f == nil {
		atomic.StoreUint32(&o.done, 1)
		return
	}
	if atomic.LoadUint32(&o.done) == 0 {
		o.doSlow(f)
	}
}

func (o *Once) doSlow(f func()) {
	o.m.Lock()
	if o.done == 0 {
		f()
		atomic.StoreUint32(&o.done, 1)
	}
	o.m.Unlock()
}
