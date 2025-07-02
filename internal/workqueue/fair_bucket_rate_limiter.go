package workqueue

import (
	"time"
)

// FairBucketRateLimiter adapts a standard bucket to the workqueue ratelimiter API
type FairBucketRateLimiter[T comparable] struct {
	*Limiter
}

func (r *FairBucketRateLimiter[T]) When(item T) time.Duration {
	return r.Limiter.Reserve().Delay()
}

func (r *FairBucketRateLimiter[T]) NumRequeues(item T) int {
	return 0
}

func (r *FairBucketRateLimiter[T]) Forget(item T) {
}
