package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load empty path: %v", err)
	}
	if cfg.Server.Port == 0 {
		t.Error("expected non-zero server port from defaults")
	}
	if len(cfg.Profiles) == 0 {
		t.Error("expected profiles in default config")
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/to/config.yaml")
	if err != nil {
		t.Fatalf("Load missing file: expected fallback to defaults, got error: %v", err)
	}
	if cfg.Server.Port == 0 {
		t.Error("expected non-zero server port from fallback defaults")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
server:
  host: "0.0.0.0"
  port: 9090
database:
  path: "./test.db"
session:
  idle_timeout_minutes: 10
tasks:
  randomize: true
replication:
  enabled: false
  count: 1
profiles:
  - match: { model: "test-model", effort: "low" }
    difficulty: [1, 2]
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("got host %q, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("got port %d, want 9090", cfg.Server.Port)
	}
	if cfg.Session.IdleTimeoutMinutes != 10 {
		t.Errorf("got idle_timeout_minutes %d, want 10", cfg.Session.IdleTimeoutMinutes)
	}
}

func TestValidateOK(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate on defaults: %v", err)
	}
}

func TestValidateBadPort(t *testing.T) {
	cfg, _ := Load("")
	cfg.Server.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for port=0")
	}
	cfg.Server.Port = 70000
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for port=70000")
	}
}

func TestValidateBadIdleTimeout(t *testing.T) {
	cfg, _ := Load("")
	cfg.Session.IdleTimeoutMinutes = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for idle_timeout_minutes=0")
	}
}

func TestValidateBadDifficulty(t *testing.T) {
	cfg, _ := Load("")
	cfg.Profiles = []Profile{
		{Match: ProfileMatch{Model: "m", Effort: "low"}, Difficulty: [2]int{0, 2}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for difficulty[0]=0")
	}

	cfg.Profiles = []Profile{
		{Match: ProfileMatch{Model: "m", Effort: "low"}, Difficulty: [2]int{1, 4}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for difficulty[1]=4")
	}

	cfg.Profiles = []Profile{
		{Match: ProfileMatch{Model: "m", Effort: "low"}, Difficulty: [2]int{3, 2}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for difficulty[0] > difficulty[1]")
	}
}

func TestValidateMissingProfileFields(t *testing.T) {
	cfg, _ := Load("")

	cfg.Profiles = []Profile{
		{Match: ProfileMatch{Model: "", Effort: "low"}, Difficulty: [2]int{1, 2}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty model")
	}

	cfg.Profiles = []Profile{
		{Match: ProfileMatch{Model: "m", Effort: ""}, Difficulty: [2]int{1, 2}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty effort")
	}

	cfg.Profiles = []Profile{
		{Match: ProfileMatch{Model: "m", Effort: "superfast"}, Difficulty: [2]int{1, 2}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid effort value")
	}
}
