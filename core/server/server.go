package server

import (
	"github.com/tristendillon/conduit/core/config"
	"github.com/tristendillon/conduit/core/logger"
)

type Server struct {
	Config *config.Config
}

func NewServer() *Server {
	config, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config: %v", err)
	}
	return &Server{
		Config: config,
	}
}

func (s *Server) Start() error {
	logger.Info("Starting server on %s:%d", s.Config.Server.Host, s.Config.Server.Port)
	return nil
}
