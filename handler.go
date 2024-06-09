package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// 白名单域名中间件
func domainMiddleware(c *gin.Context) {
	_, ok := globalConfig.Domains[c.Request.Host]
	if !ok {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
}

// postHandler 处理 POST 请求
func postHandler(c *gin.Context) {
	switch c.Request.Host {
	default:
		rpcHandler(c)
	}

}

// rpcHandler 处理 geth JSON-RPC 请求
func rpcHandler(c *gin.Context) {
	web3Req := Web3ClientRequest{}

	body, _ := io.ReadAll(c.Request.Body)
	defer c.Request.Body.Close()

	err := json.Unmarshal(body, &web3Req)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	switch web3Req.Method {
	case "eth_sendRawTransaction":
		// 交易发送到哨兵节点
		proxyHandler(c, body, false)
	case "eth_gasPrice":
		// 返回 1gwei gas price
		c.JSON(http.StatusOK, gin.H{
			"jsonrpc": "2.0",
			"id":      web3Req.Id,
			"result":  "0x3b9aca00",
		})
	default:
		// 普通请求发送到原始节点
		proxyHandler(c, body, true)

	}
}

// proxyHandler 代理请求到目标节点
func proxyHandler(c *gin.Context, body []byte, isOriginal bool) {
	// target 为空时，发送到原始节点, 使用 least_conn 策略, 保证负载均衡
	// isOriginal := len(targets) == 0
	var host string = globalConfig.Sentry
	if isOriginal {
		// 选择连接数最少的节点
		var conn uint64 = 18_446_744_073_709_551_615 // ^uint64(0)
		for k, v := range globalConfig.Original {
			if v.Fails > globalConfig.MaxFails {
				continue
			}
			if v.Conn < conn {
				conn = v.Conn
				host = k
			}
		}
		if _, ok := globalConfig.Original[host]; ok {
			globalConfig.Original[host].AddConn() // 连接数+1
		}
	}

	target, _ := url.Parse(host)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.URL = target
		req.Host = target.Host
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.Header.Set("Content-Type", c.ContentType())
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if isOriginal {
			if _, ok := globalConfig.Original[host]; ok {
				globalConfig.Original[host].AddFails() // 失败次数+1
			}
		}
		c.AbortWithStatus(http.StatusBadGateway)
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}
