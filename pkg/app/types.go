package app

import "github.com/gin-gonic/gin"

type InHTTPAdapter interface {
	Path() string
	Handle(ctx *gin.Context)
}
