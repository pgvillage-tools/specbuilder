// Command specbuilder generates rpm spec files from GitHub repos with singleton binaries.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pgvillage-tools/specbuilder/pkg/config"
	"github.com/pgvillage-tools/specbuilder/pkg/github"
	"github.com/pgvillage-tools/specbuilder/pkg/spec"
)

func envOrDefault(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func main() {
	configFile := flag.String("config",
		envOrDefault("GITHUB2SPEC_CONFIG", "/usr/rpmbuilder/conf/github2spec.yaml"),
		"path to the config file (env: GITHUB2SPEC_CONFIG)")
	dest := flag.String("dest", "",
		"destination folder for spec files (default: 'dest' from config, else $GITHUB2SPEC_DEST, else /tmp)")
	flag.Parse()

	if err := run(*configFile, *dest); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(configFile, dest string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return err
	}
	if dest == "" {
		dest = cfg.Dest
	}
	if dest == "" {
		dest = envOrDefault("GITHUB2SPEC_DEST", "/tmp")
	}
	tpl, err := os.ReadFile(cfg.Template)
	if err != nil {
		return fmt.Errorf("reading template: %w", err)
	}

	client := github.NewClient(os.Getenv("GITHUB_TOKEN"))
	var failed []string
	for _, repoCfg := range cfg.Repositories {
		if err := writeSpec(client, repoCfg, string(tpl), dest); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			failed = append(failed, repoCfg.Name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to build specs for %v", failed)
	}
	return nil
}

func writeSpec(client *github.Client, repoCfg config.Repository, tpl, dest string) error {
	repo, err := spec.NewRepo(repoCfg.Name, repoCfg.Values, client)
	if err != nil {
		return err
	}
	content, err := repo.Render(tpl)
	if err != nil {
		return fmt.Errorf("rendering spec for %s: %w", repoCfg.Name, err)
	}
	destFile := filepath.Join(dest, fmt.Sprintf("%s-%v.spec", repoCfg.Name, repo["package_arch"]))
	fmt.Printf("Writing to %s\n", destFile)
	if err := os.WriteFile(destFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing spec for %s: %w", repoCfg.Name, err)
	}
	return nil
}
