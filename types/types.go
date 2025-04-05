package types

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
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

func (l LimitResponse) AddHeader(c *gin.Context) {
	c.Header("X-RateLimit-Remaining", l.Remaining.ToString())
	c.Header("X-RateLimit-Limit", l.Limit.ToString())
}

type HeaderStrs []string

func (s HeaderStrs) ToString() string {
	b, err := json.Marshal(s)
	if err != nil {
		return "[]"
	}
	return string(b)
}
