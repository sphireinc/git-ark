# git-ark

<p align="center">
  <img src="./logo-transparent.png" alt="git-ark logo" width="400" />
</p>

<p align="center">
  <a href="https://github.com/sphireinc/git-ark/actions/workflows/ci.yml">
    <img alt="CI" src="https://github.com/sphireinc/git-ark/actions/workflows/ci.yml/badge.svg?branch=main" />
  </a>
  <a href="https://github.com/sphireinc/git-ark/actions/workflows/release.yml">
    <img alt="Release" src="https://github.com/sphireinc/git-ark/actions/workflows/release.yml/badge.svg" />
  </a>
  <a href="./LICENSE">
    <img alt="License" src="https://img.shields.io/badge/license-MIT-blue.svg" />
  </a>
</p>

`git-ark` is a CLI tool written in Go for backing up a local Git repository to one or more configured remotes.

It is built around a simple idea: keep the default backup path safe, explicit, and easy to audit.

While service-side technical failures are rare on platforms like GitHub, GitLab, Bitbucket, et al. - they can still happen.

This project was created to mitigate data loss as a result of service-side technical failures, account loss, or some 
other act of (digital) nature. 

Docs site: <https://sphireinc.github.io/git-ark/>

## What It Does

- Backs up branches and tags to multiple remotes.
- Uses mirror mode only when you ask for it.
- Can create a local bundle archive alongside remote backups.
- Synchronizes local remotes from config when you want the repo to match the file.
- Surfaces repo health, remote reachability, and provider-specific guidance.
- Records backup history in metadata so `status` can show what happened last time.

## Installation

### Build From Source

```bash
go build ./cmd/git-ark
```

### Run Without Installing

```bash
go run ./cmd/git-ark --help
```

### Tagged Builds

Tagged pushes trigger the release workflow in [.github/workflows/release.yml](./.github/workflows/release.yml), which builds binaries for Linux, macOS, and Windows, creates a GitHub Release, and attaches release assets plus checksums.  Those can be found and downloaded from [Releases](https://github.com/sphireinc/git-ark/releases).

## Quick Start

1. Generate a starter config:

```bash
git-ark init
```

2. Review the generated file and point `remotes` at your backup destination.
3. Validate the repo and config:

```bash
git-ark validate
```

4. Run a dry run first if you want to see the plan:

```bash
git-ark backup --dry-run
```

5. Run the real backup:

```bash
git-ark backup
```

6. Check the latest metadata and repo health:

```bash
git-ark status
git-ark doctor
```

## Commands

| Command | What it does |
| --- | --- |
| `git-ark init` | Writes a starter `git-ark.yml` file. |
| `git-ark validate` | Checks the config and local repo for obvious problems. |
| `git-ark backup` | Runs the backup workflow against configured remotes. |
| `git-ark status` | Shows repo state, configured remotes, and backup history. |
| `git-ark doctor` | Runs health checks and provider-aware diagnostics. |
| `git-ark remotes list` | Lists configured remotes and whether they exist locally. |
| `git-ark remotes sync` | Adds or updates local Git remotes from config. |
| `git-ark version` | Prints build/version information. |

## Common Flags

- `--config`: use an explicit config file path.
- `--repo`: override the repo directory.
- `--dry-run`: show the plan without mutating remotes or creating bundles.
- `--json`: emit machine-readable output where supported.

`git-ark backup` also supports:

- `--mode`: override the configured mode.
- `--remote`: run against a selected subset of remotes.
- `--include-disabled`: allow selected remotes that are disabled in config.
- `--yes`: skip the mirror-mode confirmation prompt.
- `--verbose`: print more target detail.
- `--quiet`: suppress normal plan and summary output.
- `--prune`: prune remote refs not present locally.

## Configuration

`git-ark init` generates a commented starter file. A minimal config can be as small as:

```yaml
remotes:
  github:
    url: git@github.com:example/example-backup.git
```

A more complete config looks like this:

```yaml
version: 1
repo: "."
mode: "safe"
remotes:
  github:
    url: git@github.com:example/example-backup.git
    enabled: true
    required: true
    provider: github
    description: GitHub backup mirror
  gitlab:
    url: git@gitlab.com:example/example-backup.git
    enabled: true
    required: true
    provider: gitlab
options:
  manage_remotes: true
  push_branches: true
  push_tags: true
  push_notes: false
  push_all_refs: false
  prune: false
  verify_clean_worktree: false
  continue_on_error: true
  confirm_dangerous_operations: true
  fetch_before_backup: false
  pull_before_backup: false
  skip_lfs: false
  include_archived_branches: true
  write_metadata: true
branch_filters:
  include:
    - main
    - release/*
  exclude:
    - wip/*
tag_filters:
  include:
    - v*
  exclude:
    - test-*
bundle:
  enabled: true
  path: ./backups
  filename_template: "{{repo}}-{{timestamp}}.bundle"
  include_all_refs: true
logging:
  level: debug
  format: text
metadata:
  path: .git/git-ark-last-backup.json
```

### Top-Level Fields

| Field | Purpose |
| --- | --- |
| `version` | Config schema version. Current value: `1`. |
| `repo` | Default repo path to operate on. |
| `mode` | Backup mode: `safe`, `mirror`, or `bundle`. |
| `remotes` | Map of remote names to remote definitions. |
| `options` | Backup behavior toggles. |
| `branch_filters` | Include/exclude glob filters for branches. |
| `tag_filters` | Include/exclude glob filters for tags. |
| `bundle` | Local bundle archive settings. |
| `logging` | Output format settings. |
| `metadata` | Path for the last-backup history file. |

### Remote Fields

| Field | Purpose |
| --- | --- |
| `url` | Remote URL. Required. |
| `enabled` | Whether the remote is part of a normal backup run. |
| `required` | Whether a failure on this remote should count as a required failure. |
| `provider` | Provider hint for diagnostics, validation, and friendlier output. |
| `description` | Human-friendly note for maintainers. |

Useful provider hints:

- `github`
- `gitlab`
- `bitbucket`
- `codeberg`
- `gitea`
- `generic`
- `ssh`
- `https`

### Option Groups

- Safety and sync behavior:
  - `manage_remotes`
  - `confirm_dangerous_operations`
  - `continue_on_error`
  - `prune`
  - `push_all_refs`
- Backup scope:
  - `push_branches`
  - `push_tags`
  - `push_notes`
  - `include_archived_branches`
  - `skip_lfs`
- Preflight steps:
  - `verify_clean_worktree`
  - `fetch_before_backup`
  - `pull_before_backup`
- History and output:
  - `write_metadata`

`push_all_refs` is broader than the normal safe mode. `prune` is opt-in and narrows the push behavior, but it can still delete remote refs that are no longer present locally.

## Modes

### Safe Mode

Safe mode is the default. It pushes branches and tags without mirroring every ref.

```bash
git push <remote> --all
git push <remote> --tags
```

### Mirror Mode

Mirror mode is explicit and destructive.

```bash
git push <remote> --mirror
```

If confirmation is enabled, `git-ark` requires `--yes` in non-interactive environments.

### Bundle Mode

Bundle mode creates a local Git bundle archive in addition to the selected backup behavior.

```bash
git bundle create <output> --all
```

## Filters

Branch and tag filters use glob matching.

If an include list is empty, everything is included unless excluded.

```yaml
branch_filters:
  include:
    - main
    - release/*
  exclude:
    - wip/*

tag_filters:
  include:
    - v*
  exclude:
    - test-*
```

## Remote Management

`git-ark remotes sync` is useful when you want the local repo to match the config file.

```bash
git-ark remotes list
git-ark remotes sync
```

Use `--dry-run` if you only want to see what would change.

## Metadata and History

After a successful backup, `git-ark` writes a history file to `.git/git-ark-last-backup.json` by default.

`git-ark status` reads that history and shows:

- The latest run
- The number of recorded backups
- A short recent history summary

If you move the metadata file in config, `status` follows that path.

## Doctor and Provider Diagnostics

`git-ark doctor` checks:

- Whether `git` is available
- Whether `ssh` is available
- Whether the repo looks valid
- Whether the local branch/tag state looks reasonable
- Whether configured remotes are reachable
- Whether configured providers look like they match the remote URL

If provider and host do not line up, `doctor` will call that out. That is intentional and usually means the URL or the provider field needs a quick review.

## Troubleshooting

- If mirror mode fails in a non-interactive shell, pass `--yes`.
- If HTTPS remotes fail, try a token-based URL or switch to SSH.
- If SSH remotes fail, make sure your agent is running and the key is loaded.
- If `validate` says no remotes are configured, add at least one remote entry.
- If `doctor` warns about a provider mismatch, check the provider value and the remote host.
- If `status` shows no history, run `git-ark backup` once with `options.write_metadata: true`.

## Windows Notes

- Use a recent Git for Windows installation.
- Make sure `git` is on `PATH`.
- Forward-compatible SSH URLs like `git@github.com:org/repo.git` work well.
- If you rely on SSH, make sure an agent is running and your key is loaded.

## Security

- Secrets are never printed intentionally.
- HTTPS credentials in URLs are redacted in output.
- Mirror mode can delete refs on the destination, so it requires explicit confirmation.
- This tool shells out to the installed `git` binary; it does not implement a custom Git protocol client.

## Limitations

- v1 does not remove local remotes.
- v1 does not implement a custom Git protocol client.
- v1 does not attempt to reconcile remote history beyond the selected push mode.
- Archived-branch handling is still limited to configuration and planning.
- Full Git LFS backup is not implemented; `skip_lfs` only skips smudge downloads during preflight fetch/pull.

## GitHub Actions

- [`.github/workflows/ci.yml`](./.github/workflows/ci.yml) runs tests, vet, and build on Linux, macOS, and Windows.
- [`.github/workflows/release.yml`](./.github/workflows/release.yml) builds tagged binaries for Linux, macOS, and Windows and publishes GitHub Releases.
- [`.github/workflows/pages.yml`](./.github/workflows/pages.yml) deploys the docs site to GitHub Pages from `docs/`.
