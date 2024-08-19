package tools

import (
	"context"
	"log"
	"net"
	"strings"
	"time"

	"github.com/48Club/service_agent/config"
	"github.com/48Club/service_agent/limit"
	"github.com/cloudflare/cloudflare-go"
	mapset "github.com/deckarep/golang-set/v2"
)

var (
	channels          chan string       = make(chan string, 1)
	cloudflareAccount config.Cloudflare = config.GlobalConfig.Cloudflare
	api               *cloudflare.API
	banedIPs          mapset.Set[string] = mapset.NewSet[string]()
)

func init() {
	if cloudflareAccount.Email == "" || cloudflareAccount.Key == "" || cloudflareAccount.Account == "" {
		log.Panic("cloudflare account not set")
	}

	_api, err := cloudflare.New(cloudflareAccount.Key, cloudflareAccount.Email)
	if err != nil {
		log.Panic("cloudflare.New err:", err)
	}

	api = _api

	go doBlockIP()
}

func BlockIP(ip string) {
	channels <- ip
}

func doBlockIP() {
	tc := time.NewTicker(250 * time.Millisecond)
	for {
		<-tc.C
		ip := <-channels

		target := "ip"
		if strings.Index(ip, ":") > 0 {
			target = "ip_range"
			_, cidr64, _ := net.ParseCIDR(ip + "/64")
			ip = cidr64.String()
		}

		if banedIPs.Contains(ip) {
			continue // skip duplicate ip
		}
		banedIPs.Add(ip)

	BEGIN:
		resp, err := api.CreateAccountAccessRule(context.Background(), cloudflareAccount.Account, cloudflare.AccessRule{
			Mode:  "block",
			Notes: "batch request ip",
			Configuration: cloudflare.AccessRuleConfiguration{
				Target: target,
				Value:  ip,
			},
			Scope: cloudflare.AccessRuleScope{
				Type: "account",
			},
		})

		if err != nil {
			log.Printf("api.CreateZoneAccessRule err:%+v", err)
			if strings.Contains(err.Error(), "\\\"code\\\": 10009") || strings.Contains(err.Error(), "firewallaccessrules.api.duplicate_of_existing") {
				continue
				// skip duplicate rule
			}
			goto BEGIN
		}
		if !resp.Success {
			log.Printf("api.CreateZoneAccessRule resp:%+v", resp)
			goto BEGIN
		}
		log.Println("success post to cloudflare, ip:", ip)
		limit.Limits.Prune(ip)
	}
}
