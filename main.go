package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	route := r.Use(domainMiddleware)
	route.POST("/", postHandler)

	if err := r.Run(":80"); err != nil {
		panic(err)
	}
}

func init() {
	loadConfig("config.json")
}
