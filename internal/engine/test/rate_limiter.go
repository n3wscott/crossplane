/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test

import (
	"sync/atomic"
	"time"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CountingRateLimiter[T comparable] struct {
	whenCount   atomic.Int64
	delayCount  atomic.Int64
	forgetCount atomic.Int64
	limiter     workqueue.TypedRateLimiter[T]
}

func NewCountingRateLimiter[T comparable](limiter workqueue.TypedRateLimiter[T]) *CountingRateLimiter[T] {
	return &CountingRateLimiter[T]{
		limiter: limiter,
	}
}

func (m *CountingRateLimiter[T]) When(item T) time.Duration {
	m.whenCount.Add(1)
	d := m.limiter.When(item)
	m.delayCount.Add(d.Milliseconds())
	return d
}

func (m *CountingRateLimiter[T]) Forget(item T) {
	m.forgetCount.Add(1)
	m.limiter.Forget(item)
}

func (m *CountingRateLimiter[T]) NumRequeues(item T) int {
	return m.limiter.NumRequeues(item)
}

func (m *CountingRateLimiter[T]) ForgetAverageRate(d time.Duration) float64 {
	if d.Seconds() == 0 {
		return 0
	}
	return float64(m.forgetCount.Load()) / d.Seconds()
}

func (m *CountingRateLimiter[T]) WhenAverageRate(d time.Duration) float64 {
	if d.Seconds() == 0 {
		return 0
	}
	return float64(m.whenCount.Load()) / d.Seconds()
}

func (m *CountingRateLimiter[T]) DelayAverageRate() float64 {
	return float64(m.delayCount.Load()) / (float64(m.forgetCount.Load()) * float64(1000))
}

var _ workqueue.TypedRateLimiter[reconcile.Request] = &CountingRateLimiter[reconcile.Request]{}
