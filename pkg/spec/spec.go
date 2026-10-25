// Package spec builds the template context for a GitHub repository and renders spec files from it.
package spec

import (
	"fmt"
	"maps"
	"regexp"
	"runtime"
	"strings"

	"github.com/pgvillage-tools/specbuilder/pkg/github"
	"github.com/pgvillage-tools/specbuilder/pkg/render"
)

const (
	name = "name"
)

// packageArch maps Go architectures to rpm architectures (as reported by `uname -m`).
var packageArch = map[string]string{
	"amd64": "x86_64",
	"arm64": "aarch64",
	"386":   "i686",
}

// PackageArch returns the rpm architecture for a Go architecture.
func PackageArch(goarch string) string {
	if arch, ok := packageArch[goarch]; ok {
		return arch
	}
	return goarch
}

// Fetcher reads repository and release info.
type Fetcher interface {
	GetRepo(site, organization, repository string) (github.Repo, error)
	GetRelease(site, organization, repository, release string) (github.Release, error)
}

// Repo is the template context for a repository.
// Values from the config take precedence over defaults and info read from the GitHub API.
type Repo map[string]any

// NewRepo builds the template context for a repository, reading repository and release info with fetcher.
func NewRepo(entry string, values map[string]any, fetcher Fetcher) (Repo, error) {
	values = maps.Clone(values)
	values[name] = entry
	setDefault(values, "site", "github.com")
	setDefault(values, "repository", values[name])
	setDefault(values, "organization", values["repository"])

	// These defaults can be overridden from the yaml config if needed
	r := Repo{
		"url": fmt.Sprintf("https://%v/%v/%v",
			values["site"], values["organization"], values["repository"]),
		"package_arch":   PackageArch(runtime.GOARCH),
		"binary_arch":    runtime.GOARCH,
		"target_release": "latest",
		"package_name":   entry,
		"asset_filter":   "{{ name }}-{{ version }}-linux-{{ binary_arch }}.tar.gz$",
	}

	// Copy values, get data from the GitHub API and overwrite with values again
	maps.Copy(r, values)
	if err := r.readRepoInfo(fetcher); err != nil {
		return nil, err
	}
	if err := r.readReleaseInfo(fetcher); err != nil {
		return nil, err
	}
	maps.Copy(r, values)
	return r, nil
}

func setDefault(values map[string]any, key string, value any) {
	if _, ok := values[key]; !ok {
		values[key] = value
	}
}

func (r Repo) str(key string) string {
	if v, ok := r[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func (r Repo) readRepoInfo(fetcher Fetcher) error {
	info, err := fetcher.GetRepo(r.str("site"), r.str("organization"), r.str("repository"))
	if err != nil {
		return fmt.Errorf("reading repo info for %s: %w", r.str(name), err)
	}
	r["repo_name"] = info.Name
	r["description"] = info.Description
	r["license"] = info.License
	return nil
}

func (r Repo) readReleaseInfo(fetcher Fetcher) error {
	release, err := fetcher.GetRelease(r.str("site"), r.str("organization"), r.str("repository"),
		r.str("target_release"))
	if err != nil {
		return fmt.Errorf("reading release info for %s: %w", r.str(name), err)
	}
	releaseInfo := map[string]any{
		"version":   strings.ReplaceAll(release.TagName, "-", "."),
		"changelog": release.Body,
	}
	maps.Copy(r, releaseInfo)

	filter := "."
	if f := r.str("asset_filter"); f != "" {
		filter = f
	}
	filter, err = r.Render(filter)
	if err != nil {
		return fmt.Errorf("rendering asset_filter for %s: %w", r.str(name), err)
	}
	re, err := regexp.Compile(filter)
	if err != nil {
		return fmt.Errorf("compiling asset_filter for %s: %w", r.str(name), err)
	}
	assets := []map[string]any{}
	for _, asset := range release.Assets {
		if re.MatchString(asset.Name) {
			assets = append(assets, map[string]any{name: asset.Name, "url": asset.URL})
		}
	}
	releaseInfo["assets"] = assets
	r["assets"] = assets
	r["release_info"] = releaseInfo
	return nil
}

// Render renders a template recursively, using the repo as context.
func (r Repo) Render(tpl string) (string, error) {
	return render.Recursive(tpl, r)
}
