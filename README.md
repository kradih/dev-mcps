# dev-mcps - Local MCP Servers

A comprehensive collection of Model Context Protocol (MCP) servers built with Go, providing essential tools for AI agents to interact with the local development environment.

## Quick Start

```bash
# Build all servers
make build

# Run the combined server (all tools)
./bin/all-server

# Or run individual servers
./bin/filesystem-server
./bin/command-server
./bin/environment-server
./bin/git-server
./bin/process-server
./bin/web-server
```

## Available Servers

| Server | Binary | Description |
|--------|--------|-------------|
| All-in-One | `all-server` | Combined server with all tools |
| Filesystem | `filesystem-server` | File and directory operations |
| Command | `command-server` | Shell command execution |
| Environment | `environment-server` | Environment variables and system info |
| Git | `git-server` | Git repository operations |
| Process | `process-server` | Process management and monitoring |
| Web | `web-server` | HTTP fetching and web content |

## Tools Reference

### Filesystem Server (13 tools)
- `read_file`, `read_file_lines` - Read file contents
- `write_file`, `append_file` - Write to files
- `delete_file`, `move_file`, `copy_file` - File operations
- `list_directory`, `create_directory`, `delete_directory` - Directory operations
- `file_info` - Get file metadata
- `search_files`, `grep` - Search functionality

### Command Server (6 tools)
- `run_command` - Execute commands synchronously
- `run_command_async` - Execute commands asynchronously
- `get_command_status` - Check async command status
- `cancel_command` - Cancel running commands
- `run_script` - Execute script files
- `get_shell_info` - Get shell information

### Environment Server (8 tools)
- `get_env`, `set_env`, `list_env`, `unset_env` - Environment variables
- `get_system_info` - System information (OS, CPU, memory)
- `get_user_info` - Current user information
- `get_path_info` - PATH and related paths
- `expand_path` - Expand path variables

### Git Server (14 tools)
- `git_status`, `git_log`, `git_diff` - Repository state
- `git_branch_list`, `git_branch_create`, `git_checkout` - Branch operations
- `git_add`, `git_commit` - Staging and committing
- `git_push`, `git_pull`, `git_clone` - Remote operations
- `git_stash`, `git_blame`, `git_show` - Utility operations

### Process Server (7 tools)
- `list_processes`, `get_process_info` - Process information
- `kill_process` - Terminate processes
- `find_process_by_port` - Find process by port
- `get_resource_usage` - System resource usage
- `wait_for_process`, `start_process` - Process lifecycle

### Web Server (6 tools)
- `fetch_url` - Fetch raw URL content
- `fetch_html`, `fetch_text`, `fetch_markdown` - Content extraction
- `fetch_json` - JSON API requests
- `extract_links` - Extract links from pages

## Configuration

Copy `config/config.yaml.example` to `~/.config/local-mcps/config.yaml`:

```bash
mkdir -p ~/.config/local-mcps
cp config/config.yaml.example ~/.config/local-mcps/config.yaml
```

Or use environment variables:

```bash
export LOCAL_MCP_LOG_LEVEL=debug
export LOCAL_MCP_FILESYSTEM_ALLOWED_PATHS=$HOME
```

## MCP Client Configuration

Use the `mcp.json` file or add to your MCP client config:

```json
{
  "mcpServers": {
    "local-all": {
      "command": "/path/to/dev-mcps/bin/all-server",
      "args": []
    }
  }
}
```

## Development

```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Format and lint
make lint

# Clean build artifacts
make clean
```

## Project Structure

```
dev-mcps/
├── cmd/                    # Server entry points
│   ├── all/               # Combined server
│   ├── filesystem/        # Filesystem server
│   ├── command/           # Command server
│   ├── environment/       # Environment server
│   ├── git/               # Git server
│   ├── process/           # Process server
│   └── web/               # Web server
├── internal/              # Internal packages
│   ├── common/            # Shared utilities
│   ├── filesystem/        # Filesystem implementation
│   ├── command/           # Command implementation
│   ├── environment/       # Environment implementation
│   ├── git/               # Git implementation
│   ├── process/           # Process implementation
│   └── web/               # Web implementation
├── pkg/mcp/               # MCP server framework
├── config/                # Configuration
├── bin/                   # Built binaries
├── mcp.json              # MCP client configuration
├── Makefile              # Build commands
└── README.md             # This file
```

## License

MIT License
