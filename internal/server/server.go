package server

import (
	"github.com/gin-gonic/gin"
)

// NewRouter creates and configures the Gin router
func NewRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	return r
}
