# git-scoper

Applies git `user.name` and `user.email` to every git repository found in a base directory.

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
| `--depth` | `2` | Max directory depth to scan |
| `--workers` | `4` | Parallel workers |

If `base-dir` is omitted, the current directory is used.

## Config

Reads from (in order):
1. `<base-dir>/gitconfig` — key=value format:
   ```
   name=Jane Doe
   email=jane@example.com
   ```
2. `~/.gitconfig` — standard git INI format, reads `[user]` section

## Example

```
$ git-scoper /Work/Acme
Config: name=Jane Doe, email=jane@acme.com
Scanning: /Work/Acme (depth 2)
------------------------
Updated: ProjectAlpha
Updated: tools/ProjectBeta
Skipped: docs
------------------------
Done. 2 updated, 1 skipped.
```
