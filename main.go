package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"service_agent/cloudflare"
	"service_agent/limit"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	cloudflare.SetRemoteAddr(r)

	r.Use(gin.Recovery())
	r.Use(cors.New(
		cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders: []string{"Accept", "Authorization", "Cache-Control", "Content-Type", "DNT", "If-Modified-Since", "Keep-Alive", "Origin", "User-Agent", "X-Requested-With"},
		},
	), customRecoveryMiddleware, setMaxRequestBodySize, limitMiddleware)

	r.NoRoute(anyHandler)
	r.NoMethod(anyHandler)

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

	if err := limits.SaveCache(); err != nil {
		log.Fatalf("save to cache failed:%+v", err)
	}

	log.Print("server exited properly")
}

func init() {
	loadConfig("config.json")

	limits = limit.IPBasedRateLimiters{
		limit.NewIPBasedRateLimiter(80, time.Second*5, "5s"),     // 16qps
		limit.NewIPBasedRateLimiter(720, time.Minute, "1m"),      // 12qps
		limit.NewIPBasedRateLimiter(28800, time.Hour, "1h"),      // 8qps
		limit.NewIPBasedRateLimiter(345600, time.Hour*24, "24h"), // 4qps
	}

	if err := limits.LoadFromCache(); err != nil {
		log.Fatalf("load from redis failed:%+v", err)
	}
}
