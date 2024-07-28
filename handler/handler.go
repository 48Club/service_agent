package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"service_agent/config"
	"service_agent/limit"
	"service_agent/tools"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gin-gonic/gin"
)

var normalRequestStatus mapset.Set[int]

func init() {
	normalRequestStatus = mapset.NewSet[int]()
	normalRequestStatus.Add(http.StatusOK)
	normalRequestStatus.Add(http.StatusTooManyRequests)
	normalRequestStatus.Add(http.StatusNoContent)
}

func CustomLoggerMiddleware(c *gin.Context) {
	defer func() {
		statusCode := c.Writer.Status()
		if normalRequestStatus.ContainsOne(statusCode) {
			return
		}
		if c.Request.Header.Get("Upgrade") == "websocket" && statusCode == http.StatusBadRequest {
			return // websocket 请求不记录 400 错误
		}
		gin.LoggerWithConfig(gin.LoggerConfig{Formatter: func(param gin.LogFormatterParams) string {
			var statusColor, methodColor, resetColor string
			if param.IsOutputColor() {
				statusColor = param.StatusCodeColor()
				methodColor = param.MethodColor()
				resetColor = param.ResetColor()
			}

			if param.Latency > time.Minute {
				param.Latency = param.Latency.Truncate(time.Second)
			}
			return fmt.Sprintf("[GIN] %v | %s(%s%3d%s) | %13v | %15s |%s %-7s %s %#v\n%s",
				param.TimeStamp.Format("2006/01/02 15:04:05"),
				param.Request.Host,
				statusColor, param.StatusCode, resetColor,
				param.Latency,
				param.ClientIP,
				methodColor, param.Method, resetColor,
				param.Path,
				param.ErrorMessage,
			)
		}})(c)
	}()
	c.Next()
}

func CustomRecoveryMiddleware(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v, req len: %d, resp len: %d, from: %s, resp code: %d", r, c.Request.ContentLength, c.Writer.Size(), c.ClientIP(), c.Writer.Status())
		}
	}()
	c.Next() // 继续处理其他 middleware 与 handler, 最后执行 defer
}

const (
	MaxRequestBodySize  = 1 << 20 / 2 // 0.5 MB
	MaxResponseBodySize = 100 << 20   // 100 MB
)

func SetMaxRequestBodySize(c *gin.Context) {
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
func LimitMiddleware(c *gin.Context) {
	if c.Request.Header.Get("Upgrade") == "websocket" && c.Request.Method == http.MethodGet {
		return // websocket 不在此处限流
	}
	if c.Request.Host == "ipfs.48.club" { // ipfs 服务不限流
		return
	}

	ip := c.ClientIP()
	jsonLimit, jsonRemaining, tooManyRequests := LimitMiddleware2(ip)

	c.Header("X-RateLimit-Remaining", jsonRemaining)
	c.Header("X-RateLimit-Limit", jsonLimit)
	c.Header("X-Powered-By", "https://x.com/48club_official")
	if tooManyRequests {
		c.AbortWithStatus(http.StatusTooManyRequests)
		return
	}

	limit.Limits.AllowPassCheck(ip)
}

func LimitMiddleware2(ip string) (string, string, bool) {
	strLimit, strRemaining, tooManyRequests := []string{}, []string{}, false

	for _, limit := range limit.Limits {
		limiter := limit.Allow(ip, true)
		strLimit = append(strLimit, fmt.Sprintf("%d/%s", limiter.Limit, limiter.Wind))
		strRemaining = append(strRemaining, fmt.Sprintf("%d/%s", limiter.Limit-limiter.Used, limiter.Wind))
		if !limiter.Allow {
			tooManyRequests = true
		}
	}

	bRemaining, _ := json.Marshal(strRemaining)
	bLimit, _ := json.Marshal(strLimit)
	return string(bLimit), string(bRemaining), tooManyRequests
}

func AnyHandler(c *gin.Context) {

	var body = []byte{}
	if c.Request.ContentLength != 0 {
		body, _ = io.ReadAll(c.Request.Body)
		defer c.Request.Body.Close()
	}

	switch c.Request.Method {
	case http.MethodPost:
		if c.Request.Host == "rpc-bsc.48.club" && c.Request.URL.Path == "/erigon/" {
			proxyHandler(c, body, config.GlobalConfig.Nginx)
			return
		}
		postHandler(c, body)
	case http.MethodOptions:
		c.AbortWithStatus(http.StatusOK)
	case http.MethodGet:
		if c.Request.URL.Path == "/ws/" && c.Request.Header.Get("Upgrade") == "websocket" {
			if c.Request.Host == "puissant-bsc.48.club" {
				handleWebSocket(c, config.GlobalConfig.Puissant, false)
				return
			} else if _, ok := config.GlobalConfig.Domains[c.Request.Host]; ok {
				handleWebSocket(c, fmt.Sprintf("ws://%s", strings.Split(config.GlobalConfig.Original, "://")[1]), true)
				return
			}
		}
		fallthrough
	default:
		proxyHandler(c, body, config.GlobalConfig.Nginx)
	}
}

// postHandler 处理 POST 请求
func postHandler(c *gin.Context, body []byte) {
	_, ok := config.GlobalConfig.Domains[c.Request.Host]
	if !ok {
		// 不在域名列表, 直接交给 nginx 处理
		proxyHandler(c, body, config.GlobalConfig.Nginx)
		return
	}

	rpcHandler(c, body)
}

func ethGasPriceHandler(c *gin.Context, i interface{}) {
	c.JSON(http.StatusOK, tools.GetGasPrice(i))
}

// rpcHandler 处理 geth JSON-RPC 请求
func rpcHandler(c *gin.Context, body []byte) {

	web3Reqi, hasGasPrice, hasSendRawTransaction, err := tools.DecodeRequestBody(body)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if hasSendRawTransaction {
		proxyHandler(c, body, config.GlobalConfig.Sentry)
	} else if hasGasPrice {
		ethGasPriceHandler(c, web3Reqi)
	} else {
		proxyHandler(c, body, config.GlobalConfig.Original)
	}

}

// proxyHandler 代理请求到目标节点
func proxyHandler(c *gin.Context, body []byte, toHost string) {
	if c.Request.Header.Get("Upgrade") == "websocket" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

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
