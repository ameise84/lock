package example

import (
	"github.com/ameise84/lock"
	"testing"
)

func TestOnce(t *testing.T) {
	once := lock.Once{}
	once.Do(func() {
		panic("")
	})
}
