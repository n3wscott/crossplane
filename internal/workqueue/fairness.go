package workqueue

import (
	"time"
)

type TODO[T comparable] interface {
	Add(item T)
	Len() int
	Get() (item T, shutdown bool)
	Done(item T)
	ShutDown()
	ShutDownWithDrain()
	ShuttingDown() bool

	// DelayingInterface = TypedInterface + AddAfter()

	// AddAfter adds an item to the workqueue after the indicated duration has passed
	AddAfter(item T, duration time.Duration)

	// RateLimitingInterface = DelayingInterface +

	// AddRateLimited adds an item to the workqueue after the rate limiter says it's ok
	AddRateLimited(item T)

	// Forget indicates that an item is finished being retried.  Doesn't matter whether it's for perm failing
	// or for success, we'll stop the rate limiter from tracking it.  This only clears the `rateLimiter`, you
	// still have to call `Done` on the queue.
	Forget(item T)

	// NumRequeues returns back how many times the item was requeued
	NumRequeues(item T) int
}

// What we want to happen:
// Inputs:
// - MaxReconcileRate -
//

// Forget() is ending a session with an object.
// Done() is ending a single loop with an object.
// AddAfter and AddRateLimited are the same thing except the after time is taken from the rate limiter.
// Add seems to bypass the delay queue, this is what we want to fix.
//

// A reconciler rate limiter will need the following:
// - The global rate limiter
// - The local rate limiter
// - The error backoff limiter.
