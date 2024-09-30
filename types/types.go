package types

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Web3ClientRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Id      interface{}   `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type Web3ClientRequests []Web3ClientRequest

type LimitResponse struct {
	Limit     HeaderStrs
	Remaining HeaderStrs
}

type HeaderStrs []string

func (s HeaderStrs) ToString() string {
	b, err := json.Marshal(s)
	if err != nil {
		return "[]"
	}
	return string(b)
}

var toSentryMethod = map[string]struct{}{
	"eth_sendRawTransaction":      {},
	"eth_sendBatchRawTransaction": {},
	"eth_get0GweiGasRemaining":    {},
}

func (req Web3ClientRequest) HasSentryMethod() bool {
	_, ok := toSentryMethod[req.Method]
	return ok
}

func (reqs Web3ClientRequests) HasSentryMethod() bool {
	for _, req := range reqs {
		if req.HasSentryMethod() {
			return true
		}
	}
	return false
}

var detectionGasLimit = map[string]struct{}{
	"eth_call":        {},
	"eth_estimateGas": {},
}

func (req Web3ClientRequest) CheckCallGas() uint64 {
	if _, ok := detectionGasLimit[req.Method]; !ok {
		return 0
	}
	sumGas := uint64(0)
	for _, param := range req.Params {
		callParams, ok := param.(map[string]interface{})
		if !ok {
			continue
		}
		if gas, ok := callParams["gas"]; ok {
			gas, ok := gas.(string)

			if !ok {
				continue
			}
			bgas, err := hexutil.DecodeBig(gas)
			if err != nil {
				continue
			}
			sumGas += bgas.Uint64()
		}
	}
	return sumGas
}

func (reqs Web3ClientRequests) CheckCallGas() uint64 {
	sumGas := uint64(0)
	for _, req := range reqs {
		sumGas += req.CheckCallGas()
	}
	return sumGas
}
