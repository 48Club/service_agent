package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Sentry        string              `json:"sentry"` // 哨兵节点
	Original      string              `json:"original"`
	Nginx         string              `json:"nginx"`
	DomainsHelper []string            `json:"domains"` // 域名列表
	Domains       map[string]struct{} `json:"-"`       // 域名列表, 用于快速查找
	WafKey        string              `json:"waf_key"`
	MaxBatchQuery int                 `json:"max_batch_query"`
	Database      struct {
		User     string `json:"username"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Database string `json:"database"`
	} `json:"db"`
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

func (c Config) DSN() string {
	db := c.Database
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", db.User, db.Password, db.Host, db.Port, db.Database)
}
