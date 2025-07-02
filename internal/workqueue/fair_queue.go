package workqueue

import (
	"slices"
)

// FairQueue is the underlying storage for items. The functions below are always
// called from the same goroutine.
type FairQueue[T comparable] interface {
	// Contains returns the existence of an item in the queue.
	Contains(item T) bool
	// Touch can be hooked when an existing item is added again. This may be
	// useful if the implementation allows priority change for the given item.
	Touch(item T)
	// Push adds a new item.
	Push(item T)
	// Len tells the total number of items.
	Len() int
	// Pop retrieves an item.
	Pop() (item T)
}

// DefaultFairQueue is a slice based FIFO queue.
func DefaultFairQueue[T comparable]() FairQueue[T] {
	return new(fairQueue[T])
}

// fairQueue is a slice which implements FairQueue.
type fairQueue[T comparable] []T

func (q *fairQueue[T]) Touch(item T) {}

func (q *fairQueue[T]) Contains(item T) bool {
	return slices.Contains(*q, item)
}

func (q *fairQueue[T]) Push(item T) {
	*q = append(*q, item)
}

func (q *fairQueue[T]) Len() int {
	return len(*q)
}

func (q *fairQueue[T]) Pop() (item T) {
	item = (*q)[0]

	// The underlying array still exists and reference this object, so the object will not be garbage collected.
	(*q)[0] = *new(T)
	*q = (*q)[1:]

	return item
}
