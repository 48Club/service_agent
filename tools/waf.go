package tools

import (
	"net"
	"strings"

	"github.com/48Club/service_agent/cloudflare"
	"github.com/gin-gonic/gin"
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

func CheckGinIP(c *gin.Context) (string, bool) {
	remoteIP, userIp := FormatIP(c.RemoteIP()), FormatIP(c.ClientIP())

	return userIp, cloudflare.IsCloudflareIP(remoteIP)
}
