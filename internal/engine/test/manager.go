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
	"net/http"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// MockManager implements manager.Manager for testing
type MockManager struct {
	client  client.Client
	cache   cache.Cache
	scheme  *runtime.Scheme
	elected chan struct{}
	config  interface{}
}

func NewMockManager(client client.Client) *MockManager {
	elected := make(chan struct{})
	close(elected) // Always elected for testing

	return &MockManager{
		client: client,
		//cache:   cache,
		scheme:  runtime.NewScheme(),
		elected: elected,
	}
}

var _ manager.Manager = &MockManager{}

func (m *MockManager) GetClient() client.Client                                    { return m.client }
func (m *MockManager) GetCache() cache.Cache                                       { return m.cache }
func (m *MockManager) GetScheme() *runtime.Scheme                                  { return m.scheme }
func (m *MockManager) GetConfig() *rest.Config                                     { return nil }
func (m *MockManager) GetRESTMapper() meta.RESTMapper                              { return nil }
func (m *MockManager) GetFieldIndexer() client.FieldIndexer                        { return nil }
func (m *MockManager) GetEventRecorderFor(string) record.EventRecorder             { return nil }
func (m *MockManager) GetLogger() logr.Logger                                      { return logr.Discard() }
func (m *MockManager) Elected() <-chan struct{}                                    { return m.elected }
func (m *MockManager) Add(manager.Runnable) error                                  { return nil }
func (m *MockManager) SetFields(interface{}) error                                 { return nil }
func (m *MockManager) AddMetricsExtraHandler(string, interface{}) error            { return nil }
func (m *MockManager) AddHealthzCheck(string, healthz.Checker) error               { return nil }
func (m *MockManager) AddReadyzCheck(string, healthz.Checker) error                { return nil }
func (m *MockManager) Start(context.Context) error                                 { return nil }
func (m *MockManager) GetControllerOptions() config.Controller                     { return config.Controller{} }
func (m *MockManager) GetHTTPClient() *http.Client                                 { return nil }
func (m *MockManager) GetWebhookServer() webhook.Server                            { return nil }
func (m *MockManager) GetAPIReader() client.Reader                                 { return m.client }
func (m *MockManager) AddMetricsServerExtraHandler(_ string, _ http.Handler) error { return nil }
