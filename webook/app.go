package main

import (
	"github.com/gin-gonic/gin"
	"refactor-webook/webook/internal/events"
)

type App struct {
	server    *gin.Engine
	consumers []events.Consumer
}
