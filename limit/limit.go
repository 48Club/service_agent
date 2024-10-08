package limit

import (
	"container/list"
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
		NewIPBasedRateLimiter(80, time.Second*5, "5s"),     // 16qps
		NewIPBasedRateLimiter(720, time.Minute, "1m"),      // 12qps
		NewIPBasedRateLimiter(28800, time.Hour, "1h"),      // 8qps
		NewIPBasedRateLimiter(345600, time.Hour*24, "24h"), // 4qps
	}
}

type SlidingWindowRateLimiter struct {
	mu         sync.Mutex
	timestamps *list.List
	limit      int
	window     time.Duration
	window2    string
}

func NewSlidingWindowRateLimiter(limit int, window time.Duration, window2 string) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		limit:      limit,
		window:     window,
		window2:    window2,
		timestamps: list.New(),
	}
}

func (rl *SlidingWindowRateLimiter) Allow(pass bool, count int) IsAllow {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for rl.timestamps.Len() > 0 {
		front := rl.timestamps.Front()
		if now.Sub(front.Value.(time.Time)) < rl.window {
			break
		}
		rl.timestamps.Remove(front)
	}

	used := rl.timestamps.Len()

	if used+count < rl.limit {
		if pass {
			return IsAllow{true, used, rl.limit, rl.window2}
		}
		// add request
		for i := 0; i < count; i++ {
			rl.timestamps.PushBack(now)
		}
		return IsAllow{true, used + count, rl.limit, rl.window2}
	}

	return IsAllow{false, used, rl.limit, rl.window2}
}

type IPBasedRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*SlidingWindowRateLimiter
	limit    int
	window   time.Duration
	window2  string
}

type IPBasedRateLimiters []*IPBasedRateLimiter

func NewIPBasedRateLimiter(limit int, window time.Duration, window2 string) *IPBasedRateLimiter {
	return &IPBasedRateLimiter{
		limiters: make(map[string]*SlidingWindowRateLimiter),
		limit:    limit,
		window:   window,
		window2:  window2,
	}
}

// pass: true means only check, false means check and add request
//
// count: if pass is false, count is the number of requests to add
func (iprl *IPBasedRateLimiter) Allow(ip string, pass bool, count int) IsAllow {
	iprl.mu.Lock()
	defer iprl.mu.Unlock()

	limiter, exists := iprl.limiters[ip]
	if !exists {
		limiter = NewSlidingWindowRateLimiter(iprl.limit, iprl.window, iprl.window2)
		iprl.limiters[ip] = limiter
	}

	return limiter.Allow(pass, count)
}

func (iprl *IPBasedRateLimiter) allowPassCheck(ip string) {
	iprl.mu.Lock()
	defer iprl.mu.Unlock()
	iprl.limiters[ip].allowPassCheck()
}

func (rl *SlidingWindowRateLimiter) allowPassCheck() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.timestamps.PushBack(time.Now())
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

func (rl *SlidingWindowRateLimiter) list2Slice() (s []int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for e := rl.timestamps.Front(); e != nil; e = e.Next() {
		t := e.Value.(time.Time)
		if t.Add(rl.window).Before(time.Now()) {
			continue
		}
		s = append(s, t.Unix())
	}
	return
}

type redisSave struct {
	IP         string  `json:"ip"`
	Timestamps []int64 `json:"timestamps"`
}

func (iprls IPBasedRateLimiters) SaveCache() error {
	for _, iprl := range iprls {
		iprl.mu.Lock()
		defer iprl.mu.Unlock()
		var redisSaves = []redisSave{}
		for ip, rl := range iprl.limiters {
			redisSaves = append(redisSaves, redisSave{
				IP:         ip,
				Timestamps: rl.list2Slice(),
			})
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
			rl := NewSlidingWindowRateLimiter(iprl.limit, iprl.window, iprl.window2)
			for _, _t := range _redisSave.Timestamps {
				t := time.Unix(_t, 0)
				if t.Add(iprl.window).Before(time.Now()) {
					continue
				}
				rl.timestamps.PushBack(t)
			}
			iprl.limiters[_redisSave.IP] = rl
		}
		_ = redisDB.Del(key)
	}
	return nil
}
