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

## Config

Reads from (in order):
1. `<base-dir>/gitconfig` — key=value format:
   ```
   name=Jane Doe
   email=jane@example.com
   ```
   If this file exists but is missing `name=` or `email=`, the tool exits with an error.
2. `~/.gitconfig` — standard git INI format, reads `[user]` section

## Output

Each scanned path appears as one of:
- `Updated: <path>` — git config was applied successfully
- `Failed: <path> (<reason>)` — git config could not be applied (e.g. not a writable repo)
- `Skipped: <path>` — direct child of `base-dir` that contains no git repository within the scan depth

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
| `0` | Success (including when some repos report `Failed:`) |
| `1` | Fatal error: inaccessible directory, no config found, or scan error |

Note: `Failed:` repos do not affect the exit code. Check individual lines for per-repo errors.
