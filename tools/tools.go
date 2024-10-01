package tools

import (
	"encoding/json"
	"errors"

	"github.com/48Club/service_agent/types"
	"github.com/gin-gonic/gin"
)

var toSentryMethod = map[string]struct{}{
	"eth_sendRawTransaction":      {},
	"eth_sendBatchRawTransaction": {},
	"eth_get0GweiGasRemaining":    {},
}

func hasMethod(t types.Web3ClientRequests) bool {
	for _, v := range t {
		_, ok := toSentryMethod[v.Method]
		if ok {
			return true
		}
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

func DecodeRequestBody(isRpc bool, body []byte) (reqCount int, i interface{}, mustSend2Sentry bool, buildRespByAgent bool, resp interface{}, err error) {
	switch CheckJOSNType(body) {
	case 123: // {
		var web3Req types.Web3ClientRequest
		err = json.Unmarshal(body, &web3Req)
		if err == nil {
			resp, buildRespByAgent = methodWithResp[web3Req.Method]
			return 1, web3Req, hasMethod(types.Web3ClientRequests{web3Req}), buildRespByAgent, resp, nil
		}
	case 91: // [
		var web3Reqs types.Web3ClientRequests
		err = json.Unmarshal(body, &web3Reqs)
		if err == nil {
			if isRpc && len(web3Reqs) == 1 {
				resp, buildRespByAgent = methodWithResp[web3Reqs[0].Method]
			}
			return len(web3Reqs), web3Reqs, hasMethod(web3Reqs), buildRespByAgent, resp, nil
		}
	default:
		err = errors.New("invalid request")
	}

	return 1, nil, false, false, false, err
}

func buildGethResponse(i interface{}, result interface{}) interface{} {
	if _req, ok := i.(types.Web3ClientRequest); ok {
		return gin.H{
			"jsonrpc": _req.JsonRPC,
			"id":      _req.Id,
			"result":  result,
		}
	} else if _reqs, ok := i.(types.Web3ClientRequests); ok {
		return []gin.H{
			{
				"jsonrpc": _reqs[0].JsonRPC,
				"id":      _reqs[0].Id,
				"result":  result,
			},
		}

	}
	return gin.H{}
}

var (
	methodWithResp = map[string]interface{}{
		"eth_gasPrice":       ethGasPrice,
		"web3_clientVersion": web3ClientVersion,
		"eth_chainId":        ethChainId,
	}
)

const (
	ethGasPrice       = "0x3b9aca00"
	web3ClientVersion = "Geth/v1.4.11/linux-amd64/go1.22.4"
	ethChainId        = "0x38"
)

func EthResp(i, resp interface{}) interface{} {
	return buildGethResponse(i, resp)
}
