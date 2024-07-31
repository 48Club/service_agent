package tools

import (
	"encoding/json"
	"errors"
	"math/big"
	"service_agent/types"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
)

var toSentryMethod = map[string]struct{}{
	"eth_sendRawTransaction":      {},
	"eth_sendBatchRawTransaction": {},
}

func hasMethod(t types.Web3ClientRequest) bool {
	_, ok := toSentryMethod[t.Method]
	return ok
}

func checkMethodType(body []byte) byte {
	for _, v := range body {
		if v != 32 {
			return v
		}
	}
	return 0
}

func DecodeRequestBody(isRpc bool, body []byte) (i interface{}, hasGasPrice, mustSend2Sentry bool, err error) {
	switch checkMethodType(body) {
	case 123: // {
		var web3Req types.Web3ClientRequest
		err = json.Unmarshal(body, &web3Req)
		if err != nil {
			return nil, false, false, err
		}
		return web3Req, isRpc && web3Req.Method == "eth_gasPrice", hasMethod(web3Req), nil
	case 91: // [
		var web3Reqs types.Web3ClientRequests
		err = json.Unmarshal(body, &web3Reqs)
		if err != nil {
			return nil, false, false, err
		}
		return web3Reqs, len(web3Reqs) == 1 && web3Reqs[0].Method == "eth_gasPrice", func() bool {
			for _, _web3Reqs := range web3Reqs {
				if isRpc && hasMethod(_web3Reqs) {
					return true
				}
			}
			return false
		}(), nil
	default:
		return nil, false, false, errors.New("invalid request body")
	}
}

var (
	gas1Gwei = hexutil.EncodeBig(big.NewInt(1e9))
)

func GetGasPrice(i interface{}) interface{} {
	if _req, ok := i.(types.Web3ClientRequest); ok {
		return gin.H{
			"jsonrpc": _req.JsonRPC,
			"id":      _req.Id,
			"result":  gas1Gwei,
		}
	} else if _reqs, ok := i.(types.Web3ClientRequests); ok {
		return []gin.H{
			{
				"jsonrpc": _reqs[0].JsonRPC,
				"id":      _reqs[0].Id,
				"result":  gas1Gwei,
			},
		}

	}
	return gin.H{}
}
