package utils

import (
	"sync"
	"testing"
)

func TestNewContainerWithDefaultAndRead(t *testing.T) {
	c := NewContainerWithDefault[int](42)

	if got := c.Read(); got != 42 {
		t.Errorf("Read() = %d, want 42", got)
	}
}

func TestContainerStoreOverwrites(t *testing.T) {
	c := NewContainerWithDefault("initial")

	c.Store("updated")

	if got := c.Read(); got != "updated" {
		t.Errorf("Read() = %q, want %q", got, "updated")
	}
}

func TestContainerStoreStruct(t *testing.T) {
	type payload struct {
		A int
		B string
	}

	c := NewContainer[payload]()
	c.Store(payload{A: 1, B: "one"})

	got := c.Read()
	if got.A != 1 || got.B != "one" {
		t.Errorf("Read() = %+v, want {1 one}", got)
	}
}

// TestContainerConcurrentAccess exercises the atomic pointer under the race detector.
func TestContainerConcurrentAccess(t *testing.T) {
	c := NewContainerWithDefault[int](0)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)

		go func(v int) {
			defer wg.Done()
			c.Store(v)
		}(i)

		go func() {
			defer wg.Done()
			_ = c.Read()
		}()
	}

	wg.Wait()
}
