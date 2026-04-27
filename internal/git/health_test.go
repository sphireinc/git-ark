package git

import (
	"context"
	"errors"
	"testing"
)

type healthRunner struct {
	result Result
	err    error
}

func (h *healthRunner) Run(ctx context.Context, repo string, args ...string) (Result, error) {
	_ = ctx
	_ = repo
	_ = args
	return h.result, h.err
}

func TestProbeRemoteCountsRefs(t *testing.T) {
	client := NewClient(&healthRunner{
		result: Result{Stdout: "ref1\trefs/heads/main\nref2\trefs/tags/v1.0.0\n"},
	})
	health := client.ProbeRemote(context.Background(), "git@github.com:example/example-backup.git")
	if !health.Reachable {
		t.Fatal("expected reachable remote")
	}
	if health.RefCount != 2 {
		t.Fatalf("ref count = %d", health.RefCount)
	}
}

func TestProbeRemoteCapturesErrorHint(t *testing.T) {
	client := NewClient(&healthRunner{
		err: errors.New("fatal: could not read from remote repository"),
	})
	health := client.ProbeRemote(context.Background(), "git@github.com:example/example-backup.git")
	if health.Reachable {
		t.Fatal("expected unreachable remote")
	}
	if health.Hint == "" {
		t.Fatal("expected error hint")
	}
}
