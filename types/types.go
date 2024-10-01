package types

import "encoding/json"

type Web3ClientRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	Id      interface{} `json:"id"`
	Method  string      `json:"method"`
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
