package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Context captures the git repository state at the time of an action.
type Context struct {
	Repository string `json:"repository,omitempty"`
	Branch     string `json:"branch,omitempty"`
	Commit     string `json:"commit,omitempty"`
	Author     string `json:"author,omitempty"`
	Committer  string `json:"committer,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
	Message    string `json:"message,omitempty"`
}

// CaptureContext runs git commands in the given directory and returns the context.
// If git is not available or the directory is not a repository, it returns
// an empty Context and a nil error.
func CaptureContext(dir string) (*Context, error) {
	ctx := &Context{}

	// Check if we're in a git repository
	repoCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	repoCmd.Dir = dir
	repoOut, err := repoCmd.Output()
	if err != nil {
		// Not a git repo or git not installed — return empty context
		return ctx, nil
	}
	ctx.Repository = strings.TrimSpace(string(repoOut))

	// Get current branch
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = dir
	branchOut, err := branchCmd.Output()
	if err == nil {
		ctx.Branch = strings.TrimSpace(string(branchOut))
	}

	// Get commit hash
	commitCmd := exec.Command("git", "rev-parse", "HEAD")
	commitCmd.Dir = dir
	commitOut, err := commitCmd.Output()
	if err == nil {
		ctx.Commit = strings.TrimSpace(string(commitOut))
	}

	// Get author
	authorCmd := exec.Command("git", "log", "-1", "--format=%an <%ae>")
	authorCmd.Dir = dir
	authorOut, err := authorCmd.Output()
	if err == nil {
		ctx.Author = strings.TrimSpace(string(authorOut))
	}

	// Get committer
	committerCmd := exec.Command("git", "log", "-1", "--format=%cn <%ce>")
	committerCmd.Dir = dir
	committerOut, err := committerCmd.Output()
	if err == nil {
		ctx.Committer = strings.TrimSpace(string(committerOut))
	}

	// Get commit timestamp
	timestampCmd := exec.Command("git", "log", "-1", "--format=%cI")
	timestampCmd.Dir = dir
	timestampOut, err := timestampCmd.Output()
	if err == nil {
		ctx.Timestamp = strings.TrimSpace(string(timestampOut))
	}

	// Get commit subject/message
	msgCmd := exec.Command("git", "log", "-1", "--format=%s")
	msgCmd.Dir = dir
	msgOut, err := msgCmd.Output()
	if err == nil {
		ctx.Message = strings.TrimSpace(string(msgOut))
	}

	return ctx, nil
}

// GetDiff returns the current working directory git diff.
func GetDiff(dir string) (string, error) {
	diffCmd := exec.Command("git", "diff")
	diffCmd.Dir = dir
	output, err := diffCmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting git diff: %w", err)
	}
	return string(output), nil
}

// GetCommitDiff returns the diff introduced by the specified commit.
func GetCommitDiff(dir, commit string) (string, error) {
	diffCmd := exec.Command("git", "show", "--format=", commit)
	diffCmd.Dir = dir
	output, err := diffCmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting commit diff for %s: %w", commit, err)
	}
	return string(output), nil
}

// DefaultReceiptsDir returns the default directory to store receipts inside a repo.
func DefaultReceiptsDir(repoRoot string) string {
	return filepath.Join(repoRoot, ".agent-proof", "receipts")
}

// InstallHook installs a git post-commit hook that triggers agent-proof recording.
func InstallHook(dir string) (string, error) {
	ctx, err := CaptureContext(dir)
	if err != nil || ctx.Repository == "" {
		return "", fmt.Errorf("directory %s is not inside a git repository", dir)
	}

	hooksDir := filepath.Join(ctx.Repository, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return "", fmt.Errorf("creating hooks dir: %w", err)
	}

	hookFile := filepath.Join(hooksDir, "post-commit")
	hookContent := `#!/bin/sh
# agent-proof auto-record hook
if command -v agent-proof >/dev/null 2>&1; then
  if [ -n "$AGENT_NAME" ]; then
    COMMIT=$(git rev-parse HEAD)
    MSG=$(git log -1 --format=%s)
    echo "Recording agent-proof receipt for commit $COMMIT..."
    agent-proof record \
      --agent "${AGENT_NAME:-ai-agent}" \
      --intent "$MSG" \
      --action "git.commit" \
      --output ".agent-proof/receipts/${COMMIT}.json" \
      --private-key "${AGENT_PROOF_KEY:-agent-proof.key}" 2>/dev/null || true
  fi
fi
`
	if err := os.WriteFile(hookFile, []byte(hookContent), 0755); err != nil {
		return "", fmt.Errorf("writing post-commit hook: %w", err)
	}

	return hookFile, nil
}
