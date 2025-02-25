package limit

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
)

type LeakyBucketLimiter struct {
	capacity   int
	windowSize time.Duration
	requests   *list.List
	avail      atomic.Int64
	mu         sync.Mutex
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
			l.requests.PushBack(now)
		}
		l.mu.Unlock()
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for l.requests.Len() > 0 {
		front := l.requests.Front()
		if now.Sub(front.Value.(time.Time)) < l.windowSize {
			break
		}
		l.requests.Remove(front)
	}

	used := l.requests.Len()
	if used+c > l.capacity {
		return false
	}

	for i := 0; i < c; i++ {
		l.requests.PushBack(now)
	}
	l.avail.Store(int64(l.capacity - (used + c)))
	return true
}

func NewLeakyBucketLimiter(capacity int, windowSize time.Duration) *LeakyBucketLimiter {
	if capacity <= 0 || windowSize <= 0 {
		return nil
	}
	l := &LeakyBucketLimiter{
		capacity:   capacity,
		windowSize: windowSize,
		requests:   list.New(),
	}
	l.avail.Store(int64(capacity))
	return l
}

var LeakyBucket = NewLeakyBucketLimiter(1248, time.Second)
