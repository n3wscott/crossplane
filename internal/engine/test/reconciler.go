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
	"context"
	"sync"
	"sync/atomic"
	"time"

	testingclock "k8s.io/utils/clock/testing"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MockReconciler implements reconcile.Reconciler with call tracking
type MockReconciler struct {
	reconcileFunc func(context.Context, reconcile.Request) (reconcile.Result, error)
	callCount     atomic.Int64
	callTimes     []time.Time
	mutex         sync.RWMutex
	clock         *testingclock.FakeClock
}

func NewMockReconciler(clock *testingclock.FakeClock, reconcileFunc func(context.Context, reconcile.Request) (reconcile.Result, error)) *MockReconciler {
	if reconcileFunc == nil {
		reconcileFunc = func(context.Context, reconcile.Request) (reconcile.Result, error) {
			return reconcile.Result{}, nil
		}
	}
	return &MockReconciler{
		reconcileFunc: reconcileFunc,
		callTimes:     make([]time.Time, 0),
		clock:         clock,
	}
}

func (m *MockReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.callCount.Add(1)
	m.callTimes = append(m.callTimes, m.clock.Now())

	res, err := m.reconcileFunc(ctx, req)

	//fmt.Println("Sleeping", m.clock.Now())
	//<-m.clock.After(10 * time.Millisecond)
	//fmt.Println("Continue", m.clock.Now())

	return res, err
}

func (m *MockReconciler) GetCallCount() int64 {
	return m.callCount.Load()
}

func (m *MockReconciler) GetCallTimes() []time.Time {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	times := make([]time.Time, len(m.callTimes))
	copy(times, m.callTimes)
	return times
}

func (m *MockReconciler) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.callCount.Store(0)
	m.callTimes = m.callTimes[:0]
}
