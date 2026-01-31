package command

import (
	"github.com/local-mcps/dev-mcps/config"
	"github.com/local-mcps/dev-mcps/internal/common"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

type Server struct {
	config    *config.CommandConfig
	validator *common.CommandValidator
	logger    *common.Logger
	executor  *Executor
}

func NewServer(cfg *config.CommandConfig) *Server {
	return &Server{
		config:    cfg,
		validator: common.NewCommandValidator(cfg.AllowedCommands, cfg.DeniedCommands),
		logger:    common.NewLogger(common.LogLevelInfo, common.LogFormatJSON, nil, "command"),
		executor:  NewExecutor(cfg),
	}
}

func (s *Server) RegisterTools(server *mcp.Server) {
	server.RegisterTool(s.runCommandTool())
	server.RegisterTool(s.runCommandAsyncTool())
	server.RegisterTool(s.getCommandStatusTool())
	server.RegisterTool(s.cancelCommandTool())
	server.RegisterTool(s.runScriptTool())
	server.RegisterTool(s.getShellInfoTool())
}
