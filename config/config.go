package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/48Club/service_agent/limit"
)

type Config struct {
	Sentry              string                       `json:"sentry"`  // 哨兵节点
	DomainsHelper       []string                     `json:"domains"` // 域名列表
	Domains             map[string]struct{}          `json:"-"`       // 域名列表, 用于快速查找
	ExceptionLimiter    []exceptionLimiter           `json:"exception_limiter"`
	ExceptionLimiterMap map[string]*exceptionLimiter `json:"-"` // 异常限制器, 用于快速查找
	MaxBatchQuery       int                          `json:"max_batch_query"`
}

type exceptionLimiter struct {
	Domain string                    `json:"domain"`
	Window time.Duration             `json:"window"`
	Limit  int                       `json:"limit"`
	XToken string                    `json:"x-48-token"`
	Limter limit.IPBasedRateLimiters `json:"-"`
}

var GlobalConfig = Config{}

func init() {
	file, err := os.ReadFile("config.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(file, &GlobalConfig)
	if err != nil {
		panic(err)
	}

	GlobalConfig.Domains = map[string]struct{}{}
	for _, domain := range GlobalConfig.DomainsHelper {
		GlobalConfig.Domains[domain] = struct{}{}
	}

	GlobalConfig.ExceptionLimiterMap = map[string]*exceptionLimiter{}
	for _, exception := range GlobalConfig.ExceptionLimiter {
		exception.Limter = limit.IPBasedRateLimiters{limit.NewIPBasedRateLimiter(exception.Limit, exception.Window*time.Second)}
		GlobalConfig.ExceptionLimiterMap[exception.Domain] = &exception
	}
}
