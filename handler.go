package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"service_agent/limit"

	"github.com/gin-gonic/gin"
)

var (
	limits limit.IPBasedRateLimiters
)

func customRecoveryMiddleware(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v, req len: %d, resp len: %d, from: %s, resp code: %d", r, c.Request.ContentLength, c.Writer.Size(), c.ClientIP(), c.Writer.Status())
		}
	}()
}

const (
	MaxRequestBodySize  = 1 << 20 / 2 // 0.5 MB
	MaxResponseBodySize = 100 << 20   // 100 MB
)

func setMaxRequestBodySize(c *gin.Context) {
	if c.Request.ContentLength > MaxRequestBodySize { // header check
		c.String(http.StatusRequestEntityTooLarge, "Request body too large")
		c.Abort()
		return
	}
	limitedReader := io.LimitReader(c.Request.Body, MaxRequestBodySize)
	body, err := io.ReadAll(limitedReader)
	if err != nil { // read error (size is bigger than MaxRequestBodySize)
		c.String(http.StatusRequestEntityTooLarge, "Request body too large")
		c.Abort()
		return
	}

	if len(body) > MaxRequestBodySize {
		c.String(http.StatusRequestEntityTooLarge, "Request body too large")
		c.Abort()
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
}

// 限流中间件
func limitMiddleware(c *gin.Context) {
	if c.Request.Host == "ipfs.48.club" { // ipfs 服务不限流
		return
	}
	ip := c.ClientIP()

	strLimit, strRemaining, tooManyRequests := []string{}, []string{}, false

	for _, limit := range limits {
		limiter := limit.Allow(ip, true)
		strLimit = append(strLimit, fmt.Sprintf("%d/%s", limiter.Limit, limiter.Wind))
		strRemaining = append(strRemaining, fmt.Sprintf("%d/%s", limiter.Limit-limiter.Used, limiter.Wind))
		if !limiter.Allow {
			tooManyRequests = true
		}
	}

	jsonRemaining, _ := json.Marshal(strRemaining)
	jsonLimit, _ := json.Marshal(strLimit)

	c.Header("X-RateLimit-Remaining", string(jsonRemaining))
	c.Header("X-RateLimit-Limit", string(jsonLimit))
	c.Header("X-Powered-By", "https://x.com/48club_official")

	if tooManyRequests {
		c.AbortWithStatus(http.StatusTooManyRequests)
		return
	}

	limits.AllowPassCheck(ip)
}

func anyHandler(c *gin.Context) {
	var body = []byte{}
	if c.Request.ContentLength != 0 {
		body, _ = io.ReadAll(c.Request.Body)
		defer c.Request.Body.Close()
	}
	if c.Request.URL.Path == "/erigon/" && c.Request.Host == "rpc-bsc.48.club" {
		proxyHandler(c, body, globalConfig.Nginx)
		return
	}
	switch c.Request.Method {
	case http.MethodPost:
		postHandler(c, body)
	case http.MethodOptions:
		c.AbortWithStatus(http.StatusOK)
	default:
		proxyHandler(c, body, globalConfig.Nginx)
	}
}

// postHandler 处理 POST 请求
func postHandler(c *gin.Context, body []byte) {
	_, ok := globalConfig.Domains[c.Request.Host]
	if !ok {
		// 不在域名列表, 直接交给 nginx 处理
		proxyHandler(c, body, globalConfig.Nginx)
		return
	}

	rpcHandler(c, body)
}

func ethGasPriceHandler(c *gin.Context, id interface{}, jsonrpc string) {
	c.JSON(http.StatusOK, gin.H{
		"jsonrpc": jsonrpc,
		"id":      id,
		"result":  "0x3b9aca00",
	})
}

// rpcHandler 处理 geth JSON-RPC 请求
func rpcHandler(c *gin.Context, body []byte) {
	var web3Req Web3ClientRequest
	err := json.Unmarshal(body, &web3Req)

	if err != nil {
		var web3Reqs Web3ClientRequests
		err = json.Unmarshal(body, &web3Reqs)
		if err == nil {
			for _, web3Req := range web3Reqs {
				if web3Req.Method == "eth_sendRawTransaction" {
					proxyHandler(c, body, globalConfig.Sentry)
					return
				}
			}

			if len(web3Reqs) == 1 && web3Reqs[0].Method == "eth_gasPrice" {
				ethGasPriceHandler(c, web3Reqs[0].Id, web3Reqs[0].JsonRPC)
				return
			}

			proxyHandler(c, body, globalConfig.Original)
			return
		}

		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	switch web3Req.Method {
	case "eth_sendRawTransaction":
		// 交易发送到哨兵节点
		proxyHandler(c, body, globalConfig.Sentry)
	case "eth_gasPrice":
		// 返回 1gwei gas price
		ethGasPriceHandler(c, web3Req.Id, web3Req.JsonRPC)
	default:
		// 普通请求发送到原始节点
		proxyHandler(c, body, globalConfig.Original)
	}
}

// proxyHandler 代理请求到目标节点
func proxyHandler(c *gin.Context, body []byte, toHost string) {
	target, _ := url.Parse(toHost)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.URL = target
		req.Host = c.Request.Host
		req.URL.Path = c.Request.URL.Path
		req.URL.RawQuery = c.Request.URL.RawQuery
		req.Header = c.Request.Header.Clone()
		req.ContentLength = int64(len(body))
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		c.AbortWithStatus(http.StatusBadGateway)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Body = http.MaxBytesReader(nil, resp.Body, MaxResponseBodySize)

		resp.Header.Del("Access-Control-Allow-Origin")

		if resp.ContentLength <= 0 {
			resp.Header.Del("Content-Length")
		}
		return nil
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}
