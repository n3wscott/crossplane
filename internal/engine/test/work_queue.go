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

type CountingWorkQueue[T comparable] struct {
	addCount            atomic.Int64
	addAfterCount       atomic.Int64
	addRateLimitedCount atomic.Int64
	doneCount           atomic.Int64
	forgetCount         atomic.Int64
	workQueue           workqueue.TypedRateLimitingInterface[T]
}

func (m *CountingWorkQueue[T]) Add(item T) {
	m.addCount.Add(1)
	m.workQueue.Add(item)
}

func (m *CountingWorkQueue[T]) Len() int {
	return m.workQueue.Len()
}

func (m *CountingWorkQueue[T]) Get() (item T, shutdown bool) {
	return m.workQueue.Get()
}

func (m *CountingWorkQueue[T]) Done(item T) {
	m.doneCount.Add(1)
	m.workQueue.Done(item)
}

func (m *CountingWorkQueue[T]) ShutDown() {
	m.workQueue.ShutDown()
}

func (m *CountingWorkQueue[T]) ShutDownWithDrain() {
	m.workQueue.ShutDownWithDrain()
}

func (m *CountingWorkQueue[T]) ShuttingDown() bool {
	return m.workQueue.ShuttingDown()
}

func (m *CountingWorkQueue[T]) AddAfter(item T, duration time.Duration) {
	m.addAfterCount.Add(1)
	m.workQueue.AddAfter(item, duration)
}

func (m *CountingWorkQueue[T]) AddRateLimited(item T) {
	m.addRateLimitedCount.Add(1)
	m.workQueue.AddRateLimited(item)
}

func (m *CountingWorkQueue[T]) Forget(item T) {
	m.forgetCount.Add(1)
	m.workQueue.Forget(item)
}

func (m *CountingWorkQueue[T]) NumRequeues(item T) int {
	return m.workQueue.NumRequeues(item)
}

func NewCountingWorkQueue[T comparable](workQueue workqueue.TypedRateLimitingInterface[T]) *CountingWorkQueue[T] {
	return &CountingWorkQueue[T]{
		workQueue: workQueue,
	}
}

func (m *CountingWorkQueue[T]) AddAverageRate(d time.Duration) float64 {
	if d.Seconds() == 0 {
		return 0
	}
	return float64(m.addCount.Load()) / d.Seconds()
}

func (m *CountingWorkQueue[T]) AddAfterAverageRate(d time.Duration) float64 {
	if d.Seconds() == 0 {
		return 0
	}
	return float64(m.addAfterCount.Load()) / d.Seconds()
}

func (m *CountingWorkQueue[T]) AddRateLimitedAverageRate(d time.Duration) float64 {
	if d.Seconds() == 0 {
		return 0
	}
	return float64(m.addRateLimitedCount.Load()) / d.Seconds()
}

func (m *CountingWorkQueue[T]) DoneAverageRate(d time.Duration) float64 {
	if d.Seconds() == 0 {
		return 0
	}
	return float64(m.doneCount.Load()) / d.Seconds()
}

func (m *CountingWorkQueue[T]) ForgetAverageRate(d time.Duration) float64 {
	if d.Seconds() == 0 {
		return 0
	}
	return float64(m.forgetCount.Load()) / d.Seconds()
}

var _ workqueue.TypedRateLimitingInterface[reconcile.Request] = &CountingWorkQueue[reconcile.Request]{}
