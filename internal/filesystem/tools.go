package filesystem

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/local-mcps/dev-mcps/internal/common"
	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	SizeBytes   int64     `json:"size_bytes"`
	Permissions string    `json:"permissions"`
	IsDirectory bool      `json:"is_directory"`
	IsSymlink   bool      `json:"is_symlink"`
	ModifiedAt  time.Time `json:"modified_at"`
}

type DirectoryEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDirectory bool   `json:"is_directory"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
}

type GrepMatch struct {
	File       string `json:"file"`
	LineNumber int    `json:"line_number"`
	Line       string `json:"line"`
}

func (s *Server) readFileTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "read_file",
		Description: "Read the contents of a file",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"path": mcp.StringProperty("Absolute path to the file"),
			},
			[]string{"path"},
		),
		Handler: s.handleReadFile,
	}
}

func (s *Server) handleReadFile(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	path, err := mcp.GetStringParam(params, "path", true)
	if err != nil {
		return nil, err
	}

	absPath, err := s.validator.ResolvePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", common.ErrNotFound, path)
		}
		return nil, err
	}

	if info.IsDir() {
		return nil, fmt.Errorf("%w: %s", common.ErrNotAFile, path)
	}

	maxSize := int64(s.config.MaxFileSizeMB) * 1024 * 1024
	if info.Size() > maxSize {
		return nil, fmt.Errorf("%w: file size %d exceeds limit %d", common.ErrFileTooLarge, info.Size(), maxSize)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	return mcp.TextResult(string(content)), nil
}

func (s *Server) readFileLinesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "read_file_lines",
		Description: "Read specific line range from a file",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"path":       mcp.StringProperty("Absolute path to the file"),
				"start_line": mcp.IntProperty("Starting line number (1-indexed)"),
				"end_line":   mcp.IntProperty("Ending line number (inclusive)"),
			},
			[]string{"path", "start_line", "end_line"},
		),
		Handler: s.handleReadFileLines,
	}
}

func (s *Server) handleReadFileLines(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	path, err := mcp.GetStringParam(params, "path", true)
	if err != nil {
		return nil, err
	}

	startLine, err := mcp.GetIntParam(params, "start_line", true, 1)
	if err != nil {
		return nil, err
	}

	endLine, err := mcp.GetIntParam(params, "end_line", true, 0)
	if err != nil {
		return nil, err
	}

	if startLine < 1 {
		startLine = 1
	}
	if endLine < startLine {
		return nil, fmt.Errorf("end_line must be >= start_line")
	}

	absPath, err := s.validator.ResolvePath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum >= startLine && lineNum <= endLine {
			lines = append(lines, scanner.Text())
		}
		if lineNum > endLine {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return mcp.TextResult(strings.Join(lines, "\n")), nil
}

func (s *Server) writeFileTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "write_file",
		Description: "Write content to a file (create or overwrite)",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"path":    mcp.StringProperty("Absolute path to the file"),
				"content": mcp.StringProperty("Content to write"),
			},
			[]string{"path", "content"},
		),
		Handler: s.handleWriteFile,
	}
}

func (s *Server) handleWriteFile(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	path, err := mcp.GetStringParam(params, "path", true)
	if err != nil {
		return nil, err
	}

	content, err := mcp.GetStringParam(params, "content", true)
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	if err := s.validator.ValidatePath(filepath.Dir(absPath)); err != nil {
		return nil, err
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return nil, err
	}

	return mcp.TextResult(fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), absPath)), nil
}

func (s *Server) appendFileTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "append_file",
		Description: "Append content to an existing file",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"path":    mcp.StringProperty("Absolute path to the file"),
				"content": mcp.StringProperty("Content to append"),
			},
			[]string{"path", "content"},
		),
		Handler: s.handleAppendFile,
	}
}

func (s *Server) handleAppendFile(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	path, err := mcp.GetStringParam(params, "path", true)
	if err != nil {
		return nil, err
	}

	content, err := mcp.GetStringParam(params, "content", true)
	if err != nil {
		return nil, err
	}

	absPath, err := s.validator.ResolvePath(path)
	if err != nil {
		if !common.IsPathNotAllowed(err) {
			absPath, _ = filepath.Abs(path)
			if err := s.validator.ValidatePath(filepath.Dir(absPath)); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	file, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return nil, err
	}

	return mcp.TextResult(fmt.Sprintf("Successfully appended %d bytes to %s", len(content), absPath)), nil
}

func (s *Server) deleteFileTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_file",
		Description: "Delete a file",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"path": mcp.StringProperty("Absolute path to the file"),
			},
			[]string{"path"},
		),
		Handler: s.handleDeleteFile,
	}
}

func (s *Server) handleDeleteFile(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	path, err := mcp.GetStringParam(params, "path", true)
	if err != nil {
		return nil, err
	}

	absPath, err := s.validator.ResolvePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, fmt.Errorf("%w: use delete_directory for directories", common.ErrNotAFile)
	}

	if err := os.Remove(absPath); err != nil {
		return nil, err
	}

	return mcp.TextResult(fmt.Sprintf("Successfully deleted %s", absPath)), nil
}

func (s *Server) moveFileTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "move_file",
		Description: "Move or rename a file",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"source":      mcp.StringProperty("Source file path"),
				"destination": mcp.StringProperty("Destination file path"),
			},
			[]string{"source", "destination"},
		),
		Handler: s.handleMoveFile,
	}
}

func (s *Server) handleMoveFile(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	source, err := mcp.GetStringParam(params, "source", true)
	if err != nil {
		return nil, err
	}

	destination, err := mcp.GetStringParam(params, "destination", true)
	if err != nil {
		return nil, err
	}

	srcPath, err := s.validator.ResolvePath(source)
	if err != nil {
		return nil, err
	}

	dstPath, err := filepath.Abs(destination)
	if err != nil {
		return nil, err
	}

	if err := s.validator.ValidatePath(filepath.Dir(dstPath)); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return nil, err
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		return nil, err
	}

	return mcp.TextResult(fmt.Sprintf("Successfully moved %s to %s", srcPath, dstPath)), nil
}

func (s *Server) copyFileTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "copy_file",
		Description: "Copy a file",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"source":      mcp.StringProperty("Source file path"),
				"destination": mcp.StringProperty("Destination file path"),
			},
			[]string{"source", "destination"},
		),
		Handler: s.handleCopyFile,
	}
}

func (s *Server) handleCopyFile(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	source, err := mcp.GetStringParam(params, "source", true)
	if err != nil {
		return nil, err
	}

	destination, err := mcp.GetStringParam(params, "destination", true)
	if err != nil {
		return nil, err
	}

	srcPath, err := s.validator.ResolvePath(source)
	if err != nil {
		return nil, err
	}

	dstPath, err := filepath.Abs(destination)
	if err != nil {
		return nil, err
	}

	if err := s.validator.ValidatePath(filepath.Dir(dstPath)); err != nil {
		return nil, err
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return nil, err
	}

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return nil, err
	}
	defer dstFile.Close()

	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return nil, err
	}

	srcInfo, _ := srcFile.Stat()
	if srcInfo != nil {
		os.Chmod(dstPath, srcInfo.Mode())
	}

	return mcp.TextResult(fmt.Sprintf("Successfully copied %d bytes from %s to %s", written, srcPath, dstPath)), nil
}

func (s *Server) listDirectoryTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_directory",
		Description: "List contents of a directory",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"path":           mcp.StringProperty("Absolute path to directory"),
				"recursive":      mcp.BoolProperty("Include subdirectories"),
				"include_hidden": mcp.BoolProperty("Include hidden files"),
			},
			[]string{"path"},
		),
		Handler: s.handleListDirectory,
	}
}

func (s *Server) handleListDirectory(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	path, err := mcp.GetStringParam(params, "path", true)
	if err != nil {
		return nil, err
	}

	recursive, _ := mcp.GetBoolParam(params, "recursive", false)
	includeHidden, _ := mcp.GetBoolParam(params, "include_hidden", false)

	absPath, err := s.validator.ResolvePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", common.ErrNotADirectory, path)
	}

	var entries []DirectoryEntry

	if recursive {
		err = filepath.Walk(absPath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if p == absPath {
				return nil
			}

			name := info.Name()
			if !includeHidden && strings.HasPrefix(name, ".") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			entries = append(entries, DirectoryEntry{
				Name:        name,
				Path:        p,
				IsDirectory: info.IsDir(),
				SizeBytes:   info.Size(),
			})
			return nil
		})
	} else {
		dirEntries, err := os.ReadDir(absPath)
		if err != nil {
			return nil, err
		}

		for _, entry := range dirEntries {
			name := entry.Name()
			if !includeHidden && strings.HasPrefix(name, ".") {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			entries = append(entries, DirectoryEntry{
				Name:        name,
				Path:        filepath.Join(absPath, name),
				IsDirectory: entry.IsDir(),
				SizeBytes:   info.Size(),
			})
		}
	}

	if err != nil {
		return nil, err
	}

	return mcp.JSONResult(map[string]interface{}{
		"path":    absPath,
		"entries": entries,
		"count":   len(entries),
	})
}

func (s *Server) createDirectoryTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "create_directory",
		Description: "Create a directory (with parents)",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"path": mcp.StringProperty("Absolute path to directory"),
			},
			[]string{"path"},
		),
		Handler: s.handleCreateDirectory,
	}
}

func (s *Server) handleCreateDirectory(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	path, err := mcp.GetStringParam(params, "path", true)
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	if err := s.validator.ValidatePath(filepath.Dir(absPath)); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, err
	}

	return mcp.TextResult(fmt.Sprintf("Successfully created directory %s", absPath)), nil
}

func (s *Server) deleteDirectoryTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_directory",
		Description: "Delete a directory",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"path":      mcp.StringProperty("Absolute path to directory"),
				"recursive": mcp.BoolProperty("Delete contents recursively"),
			},
			[]string{"path"},
		),
		Handler: s.handleDeleteDirectory,
	}
}

func (s *Server) handleDeleteDirectory(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	path, err := mcp.GetStringParam(params, "path", true)
	if err != nil {
		return nil, err
	}

	recursive, _ := mcp.GetBoolParam(params, "recursive", false)

	absPath, err := s.validator.ResolvePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", common.ErrNotADirectory, path)
	}

	if recursive {
		if err := os.RemoveAll(absPath); err != nil {
			return nil, err
		}
	} else {
		if err := os.Remove(absPath); err != nil {
			return nil, fmt.Errorf("%w: %v", common.ErrDirectoryNotEmpty, err)
		}
	}

	return mcp.TextResult(fmt.Sprintf("Successfully deleted directory %s", absPath)), nil
}

func (s *Server) fileInfoTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "file_info",
		Description: "Get file metadata (size, permissions, timestamps)",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"path": mcp.StringProperty("Absolute path to file"),
			},
			[]string{"path"},
		),
		Handler: s.handleFileInfo,
	}
}

func (s *Server) handleFileInfo(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	path, err := mcp.GetStringParam(params, "path", true)
	if err != nil {
		return nil, err
	}

	absPath, err := s.validator.ResolvePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Lstat(absPath)
	if err != nil {
		return nil, err
	}

	fileInfo := FileInfo{
		Name:        info.Name(),
		Path:        absPath,
		SizeBytes:   info.Size(),
		Permissions: fmt.Sprintf("%04o", info.Mode().Perm()),
		IsDirectory: info.IsDir(),
		IsSymlink:   info.Mode()&os.ModeSymlink != 0,
		ModifiedAt:  info.ModTime(),
	}

	return mcp.JSONResult(fileInfo)
}

func (s *Server) searchFilesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "search_files",
		Description: "Search for files by name pattern",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"directory": mcp.StringProperty("Directory to search in"),
				"pattern":   mcp.StringProperty("Glob pattern to match"),
				"max_depth": mcp.IntProperty("Maximum depth to search"),
			},
			[]string{"directory", "pattern"},
		),
		Handler: s.handleSearchFiles,
	}
}

func (s *Server) handleSearchFiles(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	directory, err := mcp.GetStringParam(params, "directory", true)
	if err != nil {
		return nil, err
	}

	pattern, err := mcp.GetStringParam(params, "pattern", true)
	if err != nil {
		return nil, err
	}

	maxDepth, _ := mcp.GetIntParam(params, "max_depth", false, 10)

	absDir, err := s.validator.ResolvePath(directory)
	if err != nil {
		return nil, err
	}

	var matches []string
	baseDepth := strings.Count(absDir, string(os.PathSeparator))

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		currentDepth := strings.Count(path, string(os.PathSeparator)) - baseDepth
		if currentDepth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		matched, err := filepath.Match(pattern, info.Name())
		if err != nil {
			return nil
		}

		if matched {
			matches = append(matches, path)
		}

		if len(matches) >= 1000 {
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return mcp.JSONResult(map[string]interface{}{
		"directory": absDir,
		"pattern":   pattern,
		"matches":   matches,
		"count":     len(matches),
	})
}

func (s *Server) grepTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "grep",
		Description: "Search for content within files",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"directory":      mcp.StringProperty("Directory to search in"),
				"pattern":        mcp.StringProperty("Regex pattern to search"),
				"file_pattern":   mcp.StringProperty("File name pattern filter"),
				"case_sensitive": mcp.BoolProperty("Case sensitive search"),
			},
			[]string{"directory", "pattern"},
		),
		Handler: s.handleGrep,
	}
}

func (s *Server) handleGrep(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	directory, err := mcp.GetStringParam(params, "directory", true)
	if err != nil {
		return nil, err
	}

	pattern, err := mcp.GetStringParam(params, "pattern", true)
	if err != nil {
		return nil, err
	}

	filePattern, _ := mcp.GetStringParam(params, "file_pattern", false)
	caseSensitive, _ := mcp.GetBoolParam(params, "case_sensitive", true)

	absDir, err := s.validator.ResolvePath(directory)
	if err != nil {
		return nil, err
	}

	if !caseSensitive {
		pattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var matches []GrepMatch
	maxMatches := 500

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if filePattern != "" {
			matched, _ := filepath.Match(filePattern, info.Name())
			if !matched {
				return nil
			}
		}

		if info.Size() > 10*1024*1024 {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			if re.MatchString(line) {
				matches = append(matches, GrepMatch{
					File:       path,
					LineNumber: lineNum,
					Line:       line,
				})

				if len(matches) >= maxMatches {
					return filepath.SkipAll
				}
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, err
	}

	return mcp.JSONResult(map[string]interface{}{
		"directory": absDir,
		"pattern":   pattern,
		"matches":   matches,
		"count":     len(matches),
		"truncated": len(matches) >= maxMatches,
	})
}
