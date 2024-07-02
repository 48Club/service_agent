package cloudflare

import (
	_ "embed"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	//go:embed ips-v4
	_ipsV4 []byte
	//go:embed ips-v6
	_ipsV6 []byte

	ips []string
)

func SetRemoteAddr(g *gin.Engine) {
	g.TrustedPlatform = gin.PlatformCloudflare
	if err := g.SetTrustedProxies(ips); err != nil {
		panic(err)
	}
}

func init() {
	ips = append(ips, strings.Split(string(_ipsV4), "\n")...)
	ips = append(ips, strings.Split(string(_ipsV6), "\n")...)
}
