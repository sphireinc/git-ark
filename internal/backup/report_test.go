package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git-ark/internal/config"
)

func TestWriteMetadataAppendsHistory(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Metadata.Path = ".git/git-ark-last-backup.json"
	svc := &Service{}
	first := Report{
		Repo:       repo,
		Mode:       "safe",
		StartedAt:  time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 4, 27, 12, 1, 0, 0, time.UTC),
		DurationMS: 60000,
		Results:    []TargetResult{{Target: "github", Success: true}},
		Success:    true,
	}
	second := Report{
		Repo:       repo,
		Mode:       "mirror",
		StartedAt:  time.Date(2026, 4, 27, 13, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 4, 27, 13, 2, 0, 0, time.UTC),
		DurationMS: 120000,
		Results:    []TargetResult{{Target: "github", Success: true}},
		Success:    true,
	}
	if err := svc.WriteMetadata(cfg, first, repo); err != nil {
		t.Fatal(err)
	}
	if err := svc.WriteMetadata(cfg, second, repo); err != nil {
		t.Fatal(err)
	}
	meta, err := ReadMetadata(filepath.Join(repo, cfg.Metadata.Path))
	if err != nil {
		t.Fatal(err)
	}
	if meta.SchemaVersion != MetadataSchemaVersion {
		t.Fatalf("schema version = %d", meta.SchemaVersion)
	}
	if meta.Repo != repo {
		t.Fatalf("repo = %q", meta.Repo)
	}
	if len(meta.History) != 2 {
		t.Fatalf("history length = %d", len(meta.History))
	}
	latest, ok := meta.LatestReport()
	if !ok {
		t.Fatal("expected latest report")
	}
	if latest.Mode != "mirror" {
		t.Fatalf("latest mode = %q", latest.Mode)
	}
	if meta.History[1].Mode != "safe" {
		t.Fatalf("older history entry missing: %+v", meta.History)
	}
}

func TestReadLegacyMetadataWrapsReport(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo, ".git", "git-ark-last-backup.json")
	legacy := Report{
		Repo:       repo,
		Mode:       "safe",
		StartedAt:  time.Date(2026, 4, 27, 14, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 4, 27, 14, 1, 0, 0, time.UTC),
		DurationMS: 60000,
		Success:    true,
	}
	raw, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	meta, err := ReadMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.History) != 1 {
		t.Fatalf("history length = %d", len(meta.History))
	}
	if meta.History[0].Mode != "safe" || !meta.History[0].Success {
		t.Fatalf("legacy report not preserved: %+v", meta.History[0])
	}
	if meta.Repo != repo {
		t.Fatalf("repo = %q", meta.Repo)
	}
}
