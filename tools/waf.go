package tools

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	types2 "github.com/48Club/ip-waf-helper/types"
	"github.com/48Club/service_agent/cloudflare"
	"github.com/48Club/service_agent/config"
	"github.com/48Club/service_agent/limit"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gin-gonic/gin"
)

var (
	channels chan string        = make(chan string, 1)
	banedIPs mapset.Set[string] = mapset.NewSet[string]()
)

func FormatIP(s string) string { // check ipv4, or format ipv6 to cidr64
	if strings.Index(s, ":") > 0 {
		_, cidr64, err := net.ParseCIDR(s + "/64") // format ipv6 to cidr64, if bad ip, return empty string and toBlockIP
		if err != nil {
			return ""
		}
		return cidr64.String()
	}

	ip := net.ParseIP(s) // bad ip will be nil, return empty string and toBlockIP
	if ip == nil || ip.String() != s {
		return ""
	}

	return s
}

func CheckGinIP(c *gin.Context) string {
	remoteIP, userIp := FormatIP(c.RemoteIP()), FormatIP(c.ClientIP())
	if IsBanedIP(remoteIP) || IsBanedIP(userIp) {
		return ""
	}

	if !cloudflare.IsCloudflareIP(remoteIP) {
		go BlockIP(remoteIP)
		return ""
	}

	return userIp
}

func IsBanedIP(ip string) bool {
	return banedIPs.Contains(ip)
}

func init() {
	go doBlockIP()
	go initBanedIPs()
}

func BlockIP(ip string) {
	channels <- ip
}

func doBlockIP() {
	tc := time.NewTicker(250 * time.Millisecond)
	for {
		<-tc.C
		ip := <-channels

		if banedIPs.Contains(ip) {
			continue // skip duplicate ip
		}
		banedIPs.Add(ip)

		var toAdd = types2.IPWaf{IP: ip}
		data, _ := json.Marshal(toAdd)
		req, _ := http.NewRequest(http.MethodPost, "https://ip-waf-helper.48.club/", bytes.NewBuffer(data))
		doReq(req, &toAdd)

		log.Printf("success post to waf, ip: %s, total in db: %d.", toAdd.IP, toAdd.ID)
		limit.Limits.Prune(ip)
	}
}

func doReq(req *http.Request, i interface{}) {
	req.Header.Set("Authorization", config.GlobalConfig.WafKey)
	req.Header.Set("Content-Type", "application/json")
	for {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("api.doReq err:%+v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Printf("api.doReq status code:%d", resp.StatusCode)
			time.Sleep(1 * time.Second)
			continue
		}

		defer resp.Body.Close()
		if i != nil {
			_ = json.NewDecoder(resp.Body).Decode(i)
		}
		return
	}
}

func initBanedIPs() {
	var ips types2.AllIPs
	req, _ := http.NewRequest(http.MethodGet, "https://ip-waf-helper.48.club/", nil)
	doReq(req, &ips)

	banedIPs.Append(ips...)
	log.Printf("init baned ips, total: %d", len(ips))
}
