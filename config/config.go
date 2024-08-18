package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Sentry        string              `json:"sentry"` // 哨兵节点
	Original      string              `json:"original"`
	Nginx         string              `json:"nginx"`
	DomainsHelper []string            `json:"domains"` // 域名列表
	Domains       map[string]struct{} `json:"-"`       // 域名列表, 用于快速查找
	Cloudflare    Cloudflare          `json:"cloudflare"`
}
type Cloudflare struct {
	Email   string `json:"email"`
	Key     string `json:"key"`
	Account string `json:"account"`
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
