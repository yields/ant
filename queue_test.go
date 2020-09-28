package ant_test

import (
	"testing"

	"github.com/yields/ant"
	"github.com/yields/ant/anttest"
)

func TestQueue(t *testing.T) {
	anttest.Queue(t, func(t testing.TB) ant.Queue {
		return ant.MemoryQueue(5)
	})
}
