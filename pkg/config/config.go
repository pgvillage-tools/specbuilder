// Package config reads the specbuilder yaml config file.
package config

import (
	"fmt"
	"os"
	"reflect"
	"slices"

	"gopkg.in/yaml.v3"
)

// Repository is a repository entry from the config, with its name and (template) values.
type Repository struct {
	Name   string
	Values map[string]any
}

// Repositories is an ordered list of repositories, unmarshalled from a yaml mapping.
type Repositories []Repository

// UnmarshalYAML decodes a yaml mapping while preserving the order of the repositories.
func (r *Repositories) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: repositories should be a mapping", node.Line)
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		repo := Repository{Name: node.Content[i].Value}
		if err := node.Content[i+1].Decode(&repo.Values); err != nil {
			return fmt.Errorf("repository %s: %w", repo.Name, err)
		}
		if repo.Values == nil {
			repo.Values = map[string]any{}
		}
		*r = append(*r, repo)
	}
	return nil
}

// Names returns the names of the repositories, in config order.
func (r Repositories) Names() []string {
	names := make([]string, 0, len(r))
	for _, repo := range r {
		names = append(names, repo.Name)
	}
	return names
}

// Select returns the repositories with the given names, in config order.
// It fails when a name is not in the config.
func (r Repositories) Select(names []string) (Repositories, error) {
	known := r.Names()
	for _, name := range names {
		if !slices.Contains(known, name) {
			return nil, fmt.Errorf("repository %s is not in the config", name)
		}
	}
	var selected Repositories
	for _, repo := range r {
		if slices.Contains(names, repo.Name) {
			selected = append(selected, repo)
		}
	}
	return selected, nil
}

// Config is the specbuilder configuration.
// Template is the default template; a repository can use another one with its own 'template' value.
type Config struct {
	Template     string       `yaml:"template"`
	Dest         string       `yaml:"dest"`
	Repositories Repositories `yaml:"repositories"`
}

// Load reads a config file.
func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("reading config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", path, err)
	}
	for _, repo := range cfg.Repositories {
		if tpl, ok := repo.Values["template"]; ok {
			if s, isString := tpl.(string); !isString || s == "" {
				return cfg, fmt.Errorf("config %s: 'template' of repository %s should be a path", path, repo.Name)
			}
		} else if cfg.Template == "" {
			return cfg, fmt.Errorf("config %s: missing 'template' (for repository %s)", path, repo.Name)
		}
	}
	return cfg, nil
}

// TemplateOf returns the template for repo: its own 'template' value, or else the default template of the config.
func (c Config) TemplateOf(repo Repository) string {
	if tpl, ok := repo.Values["template"].(string); ok {
		return tpl
	}
	return c.Template
}

// Changed returns the names of the repositories in cfg that are new, have other values or use another template than
// in base, in config order. When the destination differs, all repositories in cfg have changed.
// Repositories that are only in base are left out, as there is nothing to build for them.
func Changed(base, cfg Config) []string {
	if base.Dest != cfg.Dest {
		return cfg.Repositories.Names()
	}
	baseRepos := map[string]Repository{}
	for _, repo := range base.Repositories {
		baseRepos[repo.Name] = repo
	}
	var changed []string
	for _, repo := range cfg.Repositories {
		baseRepo, ok := baseRepos[repo.Name]
		if !ok || !reflect.DeepEqual(baseRepo.Values, repo.Values) ||
			base.TemplateOf(baseRepo) != cfg.TemplateOf(repo) {
			changed = append(changed, repo.Name)
		}
	}
	return changed
}
