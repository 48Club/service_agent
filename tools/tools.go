package tools

import (
	"encoding/json"
	"errors"

	"github.com/48Club/service_agent/types"
	"github.com/gin-gonic/gin"
)

func CheckJOSNType(body []byte) byte {
	for _, v := range body {
		if v != 32 {
			return v
		}
	}
	return 0
}

var BadBatchRequest = errors.New("bad batch request")

func DecodeRequestBody(isRpc bool, body []byte) (reqCount int, i interface{}, mustSend2Sentry bool, sumGas uint64, buildRespByAgent bool, resp interface{}, err error) {
	switch CheckJOSNType(body) {
	case 123: // {
		var web3Req types.Web3ClientRequest
		err = json.Unmarshal(body, &web3Req)
		if err == nil {
			resp, buildRespByAgent = methodWithResp[web3Req.Method]
			return 1, web3Req, web3Req.HasSentryMethod(), web3Req.CheckCallGas(), buildRespByAgent, resp, nil
		}
	case 91: // [
		var web3Reqs types.Web3ClientRequests
		err = json.Unmarshal(body, &web3Reqs)
		if err == nil {
			if isRpc && len(web3Reqs) == 1 {
				resp, buildRespByAgent = methodWithResp[web3Reqs[0].Method]
			}
			return len(web3Reqs), web3Reqs, web3Reqs.HasSentryMethod(), web3Reqs.CheckCallGas(), buildRespByAgent, resp, nil
		}
	default:
		err = errors.New("invalid request")
	}

	return 1, nil, false, 0, false, false, err
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
