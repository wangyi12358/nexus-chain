package net

import (
	"context"
	"fmt"
	"nexus-chain/pkg/config"
	"nexus-chain/pkg/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

func NewHTTPServer() (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.Cors(), middleware.GinContext2Context())
	return r, nil
}

func StartHTTPServer(lc fx.Lifecycle, r *gin.Engine, config *config.Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				fmt.Printf("Server running at :%s \n", config.HTTP.Port)
				if err := r.Run(config.HTTP.Port); err != nil {
					fmt.Println("Server stopped with error:", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			fmt.Println("Server stopping...")
			return nil
		},
	})
}
