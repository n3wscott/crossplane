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

	"github.com/crossplane/crossplane/internal/engine"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kcache "k8s.io/client-go/tools/cache"
	testingclock "k8s.io/utils/clock/testing"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockInformer generates controllable events for testing
type MockInformer struct {
	eventChan chan interface{}
	stopChan  chan struct{}
	handlers  []kcache.ResourceEventHandler
	mutex     sync.RWMutex
	clock     *testingclock.FakeClock
	frequency time.Duration
	running   atomic.Bool
}

func (m *MockInformer) GetInformer(ctx context.Context, obj client.Object, opts ...cache.InformerGetOption) (cache.Informer, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockInformer) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind, opts ...cache.InformerGetOption) (cache.Informer, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockInformer) RemoveInformer(ctx context.Context, obj client.Object) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockInformer) WaitForCacheSync(ctx context.Context) bool {
	//TODO implement me
	panic("implement me")
}

func (m *MockInformer) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockInformer) ActiveInformers() []schema.GroupVersionKind {
	//TODO implement me
	panic("implement me")
}

var _ engine.TrackingInformers = &MockInformer{}

func NewMockInformer(clock *testingclock.FakeClock, eventFrequency time.Duration) *MockInformer {
	return &MockInformer{
		eventChan: make(chan interface{}, 1000),
		stopChan:  make(chan struct{}),
		handlers:  make([]kcache.ResourceEventHandler, 0),
		clock:     clock,
		frequency: eventFrequency,
	}
}

func (m *MockInformer) AddEventHandler(handler kcache.ResourceEventHandler) (kcache.ResourceEventHandlerRegistration, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.handlers = append(m.handlers, handler)
	return &mockRegistration{}, nil
}

type mockRegistration struct{}

func (r *mockRegistration) HasSynced() bool { return true }

func (m *MockInformer) RemoveEventHandler(kcache.ResourceEventHandlerRegistration) error {
	return nil
}

func (m *MockInformer) Start(ctx context.Context) error {
	if !m.running.CompareAndSwap(false, true) {
		return nil // Already running
	}

	go m.eventLoop(ctx)
	return nil
}

func (m *MockInformer) Stop() {
	if m.running.CompareAndSwap(true, false) {
		close(m.stopChan)
	}
}

func (m *MockInformer) eventLoop(ctx context.Context) {
	ticker := m.clock.NewTicker(m.frequency)
	defer ticker.Stop()

	obj := &metav1.ObjectMeta{
		Name:      "test-object",
		Namespace: "test-namespace",
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C():
			// Generate update event
			m.triggerEvent(obj)
		}
	}
}

func (m *MockInformer) triggerEvent(obj interface{}) {
	m.mutex.RLock()
	handlers := make([]kcache.ResourceEventHandler, len(m.handlers))
	copy(handlers, m.handlers)
	m.mutex.RUnlock()

	// Trigger all handlers (simplified - in reality this would be more complex)
	for _, handler := range handlers {
		// Each handler would process the event
		_ = handler
	}
}
