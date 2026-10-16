package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadKeepsRepositoryOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := "template: spec.j2\nrepositories:\n  zeta:\n    organization: org\n  alpha:\n  mid:\n    files:\n      - src: a\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, repo := range cfg.Repositories {
		names = append(names, repo.Name)
	}
	if len(names) != 3 || names[0] != "zeta" || names[1] != "alpha" || names[2] != "mid" {
		t.Errorf("unexpected order: %v", names)
	}
	if cfg.Repositories[0].Values["organization"] != "org" || cfg.Repositories[1].Values == nil {
		t.Errorf("unexpected values: %+v", cfg.Repositories)
	}
}

func TestLoadRequiresTemplate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("repositories: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("expected an error")
	}
}
