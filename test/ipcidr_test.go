package test

import (
	"net"
	"testing"

	"github.com/48Club/service_agent/cloudflare"
	"github.com/stretchr/testify/assert"
)

func TestIpcidr(t *testing.T) {
	assert.True(t, cloudflare.IsCloudflareIP("172.68.242.51"))
	assert.False(t, cloudflare.IsCloudflareIP("8.8.8.8"))
	_, cidr64, _ := net.ParseCIDR("ffff:ffff:ffff:ffff:ffff::ffff/64")
	assert.Equal(t, "ffff:ffff:ffff:ffff::/64", cidr64.String())
	// 计算 100000000 次耗时
	t.Run("Benchmark", func(t *testing.T) {
		t.Run("IsCloudflareIP", func(t *testing.T) {
			for i := 0; i < 100000000; i++ {
				cloudflare.IsCloudflareIP("2c0f:f248::2")
			}
		})
	})

}
