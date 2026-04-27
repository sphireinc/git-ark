package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMinimalConfigAppliesDefaults(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "configs", "minimal.yml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, warnings, err := ApplyDefaults(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if cfg.Version != CurrentVersion {
		t.Fatalf("version = %d", cfg.Version)
	}
	if cfg.Repo != "." || cfg.Mode != "safe" {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
	if !cfg.Options.ManageRemotes || !cfg.Options.PushBranches || !cfg.Options.PushTags || !cfg.Options.ContinueOnError || !cfg.Options.IncludeArchivedBranches || !cfg.Options.WriteMetadata {
		t.Fatalf("option defaults not applied: %+v", cfg.Options)
	}
	remote := cfg.Remotes["github"]
	if !remote.Enabled || !remote.Required {
		t.Fatalf("remote defaults not applied: %+v", remote)
	}
	if cfg.Bundle.Path != "./backups" || cfg.Metadata.Path != ".git/git-ark-last-backup.json" {
		t.Fatalf("bundle or metadata defaults not applied: %+v %+v", cfg.Bundle, cfg.Metadata)
	}
}

func TestFullConfigParses(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "configs", "full.yml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, warnings, err := ApplyDefaults(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if cfg.Bundle.Enabled != true || cfg.Logging.Level != "debug" || cfg.BranchFilters.Include[1] != "release/*" || cfg.TagFilters.Include[0] != "v*" {
		t.Fatalf("full config values not parsed: %+v", cfg)
	}
}

func TestInvalidModeFails(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "configs", "invalid.yml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, _, err := ApplyDefaults(data)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Validate(cfg)
	if err == nil {
		t.Fatal("expected invalid mode error")
	}
}

func TestMissingRemotesFails(t *testing.T) {
	cfg := Default()
	cfg.Remotes = map[string]RemoteConfig{}
	_, err := Validate(cfg)
	if err == nil {
		t.Fatal("expected missing remotes error")
	}
}

func TestUnknownTopLevelWarns(t *testing.T) {
	data := []byte("version: 1\nextra_field: true\nremotes:\n  github:\n    url: git@github.com:example/example-backup.git\n")
	_, warnings, err := ApplyDefaults(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning for unknown field")
	}
}
