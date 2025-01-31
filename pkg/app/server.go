package app

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/connection"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/opamp"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/ping"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	applicationservice "github.com/minuk-dev/opampcommander/internal/application/service"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	domainservice "github.com/minuk-dev/opampcommander/internal/domain/service"
)

var (
	ErrAdapterInitFailed     = errors.New("adapter init failed")
	ErrDomainInitFailed      = errors.New("domain init failed")
	ErrApplicationInitFailed = errors.New("application init failed")
)

type ServerSettings struct{}

type Server struct {
	logger *slog.Logger
	Engine *gin.Engine

	// domains
	connectionUsecase domainport.ConnectionUsecase
	agentUsecase      domainport.AgentUsecase

	// applications
	opampUsecase applicationport.OpAMPUsecase

	// adapters
	pingController       *ping.Controller
	opampController      *opamp.Controller
	connectionController *connection.Controller
}

func NewServer(_ ServerSettings) *Server {
	logger := slog.Default()
	engine := gin.New()

	engine.Use(sloggin.New(logger))
	engine.Use(gin.Recovery())

	server := &Server{
		logger: logger,
		Engine: engine,

		connectionUsecase:    nil,
		agentUsecase:         nil,
		opampUsecase:         nil,
		pingController:       nil,
		opampController:      nil,
		connectionController: nil,
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
	s.connectionUsecase = domainservice.NewConnectionManager()
	if s.connectionUsecase == nil {
		return ErrDomainInitFailed
	}

	return nil
}

func (s *Server) initApplications() error {
	s.opampUsecase = applicationservice.NewOpAMPService(
		s.connectionUsecase,
		s.agentUsecase,
	)
	if s.opampUsecase == nil {
		return ErrDomainInitFailed
	}

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
		s.opampUsecase,
	)
	if s.opampController == nil {
		return ErrAdapterInitFailed
	}

	s.connectionController = connection.NewController(
		connection.WithConnectionUsecase(s.connectionUsecase),
	)
	if s.connectionController == nil {
		return ErrAdapterInitFailed
	}

	controllers := []controller{
		s.pingController,
		s.opampController,
		s.connectionController,
	}

	for _, controller := range controllers {
		routesInfo := controller.RoutesInfo()
		for _, routeInfo := range routesInfo {
			s.Engine.Handle(routeInfo.Method, routeInfo.Path, routeInfo.HandlerFunc)
		}
	}

	for _, routeInfo := range s.Engine.Routes() {
		s.logger.Info("engine routes - ", "routeInfo", routeInfo)
	}

	return nil
}
