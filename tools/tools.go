package tools

import (
	"encoding/json"
	"errors"

	"github.com/48Club/service_agent/database"
	"github.com/48Club/service_agent/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	types2 "github.com/ethereum/go-ethereum/core/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

func buildGethResponse(host string, i, result interface{}) interface{} {
	switch x := result.(type) {
	case func(*gorm.DB, string) int64:
		result = x(database.Server.DB, host)
	case func(*gorm.DB, string) []string:
		result = x(database.Server.DB, host)
	}

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
		"stat_walletCount":   database.WalletCount,
		"stat_txCount":       database.TxCount,
		"stat_walletList":    database.WalletList,
	}
)

const (
	ethGasPrice       = "0x3b9aca00"
	web3ClientVersion = "Geth/v1.4.15-ec318b9c-20240919/linux-amd64/go1.22.7/48Club"
	ethChainId        = "0x38"
)

func EthResp(host string, i, resp interface{}) interface{} {
	return buildGethResponse(host, i, resp)
}

func decodeCallData(calldata types.Web3ClientRequest) []*types2.Transaction {
	id, ok := toSentryMethod[calldata.Method]
	if ok && id < 2 {
		return decodeTx(calldata.Params)
	}
	return []*types2.Transaction{}
}

func decodeTx(params []interface{}) []*types2.Transaction {
	txs := []*types2.Transaction{}
	for _, i := range params {
		hexTx, ok := i.(string)
		if !ok {
			continue
		}
		rlpTx, err := hexutil.Decode(hexTx)
		if err != nil {
			continue
		}
		tx := &types2.Transaction{}
		if err := tx.UnmarshalBinary(rlpTx); err != nil {
			continue
		}
		txs = append(txs, tx)
	}
	return txs
}

func DecodeTx(i interface{}) types.Transactions {
	if calldata, ok := i.(types.Web3ClientRequest); ok {
		return decodeCallData(calldata)
	} else if calldatas, ok := i.(types.Web3ClientRequests); ok {
		txs := []*types2.Transaction{}
		for _, calldata := range calldatas {
			txs = append(txs, decodeCallData(calldata)...)
		}
		return txs
	}
	return []*types2.Transaction{}
}
