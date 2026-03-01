# git-scoper

Applies git `user.name` and `user.email` to every git repository found in a base directory.

## Requirements

- Go 1.21+
- `git` must be installed and available on `PATH`

## Install

```bash
go install github.com/belaytzev/git-scoper@latest
```

## Usage

```
git-scoper [flags] [base-dir]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--depth` | `2` | Max directory depth to scan, relative to `base-dir` (depth 1 = immediate children only, depth 2 = one level deeper) |
| `--workers` | `4` | Parallel workers (must be ≥ 1) |

If `base-dir` is omitted, the current directory is used.

If `base-dir` is itself a git repository, config is applied directly to it and no subdirectory scanning occurs.

## Config

Reads from (in order):
1. `<base-dir>/gitconfig` — key=value format:
   ```
   name=Jane Doe
   email=jane@example.com
   ```
   If this file exists but is missing `name=` or `email=`, the tool exits with an error. If the file exists but cannot be accessed (e.g. permission denied) or is a directory, the tool exits with an error rather than falling back to `~/.gitconfig`.
2. `~/.gitconfig` — standard git INI format, reads `[user]` section

## Output

Each scanned path appears as one of:
- `Updated: <path>` — git config was applied successfully
- `Failed: <path> (<reason>)` — git config could not be applied (e.g. not a writable repo)
- `Skipped: <path>` — direct child of `base-dir` that contains no git repository within the scan depth

Directories that cannot be read during scanning are silently ignored and do not appear in any output line.

## Example

```
$ git-scoper /Work/Acme
Config: name=Jane Doe, email=jane@acme.com
Scanning: /Work/Acme (depth 2)
------------------------
Updated: ProjectAlpha
Updated: tools/ProjectBeta
Failed: tools/ReadOnly (git config user.name failed: exit status 128)
Skipped: docs
------------------------
Done. 2 updated, 1 failed, 1 skipped.
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All repos updated successfully |
| `1` | Any failure: invalid flag values (`--depth` or `--workers` less than 1), `base-dir` does not exist or is not a directory, no config found, scan error, or one or more repos reported `Failed:` |

Note: Check individual `Failed:` lines for per-repo error details.
