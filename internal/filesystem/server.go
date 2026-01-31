package filesystem

import (
	"github.com/local-mcps/dev-mcps/config"
	"github.com/local-mcps/dev-mcps/internal/common"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

type Server struct {
	config    *config.FilesystemConfig
	validator *common.PathValidator
	logger    *common.Logger
}

func NewServer(cfg *config.FilesystemConfig) *Server {
	return &Server{
		config:    cfg,
		validator: common.NewPathValidator(cfg.AllowedPaths, cfg.DeniedPaths, cfg.FollowSymlinks),
		logger:    common.NewLogger(common.LogLevelInfo, common.LogFormatJSON, nil, "filesystem"),
	}
}

func (s *Server) RegisterTools(server *mcp.Server) {
	server.RegisterTool(s.readFileTool())
	server.RegisterTool(s.readFileLinesTool())
	server.RegisterTool(s.writeFileTool())
	server.RegisterTool(s.appendFileTool())
	server.RegisterTool(s.deleteFileTool())
	server.RegisterTool(s.moveFileTool())
	server.RegisterTool(s.copyFileTool())
	server.RegisterTool(s.listDirectoryTool())
	server.RegisterTool(s.createDirectoryTool())
	server.RegisterTool(s.deleteDirectoryTool())
	server.RegisterTool(s.fileInfoTool())
	server.RegisterTool(s.searchFilesTool())
	server.RegisterTool(s.grepTool())
}
