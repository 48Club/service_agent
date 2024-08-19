package cloudflare

import (
	_ "embed"
	"net"
	"strings"
)

var (
	//go:embed ips-v4
	_ipsV4 []byte
	//go:embed ips-v6
	_ipsV6 []byte

	_ips []*net.IPNet
)

func init() {
	ips := strings.Split(string(_ipsV4), "\n")
	ips = append(ips, strings.Split(string(_ipsV6), "\n")...)
	for _, v := range ips {
		_, ipNet, err := net.ParseCIDR(v)
		if err != nil {
			continue
		}
		_ips = append(_ips, ipNet)
	}
}

func IsCloudflareIP(ip string) bool {
	ipAddr := net.ParseIP(ip)
	for _, v := range _ips {
		if v.Contains(ipAddr) {
			return true
		}
	}
	return false
}
