package process

import (
	"github.com/local-mcps/dev-mcps/config"
	"github.com/local-mcps/dev-mcps/internal/common"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

type Server struct {
	config *config.ProcessConfig
	logger *common.Logger
}

func NewServer(cfg *config.ProcessConfig) *Server {
	return &Server{
		config: cfg,
		logger: common.NewLogger(common.LogLevelInfo, common.LogFormatJSON, nil, "process"),
	}
}

func (s *Server) RegisterTools(server *mcp.Server) {
	server.RegisterTool(s.listProcessesTool())
	server.RegisterTool(s.getProcessInfoTool())
	server.RegisterTool(s.killProcessTool())
	server.RegisterTool(s.findProcessByPortTool())
	server.RegisterTool(s.getResourceUsageTool())
	server.RegisterTool(s.waitForProcessTool())
	server.RegisterTool(s.startProcessTool())
}
