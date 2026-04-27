package backup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git-ark/internal/config"
	gitpkg "git-ark/internal/git"
)

type fakeRunner struct {
	calls []string
	envs  []map[string]string
}

func (f *fakeRunner) Run(ctx context.Context, repo string, args ...string) (gitpkg.Result, error) {
	key := strings.Join(args, " ")
	f.calls = append(f.calls, key)
	switch key {
	case "rev-parse --is-inside-work-tree":
		return gitpkg.Result{Stdout: "true\n"}, nil
	case "rev-parse --show-toplevel":
		return gitpkg.Result{Stdout: repo + "\n"}, nil
	case "for-each-ref --format=%(refname:short) refs/heads":
		return gitpkg.Result{Stdout: "main\nrelease/1.0\nwip/foo\n"}, nil
	case "tag --list":
		return gitpkg.Result{Stdout: "v1.0.0\ntest-1\n"}, nil
	case "remote":
		return gitpkg.Result{Stdout: "github\n"}, nil
	case "remote get-url github":
		return gitpkg.Result{Stdout: "git@github.com:example/example-backup.git\n"}, nil
	case "fetch --all --prune":
		return gitpkg.Result{}, nil
	case "pull --ff-only":
		return gitpkg.Result{}, nil
	case "push github --all":
		return gitpkg.Result{}, nil
	case "push --prune github --all":
		return gitpkg.Result{}, nil
	case "push github --tags":
		return gitpkg.Result{}, nil
	case "push --prune github --tags":
		return gitpkg.Result{}, nil
	case "bundle create " + filepath.Join(repo, "backups", filepath.Base(repo)+"-20260427T120000Z.bundle") + " --all":
		return gitpkg.Result{}, nil
	default:
		if strings.HasPrefix(key, "push github refs/heads/") || strings.HasPrefix(key, "push github refs/tags/") || strings.HasPrefix(key, "push github refs/notes/*") {
			return gitpkg.Result{}, nil
		}
		if strings.HasPrefix(key, "push --prune github refs/heads/") || strings.HasPrefix(key, "push --prune github refs/tags/") || strings.HasPrefix(key, "push --prune github refs/notes/*") {
			return gitpkg.Result{}, nil
		}
		return gitpkg.Result{}, errors.New("unexpected command: " + key)
	}
}

func (f *fakeRunner) RunWithEnv(ctx context.Context, repo string, env map[string]string, args ...string) (gitpkg.Result, error) {
	if env != nil {
		copyEnv := make(map[string]string, len(env))
		for key, value := range env {
			copyEnv[key] = value
		}
		f.envs = append(f.envs, copyEnv)
	} else {
		f.envs = append(f.envs, nil)
	}
	return f.Run(ctx, repo, args...)
}

func testConfig() config.Config {
	cfg := config.Default()
	cfg.Repo = "."
	cfg.Remotes = map[string]config.RemoteConfig{
		"github": {URL: "git@github.com:example/example-backup.git", Enabled: true, Required: true, Provider: "github"},
		"gitlab": {URL: "git@gitlab.com:example/example-backup.git", Enabled: false, Required: false, Provider: "gitlab"},
	}
	return cfg
}

func TestBuildPlanSafeModeNoDangerousOps(t *testing.T) {
	svc := New(gitpkg.NewClient(&fakeRunner{}))
	cfg := testConfig()
	plan, err := svc.BuildPlan(context.Background(), cfg, RunOptions{RepoPath: ".", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.DangerousOps) != 0 {
		t.Fatalf("unexpected dangerous ops: %v", plan.DangerousOps)
	}
	if plan.Mode != "safe" || !plan.DryRun {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestBuildPlanMirrorIncludesDangerousOperation(t *testing.T) {
	svc := New(gitpkg.NewClient(&fakeRunner{}))
	cfg := testConfig()
	cfg.Mode = "mirror"
	plan, err := svc.BuildPlan(context.Background(), cfg, RunOptions{RepoPath: "."})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.DangerousOps) == 0 {
		t.Fatal("expected dangerous operation")
	}
}

func TestBuildPlanBundleEnabled(t *testing.T) {
	svc := New(gitpkg.NewClient(&fakeRunner{}))
	cfg := testConfig()
	cfg.Bundle.Enabled = true
	plan, err := svc.BuildPlan(context.Background(), cfg, RunOptions{RepoPath: "."})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.BundleEnabled {
		t.Fatal("expected bundle enabled in plan")
	}
}

func TestBuildPlanPushAllRefs(t *testing.T) {
	svc := New(gitpkg.NewClient(&fakeRunner{}))
	cfg := testConfig()
	cfg.Options.PushAllRefs = true
	plan, err := svc.BuildPlan(context.Background(), cfg, RunOptions{RepoPath: "."})
	if err != nil {
		t.Fatal(err)
	}
	if plan.BranchPushMode != "all-refs" || plan.TagPushMode != "all-refs" || !plan.PushNotes {
		t.Fatalf("unexpected plan for push_all_refs: %+v", plan)
	}
}

func TestBuildPlanPruneAddsModeSuffix(t *testing.T) {
	svc := New(gitpkg.NewClient(&fakeRunner{}))
	cfg := testConfig()
	cfg.Options.Prune = true
	plan, err := svc.BuildPlan(context.Background(), cfg, RunOptions{RepoPath: "."})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.BranchPushMode, "+prune") || !strings.Contains(plan.TagPushMode, "+prune") {
		t.Fatalf("expected prune modes in plan: %+v", plan)
	}
}

func TestDryRunDoesNotMutate(t *testing.T) {
	runner := &fakeRunner{}
	svc := New(gitpkg.NewClient(runner))
	cfg := testConfig()
	report, _, err := svc.Run(context.Background(), cfg, RunOptions{RepoPath: ".", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Success {
		t.Fatal("expected dry run success")
	}
	for _, call := range runner.calls {
		if strings.HasPrefix(call, "push ") || strings.HasPrefix(call, "remote add") || strings.HasPrefix(call, "remote set-url") || strings.HasPrefix(call, "bundle create") {
			t.Fatalf("dry run mutated state: %s", call)
		}
	}
}

func TestRunPruneUsesPrunePush(t *testing.T) {
	runner := &fakeRunner{}
	svc := New(gitpkg.NewClient(runner))
	cfg := testConfig()
	cfg.Options.Prune = true
	report, _, err := svc.Run(context.Background(), cfg, RunOptions{RepoPath: "."})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Success {
		t.Fatal("expected success")
	}
	foundPrune := false
	for _, call := range runner.calls {
		if strings.Contains(call, "--prune") {
			foundPrune = true
			break
		}
	}
	if !foundPrune {
		t.Fatal("expected prune push commands")
	}
}

func TestDisabledRemotesIgnoredUnlessSelected(t *testing.T) {
	svc := New(gitpkg.NewClient(&fakeRunner{}))
	cfg := testConfig()
	plan, err := svc.BuildPlan(context.Background(), cfg, RunOptions{RepoPath: "."})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Targets) != 1 || plan.Targets[0].Name != "github" {
		t.Fatalf("disabled remotes should be ignored: %+v", plan.Targets)
	}
	_, err = svc.BuildPlan(context.Background(), cfg, RunOptions{RepoPath: ".", SelectedRemotes: []string{"gitlab"}})
	if err == nil {
		t.Fatal("expected error when selecting disabled remote without include flag")
	}
	plan, err = svc.BuildPlan(context.Background(), cfg, RunOptions{RepoPath: ".", SelectedRemotes: []string{"gitlab"}, IncludeDisabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Targets) != 1 || plan.Targets[0].Name != "gitlab" {
		t.Fatalf("expected selected disabled remote: %+v", plan.Targets)
	}
}

func TestRunMirrorRequiresYesWhenNonInteractive(t *testing.T) {
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
		_ = w.Close()
	}()
	svc := New(gitpkg.NewClient(&fakeRunner{}))
	cfg := testConfig()
	cfg.Mode = "mirror"
	_, _, err = svc.Run(context.Background(), cfg, RunOptions{RepoPath: "."})
	if err == nil || !strings.Contains(err.Error(), "--yes in non-interactive environments") {
		t.Fatalf("expected non-interactive mirror error, got %v", err)
	}
}

func TestRunSkipLFSUsesSkipSmudgeEnv(t *testing.T) {
	runner := &fakeRunner{}
	svc := New(gitpkg.NewClient(runner))
	cfg := testConfig()
	cfg.Options.FetchBeforeBackup = true
	cfg.Options.PullBeforeBackup = true
	cfg.Options.SkipLFS = true
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".gitattributes"), []byte("*.psd filter=lfs diff=lfs merge=lfs -text\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := svc.Run(context.Background(), cfg, RunOptions{RepoPath: repo})
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.envs) == 0 {
		t.Fatal("expected env-aware git calls")
	}
	found := false
	for _, env := range runner.envs {
		if env["GIT_LFS_SKIP_SMUDGE"] == "1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected GIT_LFS_SKIP_SMUDGE in envs: %#v", runner.envs)
	}
}
