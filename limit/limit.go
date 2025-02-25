package limit

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/48Club/service_agent/redis"
	redis9 "github.com/redis/go-redis/v9"
)

var (
	Limits  IPBasedRateLimiters
	redisDB = redis.New(0)
)

func (iprls IPBasedRateLimiters) Prune(ip string) {
	for _, rl := range iprls {
		rl.mu.Lock()
		delete(rl.limiters, ip)
		rl.mu.Unlock()
	}
}

func init() {
	Limits = IPBasedRateLimiters{
		NewIPBasedRateLimiter(lhLimit{true: 80, false: 48}, time.Second*5, "5s"),         // [9.6|16]qps
		NewIPBasedRateLimiter(lhLimit{true: 720, false: 432}, time.Minute, "1m"),         // [7.2|12]qps
		NewIPBasedRateLimiter(lhLimit{true: 345600, false: 207360}, time.Hour*24, "24h"), // [2.4|4]qps
	}
}

type lhLimit map[bool]int

// 修改为 FixedWindowRateLimiter
type FixedWindowRateLimiter struct {
	mu        sync.Mutex
	count     int           // 当前窗口内的请求计数
	lastReset time.Time     // 上次窗口重置时间
	limit     lhLimit       // 限制配额
	window    time.Duration // 窗口大小
	window2   string        // 窗口标识
}

func NewFixedWindowRateLimiter(_lhLimit lhLimit, window time.Duration, window2 string) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{
		limit:     _lhLimit,
		window:    window,
		window2:   window2,
		lastReset: time.Now(),
	}
}

func (rl *FixedWindowRateLimiter) Allow(pass bool, count int, seriveStat bool) IsAllow {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	// 检查是否需要重置窗口
	if now.Sub(rl.lastReset) >= rl.window {
		rl.count = 0
		rl.lastReset = now
	}

	used := rl.count

	if used+count < rl.limit[seriveStat] {
		if pass {
			return IsAllow{true, used, rl.limit[seriveStat], rl.window2}
		}
		// 增加计数
		rl.count += count
		return IsAllow{true, used + count, rl.limit[seriveStat], rl.window2}
	}

	return IsAllow{false, used, rl.limit[seriveStat], rl.window2}
}

type IPBasedRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*FixedWindowRateLimiter // 修改为 FixedWindowRateLimiter
	limit    lhLimit
	window   time.Duration
	window2  string
}

type IPBasedRateLimiters []*IPBasedRateLimiter

func NewIPBasedRateLimiter(_lhLimit lhLimit, window time.Duration, window2 string) *IPBasedRateLimiter {
	return &IPBasedRateLimiter{
		limiters: make(map[string]*FixedWindowRateLimiter),
		limit:    _lhLimit,
		window:   window,
		window2:  window2,
	}
}

func (iprl *IPBasedRateLimiter) Allow(ip string, pass bool, count int, seriveStat bool) IsAllow {
	iprl.mu.Lock()
	defer iprl.mu.Unlock()

	limiter, exists := iprl.limiters[ip]
	if !exists {
		limiter = NewFixedWindowRateLimiter(iprl.limit, iprl.window, iprl.window2)
		iprl.limiters[ip] = limiter
	}

	return limiter.Allow(pass, count, seriveStat)
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

// 修改 redisSave 结构
type redisSave struct {
	IP        string `json:"ip"`
	LastReset int64  `json:"lastReset"` // 窗口重置时间（Unix 时间戳）
	Count     int    `json:"count"`     // 当前窗口计数
}

func (iprls IPBasedRateLimiters) SaveCache() error {
	for _, iprl := range iprls {
		iprl.mu.Lock()
		defer iprl.mu.Unlock()
		var redisSaves = []redisSave{}
		for ip, rl := range iprl.limiters {
			rl.mu.Lock()
			redisSaves = append(redisSaves, redisSave{
				IP:        ip,
				LastReset: rl.lastReset.Unix(),
				Count:     rl.count,
			})
			rl.mu.Unlock()
		}

		b, err := json.Marshal(redisSaves)
		if err != nil {
			return err
		}
		if err := redisDB.SaveCache(fmt.Sprintf("rl_%s", iprl.window2), string(b), iprl.window); err != nil {
			return err
		}
	}
	return nil
}

func (iprls IPBasedRateLimiters) LoadFromCache() error {
	for _, iprl := range iprls {
		key := fmt.Sprintf("rl_%s", iprl.window2)
		b, err := redisDB.GetCache(key)
		if err != nil {
			if err == redis9.Nil {
				continue
			}
			return err
		}
		var redisSaves = []redisSave{}
		if err := json.Unmarshal([]byte(b), &redisSaves); err != nil {
			return err
		}
		for _, _redisSave := range redisSaves {
			rl := NewFixedWindowRateLimiter(iprl.limit, iprl.window, iprl.window2)
			rl.lastReset = time.Unix(_redisSave.LastReset, 0)
			rl.count = _redisSave.Count
			// 检查窗口是否过期
			if time.Since(rl.lastReset) >= rl.window {
				rl.count = 0
				rl.lastReset = time.Now()
			}
			iprl.mu.Lock()
			iprl.limiters[_redisSave.IP] = rl
			iprl.mu.Unlock()
		}
		_ = redisDB.Del(key)
	}
	return nil
}
