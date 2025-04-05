package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Sentry        string              `json:"sentry"`  // 哨兵节点
	DomainsHelper []string            `json:"domains"` // 域名列表
	Domains       map[string]struct{} `json:"-"`       // 域名列表, 用于快速查找
	MaxBatchQuery int                 `json:"max_batch_query"`
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
}
