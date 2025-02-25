package limit

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"time"
)

type Request struct {
	Timestamp time.Time
	index     int
}

type RequestHeap []Request

func (h RequestHeap) Len() int           { return len(h) }
func (h RequestHeap) Less(i, j int) bool { return h[i].Timestamp.Before(h[j].Timestamp) }
func (h RequestHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *RequestHeap) Push(x interface{}) {
	req := x.(Request)
	req.index = len(*h)
	*h = append(*h, req)
}
func (h *RequestHeap) Pop() interface{} {
	old := *h
	n := len(old)
	req := old[n-1]
	old[n-1] = Request{}
	req.index = -1
	*h = old[0 : n-1]
	return req
}

type LeakyBucketLimiter struct {
	capacity   int
	windowSize time.Duration
	requests   RequestHeap
	avail      atomic.Int64
	mu         sync.Mutex
}

func NewLeakyBucketLimiter(capacity int, windowSize time.Duration) *LeakyBucketLimiter {
	if capacity <= 0 || windowSize <= 0 {
		return nil
	}
	l := &LeakyBucketLimiter{
		capacity:   capacity,
		windowSize: windowSize,
		requests:   make(RequestHeap, 0, capacity), // 预分配容量
	}
	heap.Init(&l.requests)
	l.avail.Store(int64(capacity))
	return l
}

func (l *LeakyBucketLimiter) Acquire(c int) bool {
	if l == nil || c <= 0 {
		return true
	}

	avail := l.avail.Load()
	if avail >= int64(c) && l.avail.CompareAndSwap(avail, avail-int64(c)) {
		l.mu.Lock()
		now := time.Now()
		for i := 0; i < c; i++ {
			heap.Push(&l.requests, Request{Timestamp: now})
		}
		l.mu.Unlock()
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for l.requests.Len() > 0 && now.Sub(l.requests[0].Timestamp) >= l.windowSize {
		heap.Pop(&l.requests)
	}

	used := l.requests.Len()
	if used+c > l.capacity {
		return false
	}

	for i := 0; i < c; i++ {
		heap.Push(&l.requests, Request{Timestamp: now})
	}
	l.avail.Store(int64(l.capacity - (used + c)))
	return true
}

func (l *LeakyBucketLimiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.requests.Len()
}

var LeakyBucket = NewLeakyBucketLimiter(2048, time.Second)
