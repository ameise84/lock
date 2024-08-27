//go:build !debug

package lock

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/rs/xid"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	script = `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
`
)

var (
	scriptOnce sync.Once
	scriptSHA  string
)

func NewRedisLock(cli redis.UniversalClient, key string, ttl ...time.Duration) Locker {
	scriptOnce.Do(func() {
		scriptSHA = cli.ScriptLoad(context.Background(), script).Val()
	})
	t := 30 * time.Second
	if ttl != nil && len(ttl) > 0 && ttl[0] > 0 {
		t = ttl[0]
	}
	return &redisLock{
		cli:     cli,
		key:     "_redis_lock:" + key,
		value:   xid.New().String(),
		ttl:     t,
		dogChan: make(chan struct{}, 1),
	}
}

type redisLock struct {
	mu      SpinLock
	cli     redis.UniversalClient
	key     string
	value   string
	ttl     time.Duration
	isLock  atomic.Bool
	tr      *time.Timer
	wg      sync.WaitGroup
	dogChan chan struct{}
}

func (l *redisLock) Lock() {
	const maxBlock = 64
	block := 1
	l.mu.Lock()
	for {
		if l.checkRedisLock() {
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

func (l *redisLock) TryLock() bool {
	if !l.mu.TryLock() {
		return false
	}
	if !l.checkRedisLock() {
		l.mu.Unlock()
		return false
	}
	return true
}

func (l *redisLock) TryLockInTime(dur time.Duration) bool {
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
	tr.Stop()
	select {
	case <-tr.C:
	default:
	}
	return isLocked
}

func (l *redisLock) Unlock() {
	if l.isLock.CompareAndSwap(true, false) {
		l.cli.EvalSha(context.Background(), scriptSHA, []string{l.key}, l.value)
		l.tr.Reset(0)
	}
	<-l.dogChan
	l.tr = nil
	l.mu.Unlock()
	return
}

func (l *redisLock) checkRedisLock() bool {
	r := l.cli.SetNX(context.Background(), l.key, l.value, l.ttl)
	ok := r.Val()
	l.isLock.Store(ok)
	if ok {
		l.wg.Add(1)
		l.tr = time.NewTimer(l.ttl / 2)
		go l.startWatchDog()
		l.wg.Wait()
	}
	return ok
}

func (l *redisLock) startWatchDog() {
	k := l.ttl / 2
	l.wg.Done()
loopFor:
	for {
		select {
		case <-l.tr.C:
			if !l.isLock.Load() {
				break loopFor
			}
			r := l.cli.Expire(context.Background(), l.key, l.ttl)
			if !r.Val() {
				_gLogger.Printf("redis key[%s] expired failed", l.key)
				break loopFor
			}
			l.tr.Reset(k)
		}
	}
	l.isLock.Store(false)
	l.tr.Stop()
	select {
	case <-l.tr.C:
	default:
	}
	l.dogChan <- struct{}{}
}
