package utils

import "sync/atomic"

type Container[T any] struct {
	value atomic.Pointer[T]
}

func NewContainer[T any]() *Container[T] {
	return &Container[T]{
		value: atomic.Pointer[T]{},
	}
}
func NewContainerWithDefault[T any](value T) *Container[T] {
	container := NewContainer[T]()
	container.Store(value)

	return container
}

func (c *Container[T]) Store(val T) {
	_ = c.value.Swap(&val)
}

func (c *Container[T]) Read() T {
	value := c.value.Load()
	return *value
}
