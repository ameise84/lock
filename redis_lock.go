package lock

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/rs/xid"
	"runtime"
	"sync"
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

func NewRedisLock(cli redis.UniversalClient, key string, ttl ...time.Duration) sync.Locker {
	scriptOnce.Do(func() {
		scriptSHA = cli.ScriptLoad(context.Background(), script).Val()
	})
	t := 10 * time.Second
	if ttl != nil && len(ttl) > 0 && ttl[0] > 0 {
		t = ttl[0]
	}
	return &redisLock{
		cli:   cli,
		key:   "_redis_lock:" + key,
		value: xid.New().String(),
		ttl:   t,
	}
}

type redisLock struct {
	mu     SpinLock
	cli    redis.UniversalClient
	key    string
	value  string
	ttl    time.Duration
	isLock bool
	tr     *time.Timer
	wg     sync.WaitGroup
}

func (l *redisLock) Lock() {
	const maxBlock = 256
	block := 1
	for {
		if ok := l.TryLock(); ok {
			return
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
	ok := l.mu.TryLock()
	if !ok {
		return false
	}
	r := l.cli.SetNX(context.Background(), l.key, l.value, l.ttl)
	l.isLock = r.Val()
	if l.isLock {
		l.wg.Add(1)
		go l.startWatchDog()
		l.wg.Wait()
	} else {
		l.mu.Unlock()
	}
	return l.isLock
}

func (l *redisLock) Unlock() {
	if l.isLock {
		l.cli.EvalSha(context.Background(), scriptSHA, []string{l.key}, l.value)
		l.isLock = false
		l.wg.Add(1)
		l.tr.Reset(0)
		l.wg.Wait()
		l.mu.Unlock()
	}
	return
}

func (l *redisLock) startWatchDog() {
	k := l.ttl / 2
	l.tr = time.NewTimer(l.ttl / 2)
	l.wg.Done()
loopFor:
	for {
		select {
		case <-l.tr.C:
			if !l.isLock {
				break loopFor
			}
			r := l.cli.Expire(context.Background(), l.key, l.ttl)
			if !r.Val() {
				break loopFor
			}
			l.tr.Reset(k)
		}
	}
	l.isLock = false
	l.tr.Stop()
	l.tr = nil
	l.wg.Done()
}
