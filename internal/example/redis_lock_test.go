package example

import (
	"github.com/ameise84/lock"
	"github.com/redis/go-redis/v9"
	"runtime"
	"testing"
	"time"
)

func TestRedisLock(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Username: "",
		Password: "",
	})
	l := lock.NewRedisLock(client, "aa")

	for i := 0; i < 20; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				l.Lock()
				time.Sleep(2 * time.Second)
				l.Unlock()
				runtime.Gosched()
			}
		}()
	}

	l2 := lock.NewRedisLock(client, "aa")
	for i := 0; i < 20; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				l2.Lock()
				time.Sleep(2 * time.Second)
				l2.Unlock()
				runtime.Gosched()
			}
		}()
	}
	time.Sleep(1000 * time.Second)
}
