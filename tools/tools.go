package tools

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/48Club/service_agent/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
)

func IsRpc(host string, d map[string]struct{}) bool {
	_, ok := d[host]
	if ok {
		return true
	}
	// 判断 域名后缀是否包含 .rpc.48.club
	if strings.HasSuffix(host, ".rpc.48.club") {
		return true
	}
	return false

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

func DecodeRequestBody(host string, body []byte) (resp gin.H, buildRespByAgent bool, batchCount int) {
	batchCount = 1
	switch CheckJOSNType(body) {
	case 123: // {
		var web3Req types.Web3ClientRequest
		err := json.Unmarshal(body, &web3Req)
		if err != nil {
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
		batchCount = len(web3Reqs)
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
func decodeEthCall(p []interface{}) (s string, b bool) {
	if len(p) > 2 {
		return
	}
	v, ok := p[0].(map[string]interface{})
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
