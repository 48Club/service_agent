package tools

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/48Club/service_agent/config"
	"github.com/48Club/service_agent/types"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

func GetRpcStatus() int {
	ec, err := ethclient.Dial(config.GlobalConfig.Sentry)
	if err != nil {
		return http.StatusInternalServerError
	}
	defer ec.Close()

	block, err := ec.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return http.StatusInternalServerError
	}
	// 区块不是 20 秒内的, 认为 RPC 不可用
	if time.Since(time.Unix(int64(block.Time), 0)) > 20*time.Second {
		return http.StatusInternalServerError
	}

	return http.StatusNoContent
}

func IsRpc(host string, d mapset.Set[string]) bool {
	// 判断 域名后缀是否包含 .48.club 或 .bsc-rpc.com
	return d.ContainsOne(host) || strings.HasSuffix(host, ".48.club") || strings.HasSuffix(host, ".bsc-rpc.com")
}

func CheckJOSNType(body []byte) byte {
	for _, v := range body {
		if v != 32 {
			return v
		}
	}
	return 0
}

var BadBatchRequest = errors.New("bad batch request")

// buildRespByAgent: 是否需要由 agent 构建响应
// resp: 由 agent 构建的响应
// batchCount: 批量请求中非 eth_sendRawTransaction 的请求数量
// sikpLimit: 是否跳过限制器
func DecodeRequestBody(host string, body []byte) (resp gin.H, buildRespByAgent bool, batchCount int, skipLimit bool) {
	batchCount = 1
	switch CheckJOSNType(body) {
	case 123: // {
		var web3Req types.Web3ClientRequest
		err := json.Unmarshal(body, &web3Req)
		if err != nil {
			return
		}

		if config.GlobalConfig.SkipLimitMethods.ContainsOne(web3Req.Method) {
			skipLimit = true
			return
		}

		var _tmp string
		switch web3Req.Method {
		case "eth_gasPrice":
			_tmp, buildRespByAgent = set1weiGasPrice(host)
		case "eth_call":
			_tmp, buildRespByAgent = decodeEthCall(web3Req.Params)
		}
		if buildRespByAgent {
			resp = buildGethResponse(web3Req, _tmp)
		}
	case 91: // [
		var web3Reqs types.Web3ClientRequests
		err := json.Unmarshal(body, &web3Reqs)
		if err != nil {
			return
		}

		reqCount := len(web3Reqs)
		txCount := 0

		for _, v := range web3Reqs {
			if config.GlobalConfig.SkipLimitMethods.ContainsOne(v.Method) {
				txCount++
			}
		}

		if txCount == reqCount {
			skipLimit = true
			return
		}

		batchCount = reqCount - txCount
	}

	return
}

func set1weiGasPrice(h string) (string, bool) {
	if h == "0.48.club" {
		return "0x1", true
	}
	return "", false
}

func buildGethResponse(i types.Web3ClientRequest, result string) gin.H {
	return gin.H{
		"jsonrpc": i.JsonRPC,
		"id":      i.Id,
		"result":  result,
	}
}

// 如果用户请求特定的方法，我们可以直接返回 0x30 作为响应
func decodeEthCall(p []any) (s string, b bool) {
	if len(p) > 2 {
		return
	}
	v, ok := p[0].(map[string]any)
	if !ok {
		return
	}

	if value, ok := v["value"].(string); !ok || hexutil.EncodeBig(big.NewInt(48)) != value {
		return
	}

	if to, ok := v["to"].(string); !ok || common.HexToAddress("0x48") != common.HexToAddress(to) {
		return
	}
	return "0x30", true
}
