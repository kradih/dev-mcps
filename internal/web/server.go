package web

import (
	"github.com/local-mcps/dev-mcps/config"
	"github.com/local-mcps/dev-mcps/internal/common"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

type Server struct {
	config *config.WebConfig
	logger *common.Logger
}

func NewServer(cfg *config.WebConfig) *Server {
	return &Server{
		config: cfg,
		logger: common.NewLogger(common.LogLevelInfo, common.LogFormatJSON, nil, "web"),
	}
}

func (s *Server) RegisterTools(server *mcp.Server) {
	server.RegisterTool(s.fetchURLTool())
	server.RegisterTool(s.fetchHTMLTool())
	server.RegisterTool(s.fetchTextTool())
	server.RegisterTool(s.fetchMarkdownTool())
	server.RegisterTool(s.fetchJSONTool())
	server.RegisterTool(s.extractLinksTool())
}
