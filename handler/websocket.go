package handler

import (
	"context"
	"log"
	"net/http"
	"service_agent/ethclient"
	"service_agent/limit"
	"service_agent/tools"
	"sync"

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

func handleWebSocket(c *gin.Context, toHost string, isRpc bool) {
	ip := c.ClientIP()
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	// 将请求升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request.Clone(ctx), nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}
	defer conn.Close()

	// 连接到目标 WebSocket 服务器
	proxyConn, _, err := websocket.DefaultDialer.DialContext(ctx, toHost, http.Header{
		"Origin": {c.Request.Header.Get("Origin")},
		"Host":   {c.Request.Host},
	})

	if err != nil {
		log.Println("Failed to connect to target server:", err)
		return
	}
	defer proxyConn.Close()

	// 使用 sync.WaitGroup 确保所有 goroutine 正确完成
	var wg sync.WaitGroup
	wg.Add(2)

	cancelConn := func(c *websocket.Conn) {
		cancelCtx()
		_ = c.Close()
		wg.Done()
	}

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

				_, _, tooManyRequests := LimitMiddleware2(ip)

				if tooManyRequests {
					_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Too many requests"))
					return
				}
				limit.Limits.AllowPassCheck(ip)

				if isRpc && messageType == websocket.TextMessage {
					if web3Reqi, hasGasPrice, hasSendRawTransaction, err := tools.DecodeRequestBody(message); err == nil {
						if hasGasPrice {
							if err := conn.WriteJSON(tools.GetGasPrice(web3Reqi)); err != nil {
								log.Println("Write error to client:", err)
								return
							}
							continue
						} else if hasSendRawTransaction {
							msg, err := ethclient.SendRawTransaction(message)
							if err != nil {
								_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
								return
							}
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

	// 等待所有 goroutine 结束
	wg.Wait()
}
