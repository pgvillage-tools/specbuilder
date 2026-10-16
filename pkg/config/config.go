// Package config reads the specbuilder yaml config file.
package config

import (
	"fmt"
	"os"

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

// Config is the specbuilder configuration.
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
	if cfg.Template == "" {
		return cfg, fmt.Errorf("config %s: missing 'template'", path)
	}
	return cfg, nil
}
