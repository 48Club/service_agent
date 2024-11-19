package handler

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/48Club/service_agent/ethclient"
	"github.com/48Club/service_agent/limit"
	"github.com/48Club/service_agent/tools"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(c *gin.Context, toHost string) {
	ctx, cancelCtx := context.WithCancel(c.Request.Context())
	defer cancelCtx()

	conn, err := upgrader.Upgrade(c.Writer, c.Request.Clone(ctx), nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	defer conn.Close()

	proxyConn, _, err := websocket.DefaultDialer.DialContext(ctx, toHost, http.Header{
		"Origin": {c.Request.Header.Get("Origin")},
		"Host":   {c.Request.Host},
	})

	if err != nil {
		log.Println("Failed to connect to target server:", err)
		return
	}
	defer proxyConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	cancelConn := func(c *websocket.Conn) {
		cancelCtx()
		_ = c.Close()
		wg.Done()
	}

	isRpc, ip, host := c.GetBool("isRpc"), c.GetString("ip"), c.Request.Host

	go func() {
		defer cancelConn(proxyConn)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				messageType, message, err := conn.ReadMessage()
				if err != nil {
					log.Println("Read error from client:", err)
					return
				}

				if LimitMiddleware2(ip, true, 1, nil) {
					_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Too many requests"))
					return
				}
				limit.Limits.AllowPassCheck(ip)

				if isRpc && messageType == websocket.TextMessage {
					if reqCount, web3Reqi, mustSend2Sentry, buildRespByAgent, resp, ethCallCount, ethSendRawTransactionCount, err := tools.DecodeRequestBody(isRpc, host, message); err == nil {
						go qpsStats.Add(reqCount, ethCallCount, ethSendRawTransactionCount)
						if addLimitBatchReq(ip, reqCount) {
							// 429 Too Many Requests
							_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Too many requests"))
							return
						}

						if buildRespByAgent {
							// 由 agent 生成响应
							if err := conn.WriteJSON(tools.EthResp(web3Reqi, resp)); err != nil {
								log.Println("Write error to client:", err)
								return
							}
							continue
						} else if mustSend2Sentry {
							// 需要发送到 sentry
							msg, err := ethclient.Send2Sentry(message)
							if err != nil {
								_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
								return
							}
							// 返回 sentry 的响应
							if err := conn.WriteMessage(messageType, msg); err != nil {
								log.Println("Write error to client:", err)
								return
							}
							continue
						}
					}

				}

				if err := proxyConn.WriteMessage(messageType, message); err != nil {
					log.Println("Write error to target server:", err)
					return
				}
			}
		}
	}()

	go func() {
		defer cancelConn(conn)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				messageType, message, err := proxyConn.ReadMessage()
				if err != nil {
					log.Println("Read error from target server:", err)
					return
				}
				if err := conn.WriteMessage(messageType, message); err != nil {
					log.Println("Write error to client:", err)
					return
				}
			}
		}
	}()

	wg.Wait()
}
