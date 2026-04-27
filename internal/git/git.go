package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Client struct {
	Runner Runner
}

type EnvRunner interface {
	RunWithEnv(ctx context.Context, repo string, env map[string]string, args ...string) (Result, error)
}

func NewClient(r Runner) *Client {
	if r == nil {
		r = NewExecRunner()
	}
	return &Client{Runner: r}
}

func (c *Client) run(ctx context.Context, repo string, args ...string) (Result, error) {
	if c == nil || c.Runner == nil {
		return Result{}, errors.New("git runner is not configured")
	}
	return c.Runner.Run(ctx, repo, args...)
}

func (c *Client) runWithEnv(ctx context.Context, repo string, env map[string]string, args ...string) (Result, error) {
	if c == nil || c.Runner == nil {
		return Result{}, errors.New("git runner is not configured")
	}
	if runner, ok := c.Runner.(EnvRunner); ok {
		return runner.RunWithEnv(ctx, repo, env, args...)
	}
	return c.Runner.Run(ctx, repo, args...)
}

func (c *Client) Version(ctx context.Context) (string, error) {
	res, err := c.run(ctx, "", "--version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Stdout), nil
}

func (c *Client) EnsureInstalled(ctx context.Context) error {
	_, err := c.run(ctx, "", "--version")
	return err
}

func (c *Client) IsRepo(ctx context.Context, repo string) (bool, error) {
	res, err := c.run(ctx, repo, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(res.Stdout) == "true", nil
}

func (c *Client) RepoRoot(ctx context.Context, repo string) (string, error) {
	res, err := c.run(ctx, repo, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Stdout), nil
}

func (c *Client) CurrentBranch(ctx context.Context, repo string) (string, error) {
	res, err := c.run(ctx, repo, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Stdout), nil
}

func (c *Client) LocalBranches(ctx context.Context, repo string) ([]string, error) {
	res, err := c.run(ctx, repo, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, err
	}
	return splitNonEmptyLines(res.Stdout), nil
}

func (c *Client) LocalTags(ctx context.Context, repo string) ([]string, error) {
	res, err := c.run(ctx, repo, "tag", "--list")
	if err != nil {
		return nil, err
	}
	return splitNonEmptyLines(res.Stdout), nil
}

func (c *Client) RemoteNames(ctx context.Context, repo string) ([]string, error) {
	res, err := c.run(ctx, repo, "remote")
	if err != nil {
		return nil, err
	}
	return splitNonEmptyLines(res.Stdout), nil
}

func (c *Client) RemoteMap(ctx context.Context, repo string) (map[string]string, error) {
	names, err := c.RemoteNames(ctx, repo)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(names))
	for _, name := range names {
		url, err := c.RemoteURL(ctx, repo, name)
		if err != nil {
			continue
		}
		out[name] = url
	}
	return out, nil
}

func (c *Client) RemoteURL(ctx context.Context, repo, name string) (string, error) {
	res, err := c.run(ctx, repo, "remote", "get-url", name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Stdout), nil
}

func (c *Client) AddRemote(ctx context.Context, repo, name, url string) error {
	_, err := c.run(ctx, repo, "remote", "add", name, url)
	return err
}

func (c *Client) SetRemoteURL(ctx context.Context, repo, name, url string) error {
	_, err := c.run(ctx, repo, "remote", "set-url", name, url)
	return err
}

func (c *Client) WorktreeClean(ctx context.Context, repo string) (bool, error) {
	res, err := c.run(ctx, repo, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(res.Stdout) == "", nil
}

func (c *Client) HasCommits(ctx context.Context, repo string) (bool, error) {
	_, err := c.run(ctx, repo, "rev-parse", "--verify", "HEAD")
	if err != nil {
		if isExitNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *Client) FetchAllPrune(ctx context.Context, repo string) error {
	_, err := c.run(ctx, repo, "fetch", "--all", "--prune")
	return err
}

func (c *Client) FetchAllPruneWithEnv(ctx context.Context, repo string, env map[string]string) error {
	_, err := c.runWithEnv(ctx, repo, env, "fetch", "--all", "--prune")
	return err
}

func (c *Client) PullFFOnly(ctx context.Context, repo string) error {
	_, err := c.run(ctx, repo, "pull", "--ff-only")
	return err
}

func (c *Client) PullFFOnlyWithEnv(ctx context.Context, repo string, env map[string]string) error {
	_, err := c.runWithEnv(ctx, repo, env, "pull", "--ff-only")
	return err
}

func (c *Client) PushAll(ctx context.Context, repo, remote string) error {
	_, err := c.run(ctx, repo, "push", remote, "--all")
	return err
}

func (c *Client) PushAllPrune(ctx context.Context, repo, remote string) error {
	_, err := c.run(ctx, repo, "push", "--prune", remote, "--all")
	return err
}

func (c *Client) PushTags(ctx context.Context, repo, remote string) error {
	_, err := c.run(ctx, repo, "push", remote, "--tags")
	return err
}

func (c *Client) PushTagsPrune(ctx context.Context, repo, remote string) error {
	_, err := c.run(ctx, repo, "push", "--prune", remote, "--tags")
	return err
}

func (c *Client) PushMirror(ctx context.Context, repo, remote string) error {
	_, err := c.run(ctx, repo, "push", remote, "--mirror")
	return err
}

func (c *Client) PushNotes(ctx context.Context, repo, remote string) error {
	_, err := c.run(ctx, repo, "push", remote, "refs/notes/*")
	return err
}

func (c *Client) PushRefspecs(ctx context.Context, repo, remote string, refspecs []string) error {
	args := append([]string{"push", remote}, refspecs...)
	_, err := c.run(ctx, repo, args...)
	return err
}

func (c *Client) PushRefspecsPrune(ctx context.Context, repo, remote string, refspecs []string) error {
	args := append([]string{"push", "--prune", remote}, refspecs...)
	_, err := c.run(ctx, repo, args...)
	return err
}

func (c *Client) BundleCreate(ctx context.Context, repo, output string) error {
	_, err := c.run(ctx, repo, "bundle", "create", output, "--all")
	return err
}

func isExitNotFound(err error) bool {
	if err == nil {
		return false
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() != 0
	}
	return true
}

func splitNonEmptyLines(s string) []string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func GitErrorHint(err error, stderr string) string {
	text := strings.ToLower(strings.TrimSpace(stderr + " " + err.Error()))
	switch {
	case strings.Contains(text, "authentication"), strings.Contains(text, "permission denied"), strings.Contains(text, "could not read from remote repository"):
		return "authentication failed or permission denied"
	case strings.Contains(text, "repository not found"), strings.Contains(text, "not found"), strings.Contains(text, "could not read from repository"), strings.Contains(text, "does not appear to be a git repository"):
		return "remote repository not found"
	case strings.Contains(text, "temporary failure in name resolution"), strings.Contains(text, "could not resolve host"), strings.Contains(text, "could not resolve hostname"), strings.Contains(text, "no such host"), strings.Contains(text, "network is unreachable"), strings.Contains(text, "connection timed out"), strings.Contains(text, "failed to connect"):
		return "network or DNS failure"
	case strings.Contains(text, "non-fast-forward"), strings.Contains(text, "fetch first"), strings.Contains(text, "rejected"):
		return "non-fast-forward rejection"
	case strings.Contains(text, "no commits yet on branch"), strings.Contains(text, "ambiguous argument 'head'"), strings.Contains(text, "unknown revision or path not in the working tree"):
		return "repository has no commits"
	case strings.Contains(text, "bad git repository"), strings.Contains(text, "not a git repository"):
		return "invalid git repository"
	default:
		return ""
	}
}

func WrapGitError(operation, remote string, err error, stderr string) error {
	if err == nil {
		return nil
	}
	hint := GitErrorHint(err, stderr)
	if hint != "" {
		return fmt.Errorf("%s to %s failed: %s. Original error: %w", operation, remote, hint, err)
	}
	return fmt.Errorf("%s to %s failed. Original error: %w", operation, remote, err)
}
