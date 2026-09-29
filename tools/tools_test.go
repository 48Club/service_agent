package tools

import (
	"testing"

	"github.com/48Club/service_agent/config"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestIsValidMethodName(t *testing.T) {
	valid := []string{
		"eth_call",
		"eth_getBlockByNumber",
		"net_version",
		"txpool_content",
		"web3_clientVersion",
		"eth_simulateV1",
		"eth_call1",
		"ETH_CALL",
		"ethcall", // 下划线可以不出现
		"_",
		"", // 空方法名不含非法字符, 保持原有行为交由后端处理
	}
	for _, m := range valid {
		assert.True(t, isValidMethodName(m), "%q should be valid", m)
	}

	invalid := []string{
		"eth_call ",
		" eth_call",
		"eth-call",
		"eth.call",
		"eth/call",
		"eth_call;",
		"eth_call'",
		"eth_call\"",
		"eth_call\n",
		"eth_call\x00",
		"eth_调用",
		"eth_cäll",
		"eth__call", // 下划线多于一次
		"eth_call_",
		"_eth_call",
		"eth_signTypedData_v4",
	}
	for _, m := range invalid {
		assert.False(t, isValidMethodName(m), "%q should be invalid", m)
	}
}

func TestDecodeRequestBodyRejectsInvalidMethod(t *testing.T) {
	config.GlobalConfig.SkipLimitMethods = mapset.NewSet("eth_sendRawTransaction")

	body := []byte(`{"jsonrpc":"2.0","id":7,"method":"eth_call;drop","params":[]}`)
	resp, buildRespByAgent, batchCount, skipLimit := DecodeRequestBody("bsc-rpc.com", body)

	assert.True(t, buildRespByAgent, "invalid method must be answered by agent, not forwarded")
	assert.False(t, skipLimit)
	assert.Equal(t, 1, batchCount)
	assert.Equal(t, gin.H{
		"jsonrpc": "2.0",
		"id":      float64(7),
		"error": gin.H{
			"code":    -32601,
			"message": "the method eth_call;drop does not exist/is not available",
		},
	}, resp)
}

func TestDecodeRequestBodyForwardsValidMethod(t *testing.T) {
	config.GlobalConfig.SkipLimitMethods = mapset.NewSet("eth_sendRawTransaction")

	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`)
	resp, buildRespByAgent, batchCount, skipLimit := DecodeRequestBody("bsc-rpc.com", body)

	assert.False(t, buildRespByAgent)
	assert.Nil(t, resp)
	assert.Equal(t, 1, batchCount)
	assert.False(t, skipLimit)
}
