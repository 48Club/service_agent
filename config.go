package main

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
}

var globalConfig = Config{}

func loadConfig(f string) {
	file, err := os.ReadFile(f)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(file, &globalConfig)
	if err != nil {
		panic(err)
	}

	globalConfig.Domains = map[string]struct{}{}
	for _, domain := range globalConfig.DomainsHelper {
		globalConfig.Domains[domain] = struct{}{}
	}
}
