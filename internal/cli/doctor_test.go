package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gitpkg "git-ark/internal/git"
)

type doctorRunner struct{}

func (d *doctorRunner) Run(ctx context.Context, repo string, args ...string) (gitpkg.Result, error) {
	key := strings.Join(args, " ")
	switch key {
	case "rev-parse --is-inside-work-tree":
		return gitpkg.Result{Stdout: "true\n"}, nil
	case "branch --show-current":
		return gitpkg.Result{Stdout: "main\n"}, nil
	case "status --porcelain":
		return gitpkg.Result{Stdout: "\n"}, nil
	case "for-each-ref --format=%(refname:short) refs/heads":
		return gitpkg.Result{Stdout: "main\n"}, nil
	case "tag --list":
		return gitpkg.Result{Stdout: "v1.0.0\n"}, nil
	case "remote":
		return gitpkg.Result{Stdout: "github\n"}, nil
	case "remote get-url github":
		return gitpkg.Result{Stdout: "git@gitlab.com:example/example-backup.git\n"}, nil
	case "ls-remote --heads --tags git@gitlab.com:example/example-backup.git":
		return gitpkg.Result{Stdout: "a1b2c3d4\trefs/heads/main\n"}, nil
	default:
		return gitpkg.Result{}, os.ErrInvalid
	}
}

func TestDoctorShowsProviderDiagnostics(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(repo, "git-ark.yml")
	configYAML := []byte("version: 1\nrepo: " + repo + "\nremotes:\n  github:\n    url: git@gitlab.com:example/example-backup.git\n    provider: github\n")
	if err := os.WriteFile(configPath, configYAML, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newDoctorCommand(deps{git: gitpkg.NewClient(&doctorRunner{})})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--config", configPath, "--repo", repo})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Provider diagnostics:") {
		t.Fatalf("expected provider diagnostics section, got %s", out)
	}
	if !strings.Contains(out, "provider=GitHub") {
		t.Fatalf("expected GitHub provider detail, got %s", out)
	}
	if !strings.Contains(out, "warning: provider GitHub is configured, but the URL host gitlab.com looks like GitLab") {
		t.Fatalf("expected provider mismatch warning, got %s", out)
	}
	if !strings.Contains(stderr.String(), "GitLab") {
		t.Fatalf("expected provider warning on stderr, got %s", stderr.String())
	}
}
