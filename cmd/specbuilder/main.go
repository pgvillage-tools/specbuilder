// Command specbuilder generates rpm spec files from GitHub repos with singleton binaries.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pgvillage-tools/specbuilder/pkg/config"
	"github.com/pgvillage-tools/specbuilder/pkg/github"
	"github.com/pgvillage-tools/specbuilder/pkg/spec"
)

const readWriteUser = 0o644

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
	only := flag.String("only", os.Getenv("SPECBUILDER_ONLY"),
		"only generate these repositories, separated by commas or spaces (env: SPECBUILDER_ONLY)")
	changed := flag.String("changed", "",
		"print the repositories that are new or changed compared to this config file, instead of generating")
	flag.Parse()

	var err error
	if *changed != "" {
		err = printChanged(*changed, *configFile)
	} else {
		err = run(*configFile, *dest, splitNames(*only))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// splitNames splits a list of names, separated by commas and/or whitespace.
func splitNames(names string) []string {
	return strings.FieldsFunc(names, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
}

// printChanged prints the repositories in configFile that are new or changed compared to baseFile, one per line.
func printChanged(baseFile, configFile string) error {
	base, err := config.Load(baseFile)
	if err != nil {
		return err
	}
	cfg, err := config.Load(configFile)
	if err != nil {
		return err
	}
	for _, name := range config.Changed(base, cfg) {
		fmt.Println(name)
	}
	return nil
}

func run(configFile, dest string, only []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return err
	}
	if len(only) > 0 {
		if cfg.Repositories, err = cfg.Repositories.Select(only); err != nil {
			return err
		}
	}
	if dest == "" {
		dest = cfg.Dest
	}
	if dest == "" {
		dest = envOrDefault("GITHUB2SPEC_DEST", "/tmp")
	}
	client := github.NewClient(os.Getenv("GITHUB_TOKEN"))
	templates := map[string]string{}
	var failed []string
	for _, repoCfg := range cfg.Repositories {
		tpl, err := readTemplate(templates, cfg.TemplateOf(repoCfg))
		if err == nil {
			err = writeSpec(client, repoCfg, tpl, dest)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			failed = append(failed, repoCfg.Name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to build specs for %v", failed)
	}
	return nil
}

// readTemplate reads a template file, caching it in templates, as most repositories share the same template.
func readTemplate(templates map[string]string, path string) (string, error) {
	if tpl, ok := templates[path]; ok {
		return tpl, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading template: %w", err)
	}
	templates[path] = string(data)
	return templates[path], nil
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
	if err := os.WriteFile(destFile, []byte(content), readWriteUser); err != nil {
		return fmt.Errorf("writing spec for %s: %w", repoCfg.Name, err)
	}
	return nil
}
