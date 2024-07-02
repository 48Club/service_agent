package main

type Web3ClientRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	Id      interface{} `json:"id"`
	Method  string      `json:"method"`
}

type Web3ClientRequests []Web3ClientRequest
