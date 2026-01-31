package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/local-mcps/dev-mcps/pkg/mcp"
)

func (s *Server) runGit(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %s", err.Error(), stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (s *Server) gitStatusTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_status",
		Description: "Get repository status",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path": mcp.StringProperty("Path to repository"),
			},
			[]string{"repo_path"},
		),
		Handler: s.handleGitStatus,
	}
}

func (s *Server) handleGitStatus(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	branch, _ := s.runGit(repoPath, "rev-parse", "--abbrev-ref", "HEAD")

	status, err := s.runGit(repoPath, "status", "--porcelain")
	if err != nil {
		return nil, err
	}

	var staged, modified, untracked, deleted []string
	for _, line := range strings.Split(status, "\n") {
		if len(line) < 3 {
			continue
		}
		indexStatus := line[0]
		workTreeStatus := line[1]
		file := strings.TrimSpace(line[3:])

		if indexStatus == 'A' || indexStatus == 'M' || indexStatus == 'D' || indexStatus == 'R' {
			staged = append(staged, file)
		}
		if workTreeStatus == 'M' {
			modified = append(modified, file)
		}
		if workTreeStatus == 'D' {
			deleted = append(deleted, file)
		}
		if indexStatus == '?' && workTreeStatus == '?' {
			untracked = append(untracked, file)
		}
	}

	ahead, behind := 0, 0
	if tracking, err := s.runGit(repoPath, "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); err == nil {
		parts := strings.Fields(tracking)
		if len(parts) == 2 {
			ahead, _ = strconv.Atoi(parts[0])
			behind, _ = strconv.Atoi(parts[1])
		}
	}

	return mcp.JSONResult(map[string]interface{}{
		"branch":          branch,
		"is_clean":        len(staged) == 0 && len(modified) == 0 && len(untracked) == 0,
		"staged_files":    staged,
		"modified_files":  modified,
		"untracked_files": untracked,
		"deleted_files":   deleted,
		"ahead":           ahead,
		"behind":          behind,
	})
}

func (s *Server) gitLogTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_log",
		Description: "Get commit history",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path":   mcp.StringProperty("Path to repository"),
				"max_commits": mcp.IntProperty("Maximum commits to return"),
				"branch":      mcp.StringProperty("Branch to get log from"),
			},
			[]string{"repo_path"},
		),
		Handler: s.handleGitLog,
	}
}

func (s *Server) handleGitLog(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	maxCommits, _ := mcp.GetIntParam(params, "max_commits", false, 20)
	branch, _ := mcp.GetStringParam(params, "branch", false)

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	args := []string{"log", fmt.Sprintf("-n%d", maxCommits), "--format=%H|%h|%an <%ae>|%aI|%s"}
	if branch != "" {
		args = append(args, branch)
	}

	output, err := s.runGit(repoPath, args...)
	if err != nil {
		return nil, err
	}

	var commits []map[string]interface{}
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) == 5 {
			commits = append(commits, map[string]interface{}{
				"hash":       parts[0],
				"short_hash": parts[1],
				"author":     parts[2],
				"date":       parts[3],
				"message":    parts[4],
			})
		}
	}

	return mcp.JSONResult(map[string]interface{}{
		"commits":     commits,
		"total_count": len(commits),
	})
}

func (s *Server) gitDiffTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_diff",
		Description: "Get diff of changes",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path": mcp.StringProperty("Path to repository"),
				"staged":    mcp.BoolProperty("Show staged changes only"),
				"commit":    mcp.StringProperty("Show diff for specific commit"),
			},
			[]string{"repo_path"},
		),
		Handler: s.handleGitDiff,
	}
}

func (s *Server) handleGitDiff(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	staged, _ := mcp.GetBoolParam(params, "staged", false)
	commit, _ := mcp.GetStringParam(params, "commit", false)

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	args := []string{"diff", "--stat"}
	if staged {
		args = append(args, "--cached")
	}
	if commit != "" {
		args = []string{"show", "--stat", commit}
	}

	statOutput, _ := s.runGit(repoPath, args...)

	args = []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	if commit != "" {
		args = []string{"show", commit}
	}

	diffOutput, err := s.runGit(repoPath, args...)
	if err != nil {
		return nil, err
	}

	if len(diffOutput) > 100000 {
		diffOutput = diffOutput[:100000] + "\n... (truncated)"
	}

	return mcp.JSONResult(map[string]interface{}{
		"diff":  diffOutput,
		"stats": statOutput,
	})
}

func (s *Server) gitBranchListTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_branch_list",
		Description: "List branches",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path": mcp.StringProperty("Path to repository"),
				"remote":    mcp.BoolProperty("Include remote branches"),
			},
			[]string{"repo_path"},
		),
		Handler: s.handleGitBranchList,
	}
}

func (s *Server) handleGitBranchList(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	includeRemote, _ := mcp.GetBoolParam(params, "remote", false)

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	currentBranch, _ := s.runGit(repoPath, "rev-parse", "--abbrev-ref", "HEAD")

	localOutput, err := s.runGit(repoPath, "branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	localBranches := strings.Split(strings.TrimSpace(localOutput), "\n")

	var remoteBranches []string
	if includeRemote {
		remoteOutput, _ := s.runGit(repoPath, "branch", "-r", "--format=%(refname:short)")
		if remoteOutput != "" {
			remoteBranches = strings.Split(strings.TrimSpace(remoteOutput), "\n")
		}
	}

	return mcp.JSONResult(map[string]interface{}{
		"current_branch":   currentBranch,
		"local_branches":   localBranches,
		"remote_branches":  remoteBranches,
		"total_count":      len(localBranches) + len(remoteBranches),
	})
}

func (s *Server) gitBranchCreateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_branch_create",
		Description: "Create a new branch",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path":   mcp.StringProperty("Path to repository"),
				"branch_name": mcp.StringProperty("Name for new branch"),
				"start_point": mcp.StringProperty("Starting commit/branch"),
			},
			[]string{"repo_path", "branch_name"},
		),
		Handler: s.handleGitBranchCreate,
	}
}

func (s *Server) handleGitBranchCreate(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	branchName, err := mcp.GetStringParam(params, "branch_name", true)
	if err != nil {
		return nil, err
	}

	startPoint, _ := mcp.GetStringParam(params, "start_point", false)

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	args := []string{"branch", branchName}
	if startPoint != "" {
		args = append(args, startPoint)
	}

	if _, err := s.runGit(repoPath, args...); err != nil {
		return nil, err
	}

	return mcp.TextResult(fmt.Sprintf("Created branch %s", branchName)), nil
}

func (s *Server) gitCheckoutTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_checkout",
		Description: "Checkout a branch or commit",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path": mcp.StringProperty("Path to repository"),
				"ref":       mcp.StringProperty("Branch, tag, or commit to checkout"),
			},
			[]string{"repo_path", "ref"},
		),
		Handler: s.handleGitCheckout,
	}
}

func (s *Server) handleGitCheckout(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	ref, err := mcp.GetStringParam(params, "ref", true)
	if err != nil {
		return nil, err
	}

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	if _, err := s.runGit(repoPath, "checkout", ref); err != nil {
		return nil, err
	}

	return mcp.TextResult(fmt.Sprintf("Checked out %s", ref)), nil
}

func (s *Server) gitAddTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_add",
		Description: "Stage files for commit",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path": mcp.StringProperty("Path to repository"),
				"paths":     mcp.ArrayProperty("string", "Files/directories to stage"),
			},
			[]string{"repo_path", "paths"},
		),
		Handler: s.handleGitAdd,
	}
}

func (s *Server) handleGitAdd(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	paths, err := mcp.GetStringArrayParam(params, "paths", true)
	if err != nil {
		return nil, err
	}

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	args := append([]string{"add"}, paths...)
	if _, err := s.runGit(repoPath, args...); err != nil {
		return nil, err
	}

	return mcp.TextResult(fmt.Sprintf("Staged %d file(s)", len(paths))), nil
}

func (s *Server) gitCommitTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_commit",
		Description: "Create a commit",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path": mcp.StringProperty("Path to repository"),
				"message":   mcp.StringProperty("Commit message"),
				"author":    mcp.StringProperty("Author override (Name <email>)"),
			},
			[]string{"repo_path", "message"},
		),
		Handler: s.handleGitCommit,
	}
}

func (s *Server) handleGitCommit(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	message, err := mcp.GetStringParam(params, "message", true)
	if err != nil {
		return nil, err
	}

	author, _ := mcp.GetStringParam(params, "author", false)

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	args := []string{"commit", "-m", message}
	if author != "" {
		args = append(args, "--author", author)
	}

	output, err := s.runGit(repoPath, args...)
	if err != nil {
		return nil, err
	}

	hash, _ := s.runGit(repoPath, "rev-parse", "--short", "HEAD")

	return mcp.JSONResult(map[string]interface{}{
		"hash":    hash,
		"message": message,
		"output":  output,
	})
}

func (s *Server) gitPushTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_push",
		Description: "Push commits to remote",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path": mcp.StringProperty("Path to repository"),
				"remote":    mcp.StringProperty("Remote name (default: origin)"),
				"branch":    mcp.StringProperty("Branch to push"),
				"force":     mcp.BoolProperty("Force push"),
			},
			[]string{"repo_path"},
		),
		Handler: s.handleGitPush,
	}
}

func (s *Server) handleGitPush(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	remote, _ := mcp.GetStringParam(params, "remote", false)
	branch, _ := mcp.GetStringParam(params, "branch", false)
	force, _ := mcp.GetBoolParam(params, "force", false)

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	if !s.config.AllowPush {
		return nil, fmt.Errorf("push is disabled in configuration")
	}

	if force && !s.config.AllowForcePush {
		return nil, fmt.Errorf("force push is disabled in configuration")
	}

	args := []string{"push"}
	if force {
		args = append(args, "--force")
	}
	if remote != "" {
		args = append(args, remote)
	}
	if branch != "" {
		args = append(args, branch)
	}

	output, err := s.runGit(repoPath, args...)
	if err != nil {
		return nil, err
	}

	return mcp.TextResult(fmt.Sprintf("Push completed: %s", output)), nil
}

func (s *Server) gitPullTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_pull",
		Description: "Pull changes from remote",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path": mcp.StringProperty("Path to repository"),
				"remote":    mcp.StringProperty("Remote name (default: origin)"),
				"branch":    mcp.StringProperty("Branch to pull"),
			},
			[]string{"repo_path"},
		),
		Handler: s.handleGitPull,
	}
}

func (s *Server) handleGitPull(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	remote, _ := mcp.GetStringParam(params, "remote", false)
	branch, _ := mcp.GetStringParam(params, "branch", false)

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	args := []string{"pull"}
	if remote != "" {
		args = append(args, remote)
	}
	if branch != "" {
		args = append(args, branch)
	}

	output, err := s.runGit(repoPath, args...)
	if err != nil {
		return nil, err
	}

	return mcp.TextResult(fmt.Sprintf("Pull completed: %s", output)), nil
}

func (s *Server) gitCloneTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_clone",
		Description: "Clone a repository",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"url":         mcp.StringProperty("Repository URL"),
				"destination": mcp.StringProperty("Local destination path"),
				"branch":      mcp.StringProperty("Branch to checkout"),
				"depth":       mcp.IntProperty("Shallow clone depth"),
			},
			[]string{"url", "destination"},
		),
		Handler: s.handleGitClone,
	}
}

func (s *Server) handleGitClone(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	url, err := mcp.GetStringParam(params, "url", true)
	if err != nil {
		return nil, err
	}

	destination, err := mcp.GetStringParam(params, "destination", true)
	if err != nil {
		return nil, err
	}

	branch, _ := mcp.GetStringParam(params, "branch", false)
	depth, _ := mcp.GetIntParam(params, "depth", false, 0)

	args := []string{"clone"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	if depth > 0 {
		args = append(args, "--depth", strconv.Itoa(depth))
	}
	args = append(args, url, destination)

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), stderr.String())
	}

	return mcp.TextResult(fmt.Sprintf("Cloned %s to %s", url, destination)), nil
}

func (s *Server) gitStashTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_stash",
		Description: "Stash or apply stashed changes",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path": mcp.StringProperty("Path to repository"),
				"action":    mcp.StringProperty("Action: push, pop, list, drop"),
			},
			[]string{"repo_path", "action"},
		),
		Handler: s.handleGitStash,
	}
}

func (s *Server) handleGitStash(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	action, err := mcp.GetStringParam(params, "action", true)
	if err != nil {
		return nil, err
	}

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	validActions := map[string]bool{"push": true, "pop": true, "list": true, "drop": true}
	if !validActions[action] {
		return nil, fmt.Errorf("invalid action: %s (must be push, pop, list, or drop)", action)
	}

	output, err := s.runGit(repoPath, "stash", action)
	if err != nil {
		return nil, err
	}

	return mcp.TextResult(output), nil
}

func (s *Server) gitBlameTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_blame",
		Description: "Show who changed each line",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path": mcp.StringProperty("Path to repository"),
				"file_path": mcp.StringProperty("File to blame"),
			},
			[]string{"repo_path", "file_path"},
		),
		Handler: s.handleGitBlame,
	}
}

func (s *Server) handleGitBlame(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	filePath, err := mcp.GetStringParam(params, "file_path", true)
	if err != nil {
		return nil, err
	}

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	output, err := s.runGit(repoPath, "blame", "--line-porcelain", filePath)
	if err != nil {
		return nil, err
	}

	if len(output) > 100000 {
		output = output[:100000] + "\n... (truncated)"
	}

	return mcp.TextResult(output), nil
}

func (s *Server) gitShowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "git_show",
		Description: "Show commit details",
		InputSchema: mcp.BuildInputSchema(
			map[string]interface{}{
				"repo_path": mcp.StringProperty("Path to repository"),
				"commit":    mcp.StringProperty("Commit hash"),
			},
			[]string{"repo_path", "commit"},
		),
		Handler: s.handleGitShow,
	}
}

func (s *Server) handleGitShow(ctx context.Context, params map[string]interface{}) (*mcp.ToolResult, error) {
	repoPath, err := mcp.GetStringParam(params, "repo_path", true)
	if err != nil {
		return nil, err
	}

	commit, err := mcp.GetStringParam(params, "commit", true)
	if err != nil {
		return nil, err
	}

	if err := s.validator.ValidatePath(repoPath); err != nil {
		return nil, err
	}

	output, err := s.runGit(repoPath, "show", "--stat", commit)
	if err != nil {
		return nil, err
	}

	if len(output) > 100000 {
		output = output[:100000] + "\n... (truncated)"
	}

	return mcp.TextResult(output), nil
}
