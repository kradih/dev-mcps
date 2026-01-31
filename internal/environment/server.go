package environment

import (
	"github.com/local-mcps/dev-mcps/config"
	"github.com/local-mcps/dev-mcps/internal/common"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

type Server struct {
	config     *config.EnvironmentConfig
	logger     *common.Logger
	sessionEnv map[string]string
}

func NewServer(cfg *config.EnvironmentConfig) *Server {
	return &Server{
		config:     cfg,
		logger:     common.NewLogger(common.LogLevelInfo, common.LogFormatJSON, nil, "environment"),
		sessionEnv: make(map[string]string),
	}
}

func (s *Server) RegisterTools(server *mcp.Server) {
	server.RegisterTool(s.getEnvTool())
	server.RegisterTool(s.setEnvTool())
	server.RegisterTool(s.listEnvTool())
	server.RegisterTool(s.unsetEnvTool())
	server.RegisterTool(s.getSystemInfoTool())
	server.RegisterTool(s.getUserInfoTool())
	server.RegisterTool(s.getPathInfoTool())
	server.RegisterTool(s.expandPathTool())
}
