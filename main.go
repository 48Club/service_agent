package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/48Club/service_agent/cloudflare"
	"github.com/48Club/service_agent/handler"
	"github.com/48Club/service_agent/limit"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()
	r.Use(handler.CustomLoggerMiddleware, gin.Recovery())

	cloudflare.SetRemoteAddr(r)

	r.Use(cors.New(
		cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders: []string{"Accept", "Authorization", "Cache-Control", "Content-Type", "DNT", "If-Modified-Since", "Keep-Alive", "Origin", "User-Agent", "X-Requested-With"},
		},
	), handler.SetMaxRequestBodySize, handler.LimitMiddleware, handler.CustomRecoveryMiddleware)

	r.NoRoute(handler.AnyHandler)
	r.NoMethod(handler.AnyHandler)

	srv := &http.Server{
		Addr:    ":80",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	s := <-sig
	log.Printf("Signal (%v) received, stopping\n", s)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown failed:%+v", err)
	}

	if err := limit.Limits.SaveCache(); err != nil {
		log.Fatalf("save to cache failed:%+v", err)
	}

	log.Print("server exited properly")
}

func init() {
	if err := limit.Limits.LoadFromCache(); err != nil {
		log.Fatalf("load from redis failed:%+v", err)
	}

}
