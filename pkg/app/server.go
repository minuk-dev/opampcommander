package app

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/connection"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/opamp"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/ping"
	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/etcd"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	applicationservice "github.com/minuk-dev/opampcommander/internal/application/service"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	domainservice "github.com/minuk-dev/opampcommander/internal/domain/service"
)

var (
	ErrInfrastructureInitFailed = errors.New("infrastructure init failed")
	ErrInAdapterInitFailed      = errors.New("in adapter init failed")
	ErrOutAdapterInitFailed     = errors.New("out adapter init failed")
	ErrDomainInitFailed         = errors.New("domain init failed")
	ErrApplicationInitFailed    = errors.New("application init failed")
)

type ServerSettings struct {
	EtcdHosts []string
}

type Server struct {
	settings ServerSettings
	logger   *slog.Logger
	Engine   *gin.Engine

	// domains
	connectionUsecase domainport.ConnectionUsecase
	agentUsecase      domainport.AgentUsecase

	// applications
	opampUsecase applicationport.OpAMPUsecase

	// in adapters
	pingController       *ping.Controller
	opampController      *opamp.Controller
	connectionController *connection.Controller
	agentController      *agent.Controller

	// out adapters
	agentPersistencePort domainport.AgentPersistencePort
}

func NewServer(settings ServerSettings) *Server {
	logger := slog.Default()
	engine := gin.New()

	engine.Use(sloggin.New(logger))
	engine.Use(gin.Recovery())

	server := &Server{
		settings: settings,
		logger:   logger,
		Engine:   engine,

		agentPersistencePort: nil,
		connectionUsecase:    nil,
		agentUsecase:         nil,
		opampUsecase:         nil,
		pingController:       nil,
		agentController:      nil,
		opampController:      nil,
		connectionController: nil,
	}

	err := server.initOutAdapters()
	if err != nil {
		logger.Error("server init failed", "error", err.Error())

		return nil
	}

	err = server.initDomains()
	if err != nil {
		logger.Error("server init failed", "error", err.Error())

		return nil
	}

	err = server.initApplications()
	if err != nil {
		logger.Error("server init failed", "error", err.Error())

		return nil
	}

	err = server.initInAdapters()
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

	s.agentUsecase = domainservice.NewAgentService(
		s.agentPersistencePort,
	)
	if s.agentUsecase == nil {
		return ErrDomainInitFailed
	}

	return nil
}

func (s *Server) initApplications() error {
	s.opampUsecase = applicationservice.NewOpAMPService(
		s.connectionUsecase,
		s.agentUsecase,
		s.logger,
	)
	if s.opampUsecase == nil {
		return ErrDomainInitFailed
	}

	return nil
}

type controller interface {
	RoutesInfo() gin.RoutesInfo
}

func (s *Server) initOutAdapters() error {
	//exhaustruct:ignore
	etcdConfig := clientv3.Config{
		Endpoints: s.settings.EtcdHosts,
	}

	etcdClient, err := clientv3.New(etcdConfig)
	if err != nil {
		return fmt.Errorf("etcd client init failed: %w", err)
	}

	s.agentPersistencePort = etcd.NewAgentEtcdAdapter(etcdClient)

	return nil
}

func (s *Server) initInAdapters() error {
	s.pingController = ping.NewController()
	if s.pingController == nil {
		return ErrInAdapterInitFailed
	}

	s.opampController = opamp.NewController(
		s.opampUsecase,
	)
	if s.opampController == nil {
		return ErrInAdapterInitFailed
	}

	s.connectionController = connection.NewController(
		connection.WithConnectionUsecase(s.connectionUsecase),
	)
	if s.connectionController == nil {
		return ErrInAdapterInitFailed
	}

	s.agentController = agent.NewController(s.agentUsecase)
	if s.agentController == nil {
		return ErrInAdapterInitFailed
	}

	controllers := []controller{
		s.pingController,
		s.opampController,
		s.connectionController,
		s.agentController,
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
