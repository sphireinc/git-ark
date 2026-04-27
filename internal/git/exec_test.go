package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestExecRunnerPassesArgsSafely(t *testing.T) {
	runner := &ExecRunner{
		GitBinary: "git",
		LookPath: func(string) (string, error) {
			return os.Args[0], nil
		},
		CommandFactory: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			capturedArgs = append([]string{}, args...)
			cmd := exec.CommandContext(ctx, name, append([]string{"-test.run=TestHelperProcess", "--"}, args...)...)
			cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "HELPER_STDERR=boom", "HELPER_EXIT_CODE=7")
			return cmd
		},
	}
	_, err := runner.Run(context.Background(), "repo path", "rev-parse", "value with spaces")
	if err == nil {
		t.Fatal("expected error")
	}
	want := []string{"-C", "repo path", "rev-parse", "value with spaces"}
	if !reflect.DeepEqual(capturedArgs, want) {
		t.Fatalf("args = %v want %v", capturedArgs, want)
	}
}

func TestExecRunnerMissingGitHandled(t *testing.T) {
	runner := &ExecRunner{
		GitBinary: "git",
		LookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
	}
	_, err := runner.Run(context.Background(), ".", "--version")
	if err == nil || !strings.Contains(err.Error(), "git binary not found") {
		t.Fatalf("expected missing git error, got %v", err)
	}
}

func TestExecRunnerPreservesStderr(t *testing.T) {
	runner := &ExecRunner{
		GitBinary: "git",
		LookPath: func(string) (string, error) {
			return os.Args[0], nil
		},
		CommandFactory: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			cmd := exec.CommandContext(ctx, name, append([]string{"-test.run=TestHelperProcess", "--"}, args...)...)
			cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "HELPER_STDERR=boom", "HELPER_EXIT_CODE=7")
			return cmd
		},
	}
	res, err := runner.Run(context.Background(), ".", "--version")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(res.Stderr, "boom") {
		t.Fatalf("stderr not preserved: %q", res.Stderr)
	}
}

var capturedArgs []string

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	if msg := os.Getenv("HELPER_STDERR"); msg != "" {
		_, _ = os.Stderr.WriteString(msg)
	}
	code := 0
	if v := os.Getenv("HELPER_EXIT_CODE"); v != "" {
		if v == "7" {
			code = 7
		}
	}
	os.Exit(code)
}
