package spec

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/pgvillage-tools/specbuilder/pkg/github"
)

type fakeFetcher struct {
	repo    github.Repo
	release github.Release
	calls   []string
}

func (f *fakeFetcher) GetRepo(site, organization, repository string) (github.Repo, error) {
	f.calls = append(f.calls, "repo "+site+"/"+organization+"/"+repository)
	return f.repo, nil
}

func (f *fakeFetcher) GetRelease(site, organization, repository, release string) (github.Release, error) {
	f.calls = append(f.calls, "release "+site+"/"+organization+"/"+repository+"/"+release)
	return f.release, nil
}

func newEtcdFetcher() *fakeFetcher {
	arch := runtime.GOARCH
	return &fakeFetcher{
		repo: github.Repo{Name: "etcd", Description: "Distributed reliable key-value store", License: "Apache-2.0"},
		release: github.Release{
			TagName: "v3.5.0",
			Body:    "changes",
			Assets: []github.Asset{
				{Name: "etcd-v3.5.0-linux-" + arch + ".tar.gz", URL: "https://example.com/etcd-linux.tar.gz"},
				{Name: "etcd-v3.5.0-darwin-" + arch + ".zip", URL: "https://example.com/etcd-darwin.zip"},
			},
		},
	}
}

func etcdValues() map[string]any {
	return map[string]any{
		"organization": "etcd-io",
		"asset_filter": "{{ package_name }}-{{ version }}-linux-{{ binary_arch }}.tar.gz$",
		"files": []any{
			map[string]any{
				"src":  "etcd-{{ version }}-linux-{{ binary_arch }}/etcdctl",
				"dest": "%{_bindir}/etcdctl",
				"mode": "0755",
			},
			map[string]any{
				"src":  "etcd-{{ version }}-linux-{{ binary_arch }}/etcd",
				"dest": "%{_bindir}/etcd",
			},
		},
	}
}

func TestNewRepo(t *testing.T) {
	fetcher := newEtcdFetcher()
	repo, err := NewRepo("etcd", etcdValues(), fetcher)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"name":         "etcd",
		"repository":   "etcd",
		"url":          "https://github.com/etcd-io/etcd",
		"version":      "v3.5.0",
		"license":      "Apache-2.0",
		"package_name": "etcd",
		"binary_arch":  runtime.GOARCH,
		"package_arch": PackageArch(runtime.GOARCH),
	}
	for key, value := range want {
		if repo[key] != value {
			t.Errorf("repo[%q] = %v, want %v", key, repo[key], value)
		}
	}
	assets := repo["assets"].([]map[string]any)
	if len(assets) != 1 || assets[0]["url"] != "https://example.com/etcd-linux.tar.gz" {
		t.Errorf("unexpected assets: %v", assets)
	}
	if fetcher.calls[1] != "release github.com/etcd-io/etcd/latest" {
		t.Errorf("unexpected release call: %s", fetcher.calls[1])
	}
	if _, ok := repo["etcd"]; ok {
		t.Error("the entry name should be stored under 'name', not under its own name")
	}
}

func TestDefaultAssetFilter(t *testing.T) {
	values := etcdValues()
	delete(values, "asset_filter")
	fetcher := newEtcdFetcher()
	// only matches when {{ name }} renders empty
	fetcher.release.Assets = append(fetcher.release.Assets,
		github.Asset{Name: "etcd-tools-v3.5.0-linux-" + runtime.GOARCH + ".tar.gz", URL: "https://example.com/tools.tar.gz"})
	repo, err := NewRepo("etcd", values, fetcher)
	if err != nil {
		t.Fatal(err)
	}
	assets := repo["assets"].([]map[string]any)
	if len(assets) != 1 || assets[0]["name"] != "etcd-v3.5.0-linux-"+runtime.GOARCH+".tar.gz" {
		t.Errorf("unexpected assets: %v", assets)
	}
}

func TestConfigOverridesAPI(t *testing.T) {
	values := etcdValues()
	values["license"] = "MIT"
	values["target_release"] = "tags/v3.5.0"
	fetcher := newEtcdFetcher()
	repo, err := NewRepo("etcd", values, fetcher)
	if err != nil {
		t.Fatal(err)
	}
	if repo["license"] != "MIT" {
		t.Errorf("license = %v, want MIT", repo["license"])
	}
	if fetcher.calls[1] != "release github.com/etcd-io/etcd/tags/v3.5.0" {
		t.Errorf("unexpected release call: %s", fetcher.calls[1])
	}
}

func TestRenderSpecTemplate(t *testing.T) {
	tpl, err := os.ReadFile("../../rpmbuilder/templates/spec.j2")
	if err != nil {
		t.Skip("template not available:", err)
	}
	repo, err := NewRepo("etcd", etcdValues(), newEtcdFetcher())
	if err != nil {
		t.Fatal(err)
	}
	out, err := repo.Render(string(tpl))
	if err != nil {
		t.Fatal(err)
	}
	arch := runtime.GOARCH
	for _, line := range []string{
		"Summary: Distributed reliable key-value store",
		"Version: v3.5.0",
		"License: Apache-2.0",
		"Group: Unspecified",
		"Url: https://github.com/etcd-io/etcd",
		"Source0: etcd-v3.5.0-linux-" + arch + ".tar.gz",
		"BuildArch: " + PackageArch(arch),
		"curl -L https://example.com/etcd-linux.tar.gz -o %{_sourcedir}/etcd-v3.5.0-linux-" + arch + ".tar.gz",
		"%{__install} -m 0755 %{_builddir}/etcd-v3.5.0-linux-" + arch + "/etcdctl %{buildroot}/%{_bindir}/etcdctl",
		"%{__install} -m 0755 %{_builddir}/etcd-v3.5.0-linux-" + arch + "/etcd %{buildroot}/%{_bindir}/etcd",
	} {
		if !strings.Contains(out, line+"\n") {
			t.Errorf("rendered spec misses line %q\n%s", line, out)
		}
	}
	if strings.Contains(out, "Source1") || strings.Contains(out, "Requires") {
		t.Errorf("unexpected content in rendered spec:\n%s", out)
	}
}
