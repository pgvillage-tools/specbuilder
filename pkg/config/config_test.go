package config

import (
	"os"
	"path/filepath"
	"slices"
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
	if err := os.WriteFile(path, []byte("repositories:\n  a:\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("expected an error")
	}
}

func TestLoadPerEntryTemplate(t *testing.T) {
	cfg := loadString(t, "repositories:\n  own:\n    template: own.j2\n")
	if got := cfg.TemplateOf(cfg.Repositories[0]); got != "own.j2" {
		t.Errorf("unexpected template: %q", got)
	}
	cfg = loadString(t, "template: spec.j2\nrepositories:\n  own:\n    template: own.j2\n  default:\n")
	if got := cfg.TemplateOf(cfg.Repositories[0]); got != "own.j2" {
		t.Errorf("unexpected template for own: %q", got)
	}
	if got := cfg.TemplateOf(cfg.Repositories[1]); got != "spec.j2" {
		t.Errorf("unexpected template for default: %q", got)
	}
}

func TestLoadInvalidTemplate(t *testing.T) {
	for name, data := range map[string]string{
		"no default":   "repositories:\n  own:\n    template: own.j2\n  default:\n",
		"not a string": "template: spec.j2\nrepositories:\n  own:\n    template: [own.j2]\n",
		"empty":        "template: spec.j2\nrepositories:\n  own:\n    template: ''\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func loadString(t *testing.T, data string) Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestSelect(t *testing.T) {
	cfg := loadString(t, "template: spec.j2\nrepositories:\n  zeta:\n  alpha:\n  mid:\n")
	selected, err := cfg.Repositories.Select([]string{"mid", "zeta"})
	if err != nil {
		t.Fatal(err)
	}
	if names := selected.Names(); !slices.Equal(names, []string{"zeta", "mid"}) {
		t.Errorf("unexpected selection: %v", names)
	}
	if _, err := cfg.Repositories.Select([]string{"zeta", "unknown"}); err == nil {
		t.Error("expected an error for an unknown repository")
	}
}

func TestChanged(t *testing.T) {
	const base = `template: spec.j2
dest: /out
repositories:
  same:
    organization: org
  bumped:
    target_release: v1.0.0
    files:
      - src: a
  own:
    template: own.j2
  removed:
`
	for _, tc := range []struct {
		name, cfg string
		want      []string
	}{
		{"identical", base, nil},
		{"comments and order of keys", `# comment
dest: /out
template: spec.j2
repositories:
  same:
    # renovate: datasource=github-releases depName=org/same
    organization: org
  bumped:
    files:
      - src: a
    target_release: v1.0.0
`, nil},
		{"bumped and added", `template: spec.j2
dest: /out
repositories:
  same:
    organization: org
  bumped:
    target_release: v1.0.1
    files:
      - src: a
  added:
`, []string{"bumped", "added"}},
		{"nested value", `template: spec.j2
dest: /out
repositories:
  same:
    organization: org
  bumped:
    target_release: v1.0.0
    files:
      - src: b
`, []string{"bumped"}},
		{"default template", `template: other.j2
dest: /out
repositories:
  same:
    organization: org
  bumped:
    target_release: v1.0.0
  own:
    template: own.j2
`, []string{"same", "bumped"}},
		{"own template", `template: spec.j2
dest: /out
repositories:
  same:
    organization: org
  own:
    template: other.j2
`, []string{"own"}},
		{"dest", `template: spec.j2
dest: /other
repositories:
  same:
    organization: org
  own:
    template: own.j2
`, []string{"same", "own"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Changed(loadString(t, base), loadString(t, tc.cfg)); !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
