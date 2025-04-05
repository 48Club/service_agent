package handler

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/48Club/service_agent/config"
	"github.com/48Club/service_agent/limit"
	"github.com/48Club/service_agent/tools"
	"github.com/48Club/service_agent/types"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gin-gonic/gin"
)

var (
	normalRequestStatus = mapset.NewSet[int](http.StatusOK, http.StatusNoContent, http.StatusTooManyRequests, http.StatusUnprocessableEntity)
)

func CustomLoggerMiddleware(c *gin.Context) {
	defer func() {
		statusCode := c.Writer.Status()
		if normalRequestStatus.ContainsOne(statusCode) {
			return
		}

		if c.IsWebsocket() && statusCode == http.StatusBadRequest {
			return
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
			log.Printf("Recovered from panic: %v, req len: %d, resp len: %d, from: %s, resp code: %d", r, c.Request.ContentLength, c.Writer.Size(), c.GetString("ip"), c.Writer.Status())
		}
	}()
	c.Next()
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

func LimitMiddleware(c *gin.Context) {
	userIP, fromCDN := tools.CheckGinIP(c)
	if !fromCDN {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	c.Set("ip", userIP)

	if c.IsWebsocket() && c.Request.Method == http.MethodGet {
		return
	}

	var limitHeader = types.LimitResponse{}
	tooManyRequests := LimitMiddleware2(userIP, true, 1, &limitHeader)

	limitHeader.AddHeader(c)

	c.Header("X-Powered-By", "https://x.com/48club_official")
	c.Request.Header.Set("X-Forwarded-For", userIP)

	if tooManyRequests {
		c.AbortWithStatus(http.StatusTooManyRequests)
		return
	}

}

func LimitMiddleware2(ip string, pass bool, count int, res *types.LimitResponse) bool {
	tooManyRequests := false

	for _, limit := range limit.Limits {
		limiter := limit.Allow(ip, pass, count)
		if res != nil {
			res.Limit = append(res.Limit, fmt.Sprintf("%d/%s", limiter.Limit, limiter.Wind))
			res.Remaining = append(res.Remaining, fmt.Sprintf("%d/%s", limiter.Limit-limiter.Used, limiter.Wind))
		}
		if !limiter.Allow {
			tooManyRequests = true
		}
	}

	return tooManyRequests
}

func AnyHandler(c *gin.Context) {
	c.Set("isRpc", tools.IsRpc(c.Request.Host, config.GlobalConfig.Domains))

	var body = []byte{}
	if c.Request.ContentLength != 0 {
		body, _ = io.ReadAll(c.Request.Body)
		defer c.Request.Body.Close()
	}

	switch c.Request.Method {
	case http.MethodHead:
		fallthrough
	case http.MethodOptions:
		c.AbortWithStatus(http.StatusNoContent)
	case http.MethodPost:
		rpcHandler(c, body)
	case http.MethodGet:

		if c.Request.URL.Path == "/ws/" && c.IsWebsocket() {
			handleWebSocket(c, fmt.Sprintf("ws://%s", strings.Split(config.GlobalConfig.Sentry, "://")[1]))
		}
		fallthrough
	default:
		proxyHandler(c, body, config.GlobalConfig.Sentry)
	}
}

func addLimitBatchReq(ip string, reqCount int) bool {
	if reqCount == 1 {
		return false
	}
	reqCount = 2*reqCount - 1
	if reqCount > config.GlobalConfig.MaxBatchQuery {
		return true
	}
	return LimitMiddleware2(ip, false, reqCount, nil)
}

func rpcHandler(c *gin.Context, body []byte) {
	resp, buildRespByAgent, batchCount := tools.DecodeRequestBody(c.Request.Host, body)

	if addLimitBatchReq(c.GetString("ip"), batchCount) {
		c.AbortWithStatus(http.StatusTooManyRequests)
		return
	}
	if buildRespByAgent {
		c.JSON(http.StatusOK, resp)
		return
	}

	proxyHandler(c, body, config.GlobalConfig.Sentry)
}

func proxyHandler(c *gin.Context, body []byte, toHost string) {
	if c.IsWebsocket() {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	target, _ := url.Parse(toHost)
	proxy := &httputil.ReverseProxy{
		Transport: httpTransport,
		Rewrite: func(r *httputil.ProxyRequest) {
			req := r.Out
			req.URL = target
			req.Host = c.Request.Host
			req.URL.Path = c.Request.URL.Path
			req.URL.RawQuery = c.Request.URL.RawQuery

			req.Header = http.Header{}
			for k, v := range r.In.Header {
				if strings.HasPrefix(strings.ToLower(k), "cf-") {
					continue
				}
				req.Header[k] = v
			}

			if target.Scheme == "https" {
				req.Header.Set("X-Forwarded-Proto", "https")
			} else {
				req.Header.Set("X-Forwarded-Proto", "http")
			}

			req.ContentLength = int64(len(body))
			req.Body = io.NopCloser(bytes.NewReader(body))
		},
		ModifyResponse: func(resp *http.Response) error {
			resp.Body = http.MaxBytesReader(nil, resp.Body, MaxResponseBodySize)
			resp.Header.Del("Access-Control-Allow-Origin")
			if resp.ContentLength <= 0 {
				resp.Header.Del("Content-Length")
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			c.AbortWithStatus(http.StatusBadGateway)
		},
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

var (
	httpTransport = http.DefaultTransport.(*http.Transport)
)

func init() {
	httpTransport.DisableCompression = true
	httpTransport.DisableKeepAlives = false
	httpTransport.MaxIdleConns = 120
	httpTransport.IdleConnTimeout = 65 * time.Second
}
