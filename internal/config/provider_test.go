package config

import (
	"strings"
	"testing"
)

func TestValidateWarnsOnProviderMismatch(t *testing.T) {
	cfg := Default()
	cfg.Remotes = map[string]RemoteConfig{
		"github": {
			URL:      "git@gitlab.com:example/example-backup.git",
			Enabled:  true,
			Required: true,
			Provider: "github",
		},
	}

	warnings, err := Validate(cfg)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, warning := range warnings {
		if strings.Contains(warning, "GitHub") && strings.Contains(warning, "gitlab.com") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected provider mismatch warning, got %v", warnings)
	}
}

func TestDiagnoseProviderReportsHTTPSNote(t *testing.T) {
	diag := DiagnoseProvider(RemoteConfig{
		URL:      "https://github.com/example/example-backup.git",
		Provider: "github",
	})

	if diag.DisplayName != "GitHub" {
		t.Fatalf("display name = %q", diag.DisplayName)
	}
	if len(diag.Notes) == 0 {
		t.Fatal("expected HTTPS guidance note")
	}
	if !strings.Contains(diag.Notes[0], "token") {
		t.Fatalf("unexpected note: %v", diag.Notes)
	}
}
