# chop

<p align="center">
  <img src="logo.png" alt="chop logo" width="200" />
</p>

**CLI output compressor for Claude Code.**

Claude Code wastes 50-90% of its context window on verbose CLI output —
build logs, test results, container listings, git diffs. **chop** compresses
that output before Claude sees it, saving tokens and keeping conversations
focused.

The name comes from _chop chop_: the sound of something eating through all that verbosity before it ever reaches the context window.

## Before & After

```
# Without chop (247 tokens)
$ git status
On branch main
Your branch is up to date with 'origin/main'.

Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
  (use "git restore <file>..." to discard changes in working directory)
        modified:   src/app.ts
        modified:   src/auth/login.ts
        modified:   config.json

Untracked files:
  (use "git add <file>..." to include in what will be committed)
        src/utils/helpers.ts

no changes added to commit (use "git add" and/or "git commit")

# With chop (12 tokens — 95% savings)
$ chop git status
modified(3): src/app.ts, src/auth/login.ts, config.json
untracked(1): src/utils/helpers.ts
```

```
# Without chop (850+ tokens)
$ docker ps
CONTAINER ID   IMAGE                  COMMAND                  CREATED        STATUS        PORTS                    NAMES
a1b2c3d4e5f6   nginx:1.25-alpine      "/docker-entrypoint.…"   2 hours ago    Up 2 hours    0.0.0.0:80->80/tcp       web
f6e5d4c3b2a1   postgres:16-alpine     "docker-entrypoint.s…"   2 hours ago    Up 2 hours    0.0.0.0:5432->5432/tcp   db
...

# With chop (compact table — 70% savings)
$ chop docker ps
web        nginx:1.25-alpine     Up 2h    :80->80
db         postgres:16-alpine    Up 2h    :5432->5432
```

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/AgusRdz/chop/main/install.sh | sh
```

Specific version or custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/AgusRdz/chop/main/install.sh | CHOP_VERSION=v0.10.1 sh
curl -fsSL https://raw.githubusercontent.com/AgusRdz/chop/main/install.sh | CHOP_INSTALL_DIR=/usr/local/bin sh
```

Or with Go:

```bash
go install github.com/AgusRdz/chop@latest
```

Or build from source (requires Docker):

```bash
git clone https://github.com/AgusRdz/chop.git
cd chop
make install    # builds + copies to ~/bin/
```

The installer places the binary in `~/.local/bin` by default. If it is not in your PATH, it is added automatically to `~/.zshrc` or `~/.bashrc`. Reload your shell after installing:

```bash
source ~/.zshrc  # or ~/.bashrc
```

To install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/AgusRdz/chop/main/install.sh | CHOP_INSTALL_DIR=/usr/local/bin sh
```

Update to latest:

```bash
chop update
```

## Quick Start

### Use directly

```bash
chop git status          # compressed git status
chop docker ps           # compact container list
chop npm test            # just failures and summary
chop kubectl get pods    # essential columns only
chop terraform plan      # resource changes, no attribute noise
chop curl https://api.io # JSON compressed to structure + types
chop anything            # auto-detects and compresses any output
```

## Agent Integration

### Claude Code (automatic, zero-config)

Register a PreToolUse hook that automatically wraps every Bash command:

```bash
chop init --global       # install hook
chop init --uninstall    # remove hook
chop init --status       # check if installed
```

After this, every command Claude Code runs gets compressed transparently.
You'll see `chop git status` in the tool calls — that's the hook working.

Add this to your `CLAUDE.md` for best results:

```markdown
## Chop (Token Optimizer)

`chop` is installed globally. It compresses CLI output to reduce token consumption.

When running CLI commands via Bash, prefix with `chop` for read-only commands:
- `chop git status`, `chop git log -10`, `chop git diff`
- `chop docker ps`, `chop npm test`, `chop dotnet build`
- `chop curl <url>` (auto-compresses JSON responses)

Do NOT use chop for: interactive commands, pipes, redirects, or write commands
(git commit, git push, npm init, docker run).
```

## Supported Commands (52+)

| Category | Commands | Savings |
|----------|----------|---------|
| **Git** | `git` status/log/diff/branch, `gh` pr/issue/run | 50-90% |
| **JavaScript** | `npm` install/list/test, `pnpm`, `yarn`, `bun`, `npx`, `tsc`, `eslint`, `biome` | 70-95% |
| **Angular/Nx** | `ng` build/test/serve, `nx` build/test | 70-90% |
| **.NET** | `dotnet` build/test | 70-90% |
| **Rust** | `cargo` test/build/check/clippy | 70-90% |
| **Go** | `go` test/build/vet | 75-90% |
| **Python** | `pytest`, `pip`, `uv`, `mypy`, `ruff`, `flake8`, `pylint` | 70-90% |
| **Java** | `mvn`, `gradle`/`gradlew` | 70-85% |
| **Ruby** | `bundle`, `rspec`, `rubocop` | 70-90% |
| **PHP** | `composer` install/update | 70-85% |
| **Containers** | `docker` ps/build/images/logs/inspect/stats/etc., `docker compose` | 60-85% |
| **Kubernetes** | `kubectl` get/describe/logs/top, `helm` | 60-85% |
| **Infrastructure** | `terraform` plan/apply/init | 70-90% |
| **Build** | `make`, `cmake`, `gcc`/`g++`/`clang` | 60-80% |
| **Cloud** | `aws`, `az`, `gcloud` | 60-85% |
| **HTTP** | `curl`, `http` (HTTPie) | 50-80% |
| **Search** | `grep`, `rg` | 50-70% |
| **System** | `ping`, `ps`, `ss`/`netstat`, `df`/`du` | 50-80% |

Any command not listed above still gets compressed via auto-detection
(JSON, CSV, tables, log lines).

## Token Tracking

Every command is tracked in a local SQLite database:

```bash
chop gain              # overall stats
chop gain --history    # last 20 commands with per-command savings
chop gain --summary    # per-command breakdown
```

```
$ chop gain
chop - token savings report

today: 42 commands, 12,847 tokens saved
total: 318 commands, 89,234 tokens saved (73.2% avg)
```

## Diagnostics

```bash
chop hook-audit        # show last 20 hook rewrite log entries
chop hook-audit --clear
chop config            # show config file path and contents
```

## Migrating from ~/bin

Versions before v0.14.4 installed the binary to `~/bin`. Run the migration script to move it to `~/.local/bin` and update your shell config automatically:

```bash
curl -fsSL https://raw.githubusercontent.com/AgusRdz/chop/main/migrate.sh | sh
```

Or manually:

```bash
mkdir -p ~/.local/bin
mv ~/bin/chop ~/.local/bin/chop
```

Then remove `~/bin` from your shell config (`~/.zshrc` or `~/.bashrc`) and add `~/.local/bin` if it's not already there:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Reload your shell:

```bash
source ~/.zshrc  # or ~/.bashrc
```

## Uninstall & Reset

```bash
chop uninstall                # remove hook, data, config, and binary
chop uninstall --keep-data    # uninstall but preserve tracking history
chop reset                    # clear tracking data and audit log, keep installation
```

## Configuration

`~/.config/chop/config.yml`:

```yaml
# Skip filtering — these commands return full uncompressed output
disabled:
  - curl
  - grep
```

Commands in the `disabled` list bypass all chop filters and return raw output.
Useful when you need the full output for a specific tool (e.g., `curl` responses
you want to parse, or `grep` results you don't want summarized).

## Development

```bash
make test              # run tests
make build             # build (linux, in container)
make install           # build for your platform + install to ~/bin/
make cross             # build all platforms (linux/darwin/windows × amd64/arm64)
make release-patch     # tag + push next patch version
make release-minor     # tag + push next minor version
```

## License

MIT
