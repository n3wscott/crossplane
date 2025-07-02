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
	"sync"
	"time"
)

// Recorder measures actual reconcile rates
type Recorder struct {
	measurements []time.Time
	mutex        sync.RWMutex
}

func NewRateLimitRecorder() *Recorder {
	return &Recorder{
		measurements: make([]time.Time, 0),
	}
}

func (r *Recorder) Record(t time.Time) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.measurements = append(r.measurements, t)
}

func (r *Recorder) GetRate(window time.Duration, endTime time.Time) float64 {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	startTime := endTime.Add(-window)
	count := 0

	for _, t := range r.measurements {
		if t.After(startTime) && !t.After(endTime) {
			count++
		}
	}

	return float64(count) / window.Seconds()
}

func (r *Recorder) Reset() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.measurements = r.measurements[:0]
}
