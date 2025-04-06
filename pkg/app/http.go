package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const (
	DefaultHTTPReadTimeout = 30 * time.Second
)

func NewHTTPServer(lc fx.Lifecycle, engine *gin.Engine) *http.Server {
	srv := &http.Server{
		ReadTimeout: DefaultHTTPReadTimeout,
		Addr:        ":8080",
		Handler:     engine,
	}

	// srv.ConnContext = opampController.ConnContext
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			fmt.Println("Starting HTTP server at", srv.Addr)
			go srv.Serve(ln)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})

	return srv
}
