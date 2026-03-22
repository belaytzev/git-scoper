
```
          _ _
   __ _  (_) |_           ___   ___  ___  _ __   ___  _ __
  / _` | | | __|  _____  / __| / __|/ _ \| '_ \ / _ \| '__|
 | (_| | | | |_  |_____| \__ \| (__| (_) | |_) |  __/| |
  \__, | |_|\__|         |___/ \___|\___/| .__/ \___||_|
  |___/                                  |_|
```

# 🔭 git-scoper

> 🎯 Automatically apply git `user.name` and `user.email` to every repository in a directory — so you never commit with the wrong identity again.

[![GitHub release](https://img.shields.io/github/v/release/belaytzev/git-scoper?style=flat-square)](https://github.com/belaytzev/git-scoper/releases)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](https://opensource.org/licenses/MIT)
[![CI](https://img.shields.io/github/actions/workflow/status/belaytzev/git-scoper/ci.yml?style=flat-square&label=CI)](https://github.com/belaytzev/git-scoper/actions)
[![Homebrew](https://img.shields.io/badge/Homebrew-tap-FBB040?style=flat-square&logo=homebrew)](https://github.com/belaytzev/homebrew-tap)

---

## 📖 Table of Contents

- [✨ Why git-scoper?](#-why-git-scoper)
- [🚀 Quick Start](#-quick-start)
- [📦 Installation](#-installation)
- [🛠️ Usage](#️-usage)
- [⚙️ Configuration](#️-configuration)
- [🔄 How It Works](#-how-it-works)
- [📋 Output](#-output)
- [💡 Example](#-example)
- [🚦 Exit Codes](#-exit-codes)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)

---

## ✨ Why git-scoper?

Ever committed code with your **personal email** on a **work repo**? Or vice versa? 😬

Managing git identities across dozens of repositories is tedious and error-prone. **git-scoper** fixes that in one command.

- 🏷️ **Scope your identity** — Apply `user.name` and `user.email` to every repo under a directory
- ⚡ **Fast & parallel** — Scans directories concurrently with configurable worker count
- 🧠 **Smart config resolution** — Reads from a local `gitconfig` file or falls back to `~/.gitconfig`
- 📂 **Depth control** — Configure how deep to scan for nested repositories
- 🔒 **Safe by default** — Never silently ignores bad config; exits with clear errors
- 🍺 **Easy install** — Available via Homebrew or `go install`

---

## 🚀 Quick Start

```bash
# Install via Homebrew
brew tap belaytzev/tap && brew install git-scoper

# Create a config file in your work directory
echo -e "name=Jane Doe\nemail=jane@acme.com" > ~/Work/gitconfig

# Apply git identity to all repos under ~/Work
git-scoper ~/Work
```

That's it! 🎉 Every git repository under `~/Work` now has the correct identity configured.

---

## 📦 Installation

### 🍺 Homebrew (macOS / Linux)

```bash
brew tap belaytzev/tap
brew install git-scoper
```

### 🔧 From Source

Requires Go 1.21+:

```bash
go install github.com/belaytzev/git-scoper@latest
```

### 📋 Prerequisites

- `git` must be installed and available on `PATH`

---

## 🛠️ Usage

```
git-scoper [flags] [base-dir]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--depth` | `2` | 📏 Max directory depth to scan, relative to `base-dir` (depth 1 = immediate children only, depth 2 = one level deeper) |
| `--workers` | `4` | 🔀 Parallel workers (must be ≥ 1) |

- If `base-dir` is omitted, the current directory is used.
- If `base-dir` is itself a git repository, config is applied directly to it and no subdirectory scanning occurs.

---

## ⚙️ Configuration

git-scoper reads configuration from the following sources (in priority order):

### 1️⃣ Local config: `<base-dir>/gitconfig`

A simple key=value format:

```
name=Jane Doe
email=jane@example.com
```

> ⚠️ If this file exists but is missing `name=` or `email=`, the tool exits with an error. If the file exists but cannot be accessed (e.g. permission denied) or is a directory, the tool exits with an error rather than falling back to `~/.gitconfig`.

### 2️⃣ Global config: `~/.gitconfig`

Standard git INI format — reads the `[user]` section.

---

## 🔄 How It Works

```
┌─────────────────────────────────────────────────────┐
│                   git-scoper                        │
├─────────────────────────────────────────────────────┤
│                                                     │
│  1. 📖 Read Config                                  │
│     └─ <base-dir>/gitconfig  ──or──  ~/.gitconfig   │
│                                                     │
│  2. 🔍 Scan Directory                               │
│     └─ Walk <base-dir> up to --depth levels         │
│        looking for .git/ directories                │
│                                                     │
│  3. ⚡ Apply in Parallel                            │
│     └─ N workers run `git config user.name`         │
│        and `git config user.email` concurrently     │
│                                                     │
│  4. 📊 Report Results                               │
│     └─ Updated ✅ │ Failed ❌ │ Skipped ⏭️          │
│                                                     │
└─────────────────────────────────────────────────────┘
```

---

## 📋 Output

Each scanned path appears as one of:

| Status | Meaning |
|--------|---------|
| ✅ `Updated: <path>` | Git config was applied successfully |
| ❌ `Failed: <path> (<reason>)` | Git config could not be applied (e.g. not a writable repo) |
| ⏭️ `Skipped: <path>` | Direct child of `base-dir` that contains no git repository within the scan depth |

> 📝 Directories that cannot be read during scanning are silently ignored and do not appear in any output line.

---

## 💡 Example

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

---

## 🚦 Exit Codes

| Code | Meaning |
|------|---------|
| `0` ✅ | All repos updated successfully |
| `1` ❌ | Any failure: invalid flag values (`--depth` or `--workers` less than 1), `base-dir` does not exist or is not a directory, no config found, scan error, or one or more repos reported `Failed:` |

> 💡 **Tip**: Check individual `Failed:` lines for per-repo error details.

---

## 🤝 Contributing

Contributions are welcome! 🎉

1. 🍴 Fork the repository
2. 🌿 Create your feature branch (`git checkout -b feature/amazing-feature`)
3. 💾 Commit your changes (`git commit -m 'Add amazing feature'`)
4. 📤 Push to the branch (`git push origin feature/amazing-feature`)
5. 🔃 Open a Pull Request

Please make sure your code passes CI checks before submitting.

---

## 📄 License

MIT &copy; [Alexey Belaytzev](https://github.com/belaytzev)

See [LICENSE](LICENSE) for details.

---

<p align="center">
  Made with ❤️ for developers who juggle multiple git identities
</p>
