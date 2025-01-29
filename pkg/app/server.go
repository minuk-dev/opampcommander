package app

import (
	"github.com/gin-gonic/gin"
)

type ServerSettings struct {
}

type Server struct {
	// logger *slog.Logger
	Engine *gin.Engine
}

func NewServer(settings ServerSettings) *Server {
	engine := gin.Default()
	server := &Server{
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
	return nil
}
