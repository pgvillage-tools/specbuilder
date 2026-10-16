# specbuilder

Generate RPM spec files for GitHub projects that ship prebuilt (singleton) binaries.

For every repository in a YAML config file, specbuilder:

1. reads the repository info (name, description, license) from the GitHub API,
2. reads a release (`latest` by default) and selects the release assets that match a filter,
3. renders a [Jinja2](https://jinja.palletsprojects.com/) template into `<name>-<arch>.spec`.

specbuilder is a Go rewrite of the Python `github2spec` tool, and works with the same config files and templates.

## Installation

```
go install github.com/pgvillage-tools/specbuilder/cmd/specbuilder@latest
```

Or build from source:

```
git clone https://github.com/pgvillage-tools/specbuilder
cd specbuilder
go build ./cmd/specbuilder
```

## Usage

```
specbuilder [-config <file>] [-dest <folder>]
```

| Flag      | Environment variable | Default                                  | Description                       |
|-----------|----------------------|------------------------------------------|-----------------------------------|
| `-config` | `GITHUB2SPEC_CONFIG` | `/usr/rpmbuilder/conf/github2spec.yaml`  | Path to the config file           |
| `-dest`   | `GITHUB2SPEC_DEST`   | `/tmp`                                   | Folder to write the spec files to |
|           | `GITHUB_TOKEN`       |                                          | GitHub token (optional)           |

The destination folder is taken from `-dest`, then from `dest` in the config file, then from
`GITHUB2SPEC_DEST`, and falls back to `/tmp`.

Unauthenticated GitHub API requests are limited to 60 per hour. Each repository takes two requests, so set
`GITHUB_TOKEN` when building specs for many repositories, or when running often.

Example:

```
export GITHUB_TOKEN=ghp_...
specbuilder -config conf/github2spec.yaml -dest ./specs
```

```
Writing to specs/pgfga-x86_64.spec
Writing to specs/etcd-x86_64.spec
```

When a repository fails (for example, because it does not exist), specbuilder reports the error, continues with the
other repositories, and exits with a non-zero exit code.

The spec files are generated for the architecture specbuilder runs on. To generate specs for another architecture,
run specbuilder on that architecture, or set `package_arch` and `binary_arch` in the config.

## Configuration

```yaml
# The Jinja2 template used to generate the spec files
template: /usr/rpmbuilder/templates/spec.j2
# Optional destination folder
dest: /host/specs/
repositories:
  pgfga:
    organization: pgvillage-tools
    asset_filter: '{{ repository }}_{{ version }}_linux_{{ binary_arch }}.tar.gz$'
    files:
      - src: '{{ repository }}'
        dest: '%{_bindir}/{{ repository }}'
        mode: '0755'
  etcd:
    organization: etcd-io
    asset_filter: '{{ package_name }}-{{ version }}-linux-{{ binary_arch }}.tar.gz$'
    files:
      - src: 'etcd-{{ version }}-linux-{{ binary_arch }}/etcdctl'
        dest: '%{_bindir}/etcdctl'
      - src: 'etcd-{{ version }}-linux-{{ binary_arch }}/etcd'
        dest: '%{_bindir}/etcd'
```

Every key under `repositories` is a package. Its values are available in the template, and can contain Jinja2
expressions themselves (like `{{ repository }}` above). Templates are rendered repeatedly until the output no longer
changes, so nested expressions are resolved too.

### Variables

All variables can be set (or overridden) per repository in the config.
Values from the config always take precedence over values read from GitHub.

| Variable         | Default / source                                           | Description                                                         |
|------------------|------------------------------------------------------------|---------------------------------------------------------------------|
| `name`           | the key in `repositories`                                  | Name of the entry                                                   |
| `site`           | `github.com`                                               | GitHub site; the API is read from `https://api.<site>`              |
| `repository`     | `name`                                                     | GitHub repository                                                   |
| `organization`   | `repository`                                               | GitHub organization or user                                         |
| `package_name`   | `name`                                                     | RPM package name                                                    |
| `url`            | `https://<site>/<organization>/<repository>`               | Project url                                                         |
| `package_arch`   | current architecture, e.g. `x86_64`, `aarch64`             | RPM architecture, used in the spec file name and `BuildArch`        |
| `binary_arch`    | current Go architecture, e.g. `amd64`, `arm64`             | Architecture as typically used in release asset names               |
| `target_release` | `latest`                                                   | Release to read, e.g. `latest` or `tags/v1.2.3`                     |
| `asset_filter`   | `{{ name }}-{{ version }}-linux-{{ binary_arch }}.tar.gz$` | Regular expression (after rendering) to select release assets       |
| `repo_name`      | GitHub API                                                 | Repository name as reported by GitHub                               |
| `description`    | GitHub API                                                 | Repository description                                              |
| `license`        | GitHub API                                                 | SPDX license id (empty if GitHub reports no license)                |
| `version`        | GitHub API                                                 | Release tag, with `-` replaced by `.` (RPM versions cannot contain `-`) |
| `changelog`      | GitHub API                                                 | Release notes                                                       |
| `assets`         | GitHub API                                                 | List of matching release assets, each with a `name` and `url`       |
| `release_info`   | GitHub API                                                 | Map with `version`, `changelog` and `assets`                        |

Any other key (like `files`, `requires`, `prep` or `group` in the example template) is passed to the template as is.

`asset_filter` uses [Go regular expression syntax](https://pkg.go.dev/regexp/syntax), and matches anywhere in the
asset name unless anchored with `^` and/or `$`.

## Templates

Templates use Jinja2 syntax, rendered with [gonja](https://github.com/nikolalohinski/gonja). Undefined variables
render as empty strings, and loops over undefined lists produce no output.

An example template, which uses `files`, `requires`, `prep` and `group` from the config:

```jinja
%global _prefix /usr/local
Summary: {{ description }}
Name: {{ package_name }}
Version: {{ version }}
Release: 1%{?dist}
License: {{ license }}
Group: {{ group | default('Unspecified') }}
Url: {{ url }}
{% for asset in assets %}Source{{ loop.index0 }}: {{ asset.name }}
{% endfor %}BuildArch: {{ package_arch | default('x86_64') }}
{% for require in requires %}Requires: {{ require }}
{% endfor %}
%description
{{ description }}

%prep
mkdir -p %{_sourcedir}
{% for asset in assets %}curl -L {{ asset.url }} -o %{_sourcedir}/{{ asset.name }}
{% endfor %}
{% for cmd in prep %}{{ cmd }}
{% endfor %}

%install
mkdir -p %{buildroot}/%{_bindir}
{% for asset in assets %}tar -xvf %{_sourcedir}/{{ asset.name }}
{% endfor %}
{% for file in files %}%{__install} -m {{ file.mode | default('0755') }} %{_builddir}/{{file.src}} %{buildroot}/{{file.dest}}
{% endfor %}

%clean
rm -rf $RPM_BUILD_ROOT

%files
{% for file in files %}{{ file.dest }}
{% endfor %}
```

## Development

```
go test ./...
```

| Path                 | Description                                                     |
|----------------------|-----------------------------------------------------------------|
| `cmd/specbuilder`    | Command line tool                                               |
| `pkg/config`         | Reads the YAML config file                                      |
| `pkg/github`         | Minimal GitHub API client for repository and release info       |
| `pkg/spec`           | Builds the template context for a repository and renders specs  |
| `pkg/render`         | Recursive Jinja2 rendering                                      |
