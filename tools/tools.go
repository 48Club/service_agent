package tools

import (
	"encoding/json"
	"service_agent/types"

	"github.com/gin-gonic/gin"
)

func DecodeRequestBody(body []byte) (i interface{}, hasGasPrice, hasSendRawTransaction bool, err error) {
	var web3Req types.Web3ClientRequest
	err = json.Unmarshal(body, &web3Req)

	if err != nil {
		var web3Reqs types.Web3ClientRequests
		err = json.Unmarshal(body, &web3Reqs)
		if err == nil {
			return web3Reqs, len(web3Reqs) == 1 && web3Reqs[0].Method == "eth_gasPrice", func() bool {
				for _, _web3Reqs := range web3Reqs {
					if _web3Reqs.Method == "eth_sendRawTransaction" {
						return true
					}
				}
				return false
			}(), nil

		}
		return nil, false, false, err
	}
	return web3Req, web3Req.Method == "eth_gasPrice", web3Req.Method == "eth_sendRawTransaction", nil
}

func GetGasPrice(i interface{}) gin.H {
	if _req, ok := i.(types.Web3ClientRequest); ok {
		return gin.H{
			"jsonrpc": _req.JsonRPC,
			"id":      _req.Id,
			"result":  "0x3b9aca00",
		}
	} else if _reqs, ok := i.(types.Web3ClientRequests); ok {
		return gin.H{
			"jsonrpc": _reqs[0].JsonRPC,
			"id":      _reqs[0].Id,
			"result":  "0x3b9aca00",
		}
	}
	return gin.H{}
}
