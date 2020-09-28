package ant_test

import (
	"testing"

	"github.com/yields/ant"
	"github.com/yields/ant/anttest"
)

func TestQueue(t *testing.T) {
	anttest.TestQueue(t, func(t testing.TB) ant.Queue {
		return ant.MemoryQueue(5)
	})
}

func BenchmarkQueue(b *testing.B) {
	anttest.BenchmarkQueue(b, func(t testing.TB) ant.Queue {
		return ant.MemoryQueue(5)
	})
}
