package test

import (
	"math/big"
	"net"
	"testing"

	"github.com/48Club/service_agent/cloudflare"
	"github.com/stretchr/testify/assert"
)

func TestIpcidr(t *testing.T) {
	assert.True(t, cloudflare.IsCloudflareIP("172.68.242.51"))
	assert.False(t, cloudflare.IsCloudflareIP("8.8.8.8"))
	_, cidr64, err := net.ParseCIDR("ffff:ffff:ffff:ffff:ffff::ffff/64")
	assert.Nil(t, err)
	assert.Equal(t, "ffff:ffff:ffff:ffff::/64", cidr64.String())
	// 计算 100000000 次耗时
	t.Run("Benchmark", func(t *testing.T) {
		firstIp := net.ParseIP("2c0f:f248::2")
		assert.True(t, firstIp != nil)
		firstIpBig := big.NewInt(0).SetBytes(firstIp.To16())

		t.Run("IsCloudflareIP", func(t *testing.T) {
			for i := 0; i < 100000000; i++ {
				firstIpBig.Add(firstIpBig, big.NewInt(1))
				ipBytes := firstIpBig.FillBytes(make([]byte, 16))
				ip := net.IP(ipBytes).String()
				cloudflare.IsCloudflareIP(ip)

			}
		})
	})

}
