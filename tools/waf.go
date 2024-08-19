package tools

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/48Club/service_agent/config"
	"github.com/48Club/service_agent/limit"
	mapset "github.com/deckarep/golang-set/v2"
)

var (
	channels chan string        = make(chan string, 1)
	banedIPs mapset.Set[string] = mapset.NewSet[string]()
)

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

		if strings.Index(ip, ":") > 0 {
			_, cidr64, _ := net.ParseCIDR(ip + "/64")
			ip = cidr64.String()
		}

		if banedIPs.Contains(ip) {
			continue // skip duplicate ip
		}
		banedIPs.Add(ip)

		data, _ := json.Marshal(IPWaf{IP: ip})
		req, _ := http.NewRequest(http.MethodPost, "https://ip-waf-helper.48.club/", bytes.NewBuffer(data))
		doReq(req, nil)

		log.Println("success post to waf, ip:", ip)
		limit.Limits.Prune(ip)
	}
}

type IPWaf struct { // https://github.com/48Club/ip-waf-helper/blob/main/types/tables.go
	IP string `json:"ip"`
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
	var ips []IPWaf
	req, _ := http.NewRequest(http.MethodGet, "https://ip-waf-helper.48.club/", nil)
	doReq(req, &ips)

	for _, v := range ips {
		banedIPs.Add(v.IP)
	}
	log.Printf("init baned ips, total: %d", len(ips))
}
