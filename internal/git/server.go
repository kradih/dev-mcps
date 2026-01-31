package git

import (
	"github.com/local-mcps/dev-mcps/config"
	"github.com/local-mcps/dev-mcps/internal/common"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

type Server struct {
	config    *config.GitConfig
	validator *common.PathValidator
	logger    *common.Logger
}

func NewServer(cfg *config.GitConfig) *Server {
	return &Server{
		config:    cfg,
		validator: common.NewPathValidator(cfg.AllowedRepositories, nil, true),
		logger:    common.NewLogger(common.LogLevelInfo, common.LogFormatJSON, nil, "git"),
	}
}

func (s *Server) RegisterTools(server *mcp.Server) {
	server.RegisterTool(s.gitStatusTool())
	server.RegisterTool(s.gitLogTool())
	server.RegisterTool(s.gitDiffTool())
	server.RegisterTool(s.gitBranchListTool())
	server.RegisterTool(s.gitBranchCreateTool())
	server.RegisterTool(s.gitCheckoutTool())
	server.RegisterTool(s.gitAddTool())
	server.RegisterTool(s.gitCommitTool())
	server.RegisterTool(s.gitPushTool())
	server.RegisterTool(s.gitPullTool())
	server.RegisterTool(s.gitCloneTool())
	server.RegisterTool(s.gitStashTool())
	server.RegisterTool(s.gitBlameTool())
	server.RegisterTool(s.gitShowTool())
}
