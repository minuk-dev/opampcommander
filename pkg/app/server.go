package app

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"

	"github.com/minuk-dev/minuk-apiserver/internal/adapter/in/http/v1/opamp"
	"github.com/minuk-dev/minuk-apiserver/internal/adapter/in/http/v1/ping"
	"github.com/minuk-dev/minuk-apiserver/internal/domain/port"
	"github.com/minuk-dev/minuk-apiserver/internal/domain/service"
)

var (
	ErrAdapterInitFailed = errors.New("adapter init failed")
	ErrDomainInitFailed  = errors.New("domain init failed")
)

type ServerSettings struct{}

type Server struct {
	logger *slog.Logger
	Engine *gin.Engine

	// domains
	connectionUsecase port.ConnectionUsecase

	// applications

	// adapters
	pingController  *ping.Controller
	opampController *opamp.Controller
}

func NewServer(_ ServerSettings) *Server {
	logger := slog.Default()
	engine := gin.New()

	engine.Use(sloggin.New(logger))
	engine.Use(gin.Recovery())

	server := &Server{
		logger: logger,
		Engine: engine,

		connectionUsecase: nil,
		pingController:    nil,
		opampController:   nil,
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
	s.connectionUsecase = service.NewConnectionManager()
	if s.connectionUsecase == nil {
		return ErrDomainInitFailed
	}

	return nil
}

func (s *Server) initApplications() error {
	return nil
}

type controller interface {
	RoutesInfo() gin.RoutesInfo
}

func (s *Server) initAdapters() error {
	s.pingController = ping.NewController()
	if s.pingController == nil {
		return ErrAdapterInitFailed
	}

	s.opampController = opamp.NewController(
		opamp.WithConnectionUsecase(s.connectionUsecase),
	)
	if s.opampController == nil {
		return ErrAdapterInitFailed
	}

	controllers := []controller{
		s.pingController,
		s.opampController,
	}

	for _, controller := range controllers {
		routesInfo := controller.RoutesInfo()
		for _, routeInfo := range routesInfo {
			s.Engine.Handle(routeInfo.Method, routeInfo.Path, routeInfo.HandlerFunc)
		}
	}

	return nil
}
