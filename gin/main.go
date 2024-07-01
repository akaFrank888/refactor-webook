package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	server := gin.Default()
	// middleware
	server.Use(func(ctx *gin.Context) {
		// ...HandlerFunc （参数为*gin.Context的匿名函数）

	})

	// 注册路由
	server.GET("/hello", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "hello world")
	})

	server.Run(":8080")
}
