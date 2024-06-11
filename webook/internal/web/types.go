package web

import "github.com/gin-gonic/gin"

type handler interface {
	RegisterRoutes(server *gin.Engine)
}

type Page struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}
