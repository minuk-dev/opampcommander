package app

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"

	"github.com/minuk-dev/minuk-apiserver/internal/adapter/in/http/v1/ping"
)

var ErrAdapterInitFailed = errors.New("adapter init failed")

type ServerSettings struct{}

type Server struct {
	logger *slog.Logger
	Engine *gin.Engine
}

func NewServer(_ ServerSettings) *Server {
	logger := slog.Default()
	engine := gin.New()

	engine.Use(sloggin.New(logger))
	engine.Use(gin.Recovery())

	server := &Server{
		logger: logger,
		Engine: engine,
	}

	err := server.initDomains()
	if err != nil {
		logger.Error("server init failed", "error", err.Error())

		return nil
	}

	err = server.initApplications()
	if err != nil {
		logger.Error("server init failed", "error", err.Error())

		return nil
	}

	err = server.initAdapters()
	if err != nil {
		logger.Error("server init failed", "error", err.Error())

		return nil
	}

	return server
}

func (s *Server) Run() error {
	err := s.Engine.Run()
	if err != nil {
		return fmt.Errorf("server run failed: %w", err)
	}

	return nil
}

func (s *Server) initDomains() error {
	return nil
}

func (s *Server) initApplications() error {
	return nil
}

func (s *Server) initAdapters() error {
	pingController := ping.NewController()
	if pingController == nil {
		return ErrAdapterInitFailed
	}

	s.Engine.GET(pingController.Path(), pingController.Handle)

	return nil
}
