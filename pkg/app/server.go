package app

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"

	"github.com/minuk-dev/minuk-apiserver/internal/adapter/in/http/v1/ping"
)

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
		// todo: log error
		return nil
	}

	err = server.initApplications()
	if err != nil {
		// todo: log error
		return nil
	}

	err = server.initAdapters()
	if err != nil {
		// todo: log error
		return nil
	}

	return server
}

func (s *Server) Run() error {
	return s.Engine.Run()
}

func (s *Server) initDomains() error {
	return nil
}

func (s *Server) initApplications() error {
	return nil
}

func (s *Server) initAdapters() error {
	pingController := ping.NewController()
	s.Engine.GET(pingController.Path(), pingController.Handle)

	return nil
}
