package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Runner interface {
	Run(ctx context.Context, repo string, args ...string) (Result, error)
}

type Result struct {
	Args     []string
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

type ExecRunner struct {
	GitBinary      string
	LookPath       func(string) (string, error)
	CommandFactory func(ctx context.Context, name string, args ...string) *exec.Cmd
}

func NewExecRunner() *ExecRunner {
	return &ExecRunner{
		GitBinary: "git",
	}
}

func (r *ExecRunner) Run(ctx context.Context, repo string, args ...string) (Result, error) {
	return r.run(ctx, repo, nil, args...)
}

func (r *ExecRunner) RunWithEnv(ctx context.Context, repo string, env map[string]string, args ...string) (Result, error) {
	return r.run(ctx, repo, env, args...)
}

func (r *ExecRunner) run(ctx context.Context, repo string, env map[string]string, args ...string) (Result, error) {
	binary := r.GitBinary
	if binary == "" {
		binary = "git"
	}
	lookPath := r.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	resolved, err := lookPath(binary)
	if err != nil {
		return Result{Args: append([]string{}, args...)}, fmt.Errorf("git binary not found: %w", err)
	}
	fullArgs := make([]string, 0, len(args)+2)
	if repo != "" {
		fullArgs = append(fullArgs, "-C", repo)
	}
	fullArgs = append(fullArgs, args...)
	start := time.Now()
	cmdFactory := r.CommandFactory
	if cmdFactory == nil {
		cmdFactory = exec.CommandContext
	}
	cmd := cmdFactory(ctx, resolved, fullArgs...)
	if len(env) > 0 {
		cmd.Env = mergeEnv(os.Environ(), env)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	result := Result{
		Args:     append([]string{}, fullArgs...),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}
	if runErr != nil {
		if exitErr := new(exec.ExitError); errors.As(runErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return result, runErr
		}
		return result, runErr
	}
	result.ExitCode = 0
	return result, nil
}

func mergeEnv(base []string, overrides map[string]string) []string {
	out := make([]string, 0, len(base)+len(overrides))
	index := make(map[string]int, len(base))
	for _, kv := range base {
		key, _, ok := strings.Cut(kv, "=")
		if !ok {
			out = append(out, kv)
			continue
		}
		index[key] = len(out)
		out = append(out, kv)
	}
	for key, value := range overrides {
		if pos, ok := index[key]; ok {
			out[pos] = key + "=" + value
			continue
		}
		out = append(out, key+"="+value)
	}
	return out
}
