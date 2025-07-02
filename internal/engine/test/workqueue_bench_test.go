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
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/crossplane/crossplane/internal/workqueue"
	"k8s.io/apimachinery/pkg/types"
	workqueue2 "k8s.io/client-go/util/workqueue"
	testingclock "k8s.io/utils/clock/testing"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type WorkQueueBenchConfig struct {
	TestDurationMS      int     // Total test duration
	ProducerFrequencyMS int     // How often producer adds items (0 = as fast as possible)
	ConsumerDelayMS     int     // Delay between Get() calls in consumer
	RateLimitQPS        float32 // Rate limit QPS (0 = no rate limit)
	RateLimitBurst      int     // Rate limit burst (0 = no rate limit)
	TimeAdvanceUS       int     // Time advancement interval
	ProducerCount       int     // Number of producer goroutines
	ConsumerCount       int     // Number of consumer goroutines
}

type WorkQueueBenchResults struct {
	TotalItemsProduced int64
	TotalItemsConsumed int64
	GetCallTimes       []time.Time
	ProducerRate       float64
	ConsumerRate       float64
	TestDuration       time.Duration
	QueueLengthSamples []int
	QueueLengthTimes   []time.Time
	AverageQueueLength float64
	MaxQueueLength     int
}

type WorkQueueBenchHarness struct {
	clock       *testingclock.FakeClock
	workQueue   *CountingWorkQueue[reconcile.Request]
	rateLimiter *CountingRateLimiter[reconcile.Request]
	config      WorkQueueBenchConfig

	// Metrics
	itemsProduced      atomic.Int64
	itemsConsumed      atomic.Int64
	getCallTimes       []time.Time
	getCallTimesMutex  sync.Mutex
	queueLengthSamples []int
	queueLengthTimes   []time.Time
	queueLengthMutex   sync.Mutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewWorkQueueBenchHarness(config WorkQueueBenchConfig) *WorkQueueBenchHarness {
	clock := testingclock.NewFakeClock(time.Now())

	var rateLimiter workqueue2.TypedRateLimiter[reconcile.Request]
	var countingRateLimiter *CountingRateLimiter[reconcile.Request]

	if config.RateLimitQPS > 0 {
		rateLimiter = &workqueue.TypedBucketRateLimiter[reconcile.Request]{
			Limiter: workqueue.NewLimiter(workqueue.Limit(config.RateLimitQPS), config.RateLimitBurst, clock),
		}
	} else {
		rateLimiter = workqueue.NewTypedMaxOfRateLimiter[reconcile.Request]()
	}

	countingRateLimiter = NewCountingRateLimiter[reconcile.Request](rateLimiter)

	wq := workqueue.NewTypedRateLimitingQueueWithConfig[reconcile.Request](countingRateLimiter, workqueue.TypedRateLimitingQueueConfig[reconcile.Request]{
		Clock: clock,
		Name:  "bench-queue",
	})

	countingQueue := NewCountingWorkQueue[reconcile.Request](wq)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.TestDurationMS)*time.Millisecond)

	return &WorkQueueBenchHarness{
		clock:       clock,
		workQueue:   countingQueue,
		rateLimiter: countingRateLimiter,
		config:      config,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (h *WorkQueueBenchHarness) Start() {
	// Start queue length monitoring
	h.wg.Add(1)
	go h.queueLengthMonitor()

	// Start time advancement
	h.wg.Add(1)
	go h.timeAdvancer()

	// Start producers
	for i := 0; i < h.config.ProducerCount; i++ {
		h.wg.Add(1)
		go h.producer(i)
	}

	// Start consumers
	for i := 0; i < h.config.ConsumerCount; i++ {
		h.wg.Add(1)
		go h.consumer(i)
	}
}

func (h *WorkQueueBenchHarness) Stop() {
	h.cancel()
	h.workQueue.ShutDown()
	h.wg.Wait()
}

func (h *WorkQueueBenchHarness) timeAdvancer() {
	defer h.wg.Done()

	// Advance time in small increments to simulate real time passage
	testDuration := time.Duration(h.config.TestDurationMS) * time.Millisecond
	timeAdvance := time.Duration(h.config.TimeAdvanceUS) * time.Microsecond
	totalSteps := int(testDuration / timeAdvance)

	fmt.Println("START -->", h.clock.Now())
	for i := 0; i < totalSteps; i++ {
		select {
		case <-h.ctx.Done():
			return
		default:
			h.clock.Step(timeAdvance)
			time.Sleep(time.Nanosecond) // Give goroutines time to process
		}
	}
	fmt.Println("STOP -->", h.clock.Now())
	h.Stop()
}

func (h *WorkQueueBenchHarness) producer(id int) {
	defer h.wg.Done()

	var tickerChan <-chan time.Time

	if h.config.ProducerFrequencyMS == 0 {
		h.config.ProducerFrequencyMS = 1
	}

	ticker := h.clock.NewTicker(time.Duration(h.config.ProducerFrequencyMS) * time.Millisecond)
	defer ticker.Stop()
	tickerChan = ticker.C()

	itemCounter := 0

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-tickerChan:
			// Controlled frequency
			h.addItem(id, itemCounter)
			itemCounter++
		}
	}
}

func (h *WorkQueueBenchHarness) addItem(producerID, itemID int) {
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test",
			Name:      "test",
		},
	}
	h.workQueue.AddRateLimited(req)
	h.itemsProduced.Add(1)
}

func (h *WorkQueueBenchHarness) consumer(id int) {
	defer h.wg.Done()

	consumerDelay := time.Duration(h.config.ConsumerDelayMS) * time.Millisecond

	for {
		select {
		case <-h.ctx.Done():
			return
		default:
			h.getCallTimesMutex.Lock()
			h.getCallTimes = append(h.getCallTimes, h.clock.Now())
			h.getCallTimesMutex.Unlock()

			item, shutdown := h.workQueue.Get()
			if shutdown {
				return
			}

			h.itemsConsumed.Add(1)

			// Simulate processing time
			if consumerDelay > 0 {
				<-h.clock.After(consumerDelay)
			}
			time.Sleep(time.Nanosecond)
			//fmt.Println("Done at -->", h.clock.Now())

			h.workQueue.Done(item)
			h.workQueue.Forget(item) // TODO: we could choose to do this every 10 or something to make it more interesting.
		}
	}
}

func (h *WorkQueueBenchHarness) queueLengthMonitor() {
	defer h.wg.Done()

	ticker := h.clock.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C():
			length := h.workQueue.Len()
			now := h.clock.Now()

			h.queueLengthMutex.Lock()
			h.queueLengthSamples = append(h.queueLengthSamples, length)
			h.queueLengthTimes = append(h.queueLengthTimes, now)
			h.queueLengthMutex.Unlock()
		}
	}
}

func (h *WorkQueueBenchHarness) GetResults() WorkQueueBenchResults {
	testDuration := time.Duration(h.config.TestDurationMS) * time.Millisecond
	produced := h.itemsProduced.Load()
	consumed := h.itemsConsumed.Load()

	var avgQueueLength float64
	var maxQueueLength int

	h.queueLengthMutex.Lock()
	if len(h.queueLengthSamples) > 0 {
		sum := 0
		for _, length := range h.queueLengthSamples {
			sum += length
			if length > maxQueueLength {
				maxQueueLength = length
			}
		}
		avgQueueLength = float64(sum) / float64(len(h.queueLengthSamples))
	}
	queueLengthSamples := make([]int, len(h.queueLengthSamples))
	copy(queueLengthSamples, h.queueLengthSamples)
	queueLengthTimes := make([]time.Time, len(h.queueLengthTimes))
	copy(queueLengthTimes, h.queueLengthTimes)
	h.queueLengthMutex.Unlock()

	h.getCallTimesMutex.Lock()
	getCallTimes := make([]time.Time, len(h.getCallTimes))
	copy(getCallTimes, h.getCallTimes)
	h.getCallTimesMutex.Unlock()

	return WorkQueueBenchResults{
		TotalItemsProduced: produced,
		TotalItemsConsumed: consumed,
		GetCallTimes:       getCallTimes,
		ProducerRate:       float64(produced) / testDuration.Seconds(),
		ConsumerRate:       float64(consumed) / testDuration.Seconds(),
		TestDuration:       testDuration,
		QueueLengthSamples: queueLengthSamples,
		QueueLengthTimes:   queueLengthTimes,
		AverageQueueLength: avgQueueLength,
		MaxQueueLength:     maxQueueLength,
	}
}

func generateItemName(producerID, itemID int) string {
	return "producer-" + strconv.Itoa(producerID) + "-item-" + strconv.Itoa(itemID)
}

func TestWorkQueueBench(t *testing.T) {
	testCases := map[string]struct {
		config WorkQueueBenchConfig
		want   struct {
			minConsumerRate float64
			maxQueueLength  int
		}
	}{
		"FastProducerSlowConsumer": {
			config: WorkQueueBenchConfig{
				TestDurationMS:      5000,
				ProducerFrequencyMS: 1,  // As fast as possible
				ConsumerDelayMS:     10, // Slow consumer
				RateLimitQPS:        0,  // No rate limit
				RateLimitBurst:      0,
				TimeAdvanceUS:       1,
				ProducerCount:       1,
				ConsumerCount:       1,
			},
			want: struct {
				minConsumerRate float64
				maxQueueLength  int
			}{
				minConsumerRate: 50.0, // At least 50/s with 10ms delays
				maxQueueLength:  100,  // Queue should build up
			},
		},
		"RateLimitedProducer": {
			config: WorkQueueBenchConfig{
				TestDurationMS:      3000,
				ProducerFrequencyMS: 0,     // As fast as possible
				ConsumerDelayMS:     1,     // Fast consumer
				RateLimitQPS:        100.0, // Rate limited
				RateLimitBurst:      5,
				TimeAdvanceUS:       1,
				ProducerCount:       1,
				ConsumerCount:       1,
			},
			want: struct {
				minConsumerRate float64
				maxQueueLength  int
			}{
				minConsumerRate: 5.0, // Should be limited to ~10/s
				maxQueueLength:  20,  // Small queue with rate limiting
			},
		},
		"MultiProducerConsumer": {
			config: WorkQueueBenchConfig{
				TestDurationMS:      4000,
				ProducerFrequencyMS: 10, // 100/s per producer
				ConsumerDelayMS:     5,  // 200/s per consumer
				RateLimitQPS:        0,  // No rate limit
				RateLimitBurst:      0,
				TimeAdvanceUS:       1,
				ProducerCount:       3,
				ConsumerCount:       2,
			},
			want: struct {
				minConsumerRate float64
				maxQueueLength  int
			}{
				minConsumerRate: 200.0, // ~300/s production, ~400/s consumption
				maxQueueLength:  50,    // Should balance out
			},
		},
		"HighThroughput": {
			config: WorkQueueBenchConfig{
				TestDurationMS:      2000,
				ProducerFrequencyMS: 0, // As fast as possible
				ConsumerDelayMS:     0, // As fast as possible
				RateLimitQPS:        0, // No rate limit
				RateLimitBurst:      0,
				TimeAdvanceUS:       1,
				ProducerCount:       2,
				ConsumerCount:       2,
			},
			want: struct {
				minConsumerRate float64
				maxQueueLength  int
			}{
				minConsumerRate: 500.0, // High throughput
				maxQueueLength:  1000,  // May build up initially
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			harness := NewWorkQueueBenchHarness(tc.config)

			harness.Start()

			// Wait for test duration (simulated)
			<-harness.ctx.Done()

			results := harness.GetResults()

			t.Logf("Test %s Results:", name)
			t.Logf("  Produced: %d items (%.2f/s)", results.TotalItemsProduced, results.ProducerRate)
			t.Logf("  Consumed: %d items (%.2f/s)", results.TotalItemsConsumed, results.ConsumerRate)
			t.Logf("  Get() calls: %d (%.2f/s)", len(results.GetCallTimes), float64(len(results.GetCallTimes))/results.TestDuration.Seconds())
			t.Logf("  Queue length: avg=%.1f, max=%d", results.AverageQueueLength, results.MaxQueueLength)
			t.Logf("  Test duration: %v", results.TestDuration)

			t.Logf("  WorkQueue metrics:")
			t.Logf("    Add: %d (%.2f/s)", harness.workQueue.addCount.Load(), harness.workQueue.AddAverageRate(results.TestDuration))
			t.Logf("    Done: %d (%.2f/s)", harness.workQueue.doneCount.Load(), harness.workQueue.DoneAverageRate(results.TestDuration))
			t.Logf("    AddRateLimited: %d (%.2f/s)", harness.workQueue.addRateLimitedCount.Load(), harness.workQueue.AddRateLimitedAverageRate(results.TestDuration))
			t.Logf("    AddAfter: %d (%.2f/s)", harness.workQueue.addAfterCount.Load(), harness.workQueue.AddAfterAverageRate(results.TestDuration))

			t.Logf("  RateLimiter metrics:")
			t.Logf("    When() calls: %d (%.2f/s)", harness.rateLimiter.whenCount.Load(), harness.rateLimiter.WhenAverageRate(results.TestDuration))
			t.Logf("    Forget() calls: %d (%.2f/s)", harness.rateLimiter.forgetCount.Load(), harness.rateLimiter.ForgetAverageRate(results.TestDuration))
			t.Logf("    Average delay: %.2fms", harness.rateLimiter.DelayAverageRate())

			// Validate minimum consumer rate
			if results.ConsumerRate < tc.want.minConsumerRate {
				t.Errorf("Consumer rate %.2f/s is below minimum expected %.2f/s",
					results.ConsumerRate, tc.want.minConsumerRate)
			}

			// Validate reasonable queue behavior
			if results.MaxQueueLength > tc.want.maxQueueLength {
				t.Errorf("Max queue length %d exceeds expected maximum %d",
					results.MaxQueueLength, tc.want.maxQueueLength)
			}

			// Validate we processed some items
			if results.TotalItemsConsumed == 0 {
				t.Error("Expected some items to be consumed, got 0")
			}

			// Validate Get() was called
			if len(results.GetCallTimes) == 0 {
				t.Error("Expected Get() to be called, got 0 calls")
			}
		})
	}
}
