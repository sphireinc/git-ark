package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git-ark/internal/backup"
	"git-ark/internal/config"
	gitpkg "git-ark/internal/git"
)

type statusRunner struct{}

func (s *statusRunner) Run(ctx context.Context, repo string, args ...string) (gitpkg.Result, error) {
	key := strings.Join(args, " ")
	switch key {
	case "branch --show-current":
		return gitpkg.Result{Stdout: "main\n"}, nil
	case "status --porcelain":
		return gitpkg.Result{Stdout: "\n"}, nil
	case "for-each-ref --format=%(refname:short) refs/heads":
		return gitpkg.Result{Stdout: "main\nrelease/1.0\n"}, nil
	case "tag --list":
		return gitpkg.Result{Stdout: "v1.0.0\n"}, nil
	case "remote":
		return gitpkg.Result{Stdout: "github\n"}, nil
	case "remote get-url github":
		return gitpkg.Result{Stdout: "git@github.com:example/example-backup.git\n"}, nil
	default:
		return gitpkg.Result{}, os.ErrInvalid
	}
}

func TestStatusShowsMetadataHistory(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Repo = repo
	cfg.Remotes = map[string]config.RemoteConfig{
		"github": {URL: "git@github.com:example/example-backup.git", Enabled: true, Required: true, Provider: "github"},
	}
	cfgPath := filepath.Join(repo, "git-ark.yml")
	configYAML := []byte("version: 1\nrepo: " + repo + "\nremotes:\n  github:\n    url: git@github.com:example/example-backup.git\n    provider: github\n")
	if err := os.WriteFile(cfgPath, configYAML, 0o644); err != nil {
		t.Fatal(err)
	}
	svc := backup.New(gitpkg.NewClient(&statusRunner{}))
	reports := []backup.Report{
		{
			Repo:       repo,
			Mode:       "safe",
			StartedAt:  time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC),
			FinishedAt: time.Date(2026, 4, 27, 10, 1, 0, 0, time.UTC),
			DurationMS: 60000,
			Success:    true,
		},
		{
			Repo:       repo,
			Mode:       "mirror",
			StartedAt:  time.Date(2026, 4, 27, 11, 0, 0, 0, time.UTC),
			FinishedAt: time.Date(2026, 4, 27, 11, 2, 0, 0, time.UTC),
			DurationMS: 120000,
			Success:    true,
		},
	}
	for _, report := range reports {
		if err := svc.WriteMetadata(cfg, report, repo); err != nil {
			t.Fatal(err)
		}
	}
	cmd := newStatusCommand(deps{git: gitpkg.NewClient(&statusRunner{})})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Recorded backups: 2") {
		t.Fatalf("status output missing history count: %s", out)
	}
	if !strings.Contains(out, "Latest backup mode: mirror") {
		t.Fatalf("status output missing latest mode: %s", out)
	}
	if !strings.Contains(out, "Recent backups:") {
		t.Fatalf("status output missing history list: %s", out)
	}
	if !strings.Contains(out, "mode=mirror success") || !strings.Contains(out, "mode=safe success") {
		t.Fatalf("status output missing recent entries: %s", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output: %s", stderr.String())
	}
}
