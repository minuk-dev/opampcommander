package app

import (
	"github.com/gin-gonic/gin"
)

type Server struct {
	Engine *gin.Engine
}

func NewServer() *Server {
	engine := gin.Default()
	server := &Server{
		Engine: engine,
	}

	server.initDomains()
	server.initApplications()
	server.initAdapters()

	return server
}

func (a *Server) Run() error {
	return a.Engine.Run()
}

func (a *Server) initDomains() error {
	return nil
}

func (a *Server) initApplications() error {
	return nil
}

func (a *Server) initAdapters() error {
	return nil
}
