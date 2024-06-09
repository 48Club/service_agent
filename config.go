package main

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Upstream struct {
	Fails     uint   `json:"-"` // 失败次数
	Conn      uint64 `json:"-"` // 连接数
	AliveTime int64  `json:"-"` // 上次在线时间
	mu        sync.Mutex
}

// 递增失败次数
func (u *Upstream) AddFails() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Fails++
}

// 重置失败次数
func (u *Upstream) ResetFails() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Fails = 0
}

// 递增连接数;
//
// 更新上次在线时间
func (u *Upstream) AddConn() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Conn++
	u.AliveTime = time.Now().Unix()
}

type Config struct {
	Sentry         string               `json:"sentry"`   // 哨兵节点
	DomainsHelper  []string             `json:"domains"`  // 域名列表
	Domains        map[string]struct{}  `json:"-"`        // 域名列表
	OriginalHelper []string             `json:"original"` // 原始节点
	Original       map[string]*Upstream `json:"-"`        // 原始节点
	TimeOut        uint                 `json:"-"`        // 超时时间
	MaxFails       uint                 `json:"-"`        // 最大失败次数
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

	for _, domain := range globalConfig.DomainsHelper {
		globalConfig.Domains[domain] = struct{}{}
	}

	for _, original := range globalConfig.OriginalHelper {
		globalConfig.Original[original] = &Upstream{
			Fails:     0,
			Conn:      0,
			AliveTime: time.Now().Unix(),
		}
	}
}
