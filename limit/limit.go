package limit

import (
	"fmt"
	"sync"
	"time"

	"github.com/48Club/service_agent/types"
)

var (
	Limits IPBasedRateLimiters
)

func (_limits IPBasedRateLimiters) Allow(ip string, pass bool, count int, res *types.LimitResponse) (tonanyRequests bool) {
	for _, limit := range _limits {
		limiter := limit.Allow(ip, pass, count)

		if res != nil {
			res.Limit = append(res.Limit, fmt.Sprintf("%d/%s", limiter.Limit, limiter.Wind))
			res.Remaining = append(res.Remaining, fmt.Sprintf("%d/%s", limiter.Limit-limiter.Used, limiter.Wind))
		}
		if !limiter.Allow {
			tonanyRequests = true
		}
	}

	return
}

func (iprls IPBasedRateLimiters) Prune(ip string) {
	for _, rl := range iprls {
		rl.mu.Lock()
		delete(rl.limiters, ip)
		rl.mu.Unlock()
	}
}

func init() {
	Limits = IPBasedRateLimiters{
		NewIPBasedRateLimiter(80, time.Second*5), // [9.6|16]qps
		NewIPBasedRateLimiter(720, time.Minute),  // [7.2|12]qps
	}
}

// 修改为 FixedWindowRateLimiter
type FixedWindowRateLimiter struct {
	mu        sync.Mutex
	count     int           // 当前窗口内的请求计数
	lastReset time.Time     // 上次窗口重置时间
	limit     int           // 限制配额
	window    time.Duration // 窗口大小
	window2   string        // 窗口标识
}

func NewFixedWindowRateLimiter(limit int, window time.Duration, window2 string) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{
		limit:     limit,
		window:    window,
		window2:   window2,
		lastReset: time.Now(),
	}
}

func (rl *FixedWindowRateLimiter) Allow(pass bool, count int) IsAllow {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	// 检查是否需要重置窗口
	if now.Sub(rl.lastReset) >= rl.window {
		rl.count = 0
		rl.lastReset = now
	}

	used := rl.count

	if used+count < rl.limit {
		if pass {
			return IsAllow{true, used, rl.limit, rl.window2}
		}
		// 增加计数
		rl.count += count
		return IsAllow{true, used + count, rl.limit, rl.window2}
	}

	return IsAllow{false, used, rl.limit, rl.window2}
}

type IPBasedRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*FixedWindowRateLimiter // 修改为 FixedWindowRateLimiter
	limit    int
	window   time.Duration
	window2  string
}

type IPBasedRateLimiters []*IPBasedRateLimiter

func NewIPBasedRateLimiter(limit int, window time.Duration) *IPBasedRateLimiter {
	return &IPBasedRateLimiter{
		limiters: make(map[string]*FixedWindowRateLimiter),
		limit:    limit,
		window:   window,
		window2:  window.String(),
	}
}

func (iprl *IPBasedRateLimiter) Allow(ip string, pass bool, count int) IsAllow {
	iprl.mu.Lock()
	defer iprl.mu.Unlock()

	limiter, exists := iprl.limiters[ip]
	if !exists {
		limiter = NewFixedWindowRateLimiter(iprl.limit, iprl.window, iprl.window2)
		iprl.limiters[ip] = limiter
	}

	return limiter.Allow(pass, count)
}

func (iprl *IPBasedRateLimiter) allowPassCheck(ip string) {
	iprl.mu.Lock()
	defer iprl.mu.Unlock()
	iprl.limiters[ip].allowPassCheck()
}

func (rl *FixedWindowRateLimiter) allowPassCheck() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.count++ // 简单增加计数
}

func (iprls IPBasedRateLimiters) AllowPassCheck(ip string) {
	for _, limiter := range iprls {
		limiter.allowPassCheck(ip)
	}
}

type IsAllow struct {
	Allow bool
	Used  int
	Limit int
	Wind  string
}
