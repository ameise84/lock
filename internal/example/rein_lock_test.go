package example

import (
	"github.com/ameise84/lock"
	"log"
	"sync"
	"testing"
	"time"
)

type l struct {
}

func TestReinLock(t *testing.T) {
	rLock := lock.ReinLock{}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for i := 0; i < 1000; i++ {
			rLock.Lock()
			rLock.Lock()
			log.Printf("1-->%v", i)
			rLock.Unlock()
			rLock.Unlock()
			time.Sleep(time.Millisecond)
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < 1000; i++ {
			rLock.Lock()
			rLock.Lock()
			log.Printf("2-->%v", i)
			rLock.Unlock()
			rLock.Unlock()
			time.Sleep(time.Millisecond)
		}
		wg.Done()
	}()
	wg.Wait()
}
