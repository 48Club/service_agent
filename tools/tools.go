package tools

import (
	"encoding/json"
	"errors"

	"github.com/48Club/service_agent/types"
	"github.com/gin-gonic/gin"
)

var toSentryMethod = map[string]int{
	"eth_sendRawTransaction":      0,
	"eth_sendBatchRawTransaction": 1,
	"eth_get0GweiGasRemaining":    2,
}

func IsRpc(host string, d map[string]struct{}) bool {
	_, ok := d[host]
	if ok {
		return true
	}
	// 判断 域名后缀是否包含 .rpc.48.club
	if len(host) > 12 && host[len(host)-12:] == ".rpc.48.club" {
		return true
	}
	return false

}

func iterateEthRequestMethod(t types.Web3ClientRequests) (mustSend2Sentry bool, ethCallCount, ethSendRawTransaction int) {
	for _, v := range t {
		if v.Method == "eth_call" {
			ethCallCount++
		} else if !mustSend2Sentry {
			id, ok := toSentryMethod[v.Method]
			if ok {
				mustSend2Sentry = true
				if id < 2 {
					ethSendRawTransaction++
				}
			}
		}
	}
	return
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

func DecodeRequestBody(isRpc bool, host string, body []byte) (reqCount int, i interface{}, mustSend2Sentry bool, buildRespByAgent bool, resp interface{}, ethCallCount, ethSendRawTransactionCount int, err error) {
	switch CheckJOSNType(body) {
	case 123: // {
		var web3Req types.Web3ClientRequest
		err = json.Unmarshal(body, &web3Req)
		if err == nil {
			resp, buildRespByAgent = methodWithResp[web3Req.Method]
			mustSend2Sentry, ethCallCount, ethSendRawTransactionCount = iterateEthRequestMethod(types.Web3ClientRequests{web3Req})
			return 1, web3Req, mustSend2Sentry, buildRespByAgent, set1weiGasPrice(host, web3Req.Method, resp), ethCallCount, ethSendRawTransactionCount, nil
		}
	case 91: // [
		var web3Reqs types.Web3ClientRequests
		err = json.Unmarshal(body, &web3Reqs)
		if err == nil {
			if isRpc && len(web3Reqs) == 1 {
				resp, buildRespByAgent = methodWithResp[web3Reqs[0].Method]
				resp = set1weiGasPrice(host, web3Reqs[0].Method, resp)
			}
			mustSend2Sentry, ethCallCount, ethSendRawTransactionCount = iterateEthRequestMethod(web3Reqs)
			return len(web3Reqs), web3Reqs, mustSend2Sentry, buildRespByAgent, resp, ethCallCount, ethSendRawTransactionCount, nil
		}
	default:
		err = errors.New("invalid request")
	}

	return 0, nil, false, false, false, 0, 0, err
}

func set1weiGasPrice(h, m string, o interface{}) interface{} {
	if h == "0.48.club" && m == "eth_gasPrice" {
		return "0x1"
	}
	return o
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
