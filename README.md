# chop

CLI output compressor for AI coding agents.

Reduces token consumption by 50-98% by filtering and compressing CLI output.
Works with **any AI coding agent** — Claude Code, Cursor, Copilot, Aider, or
any tool that runs shell commands and reads the output.

Proxies any command, applies smart filtering for known tools, and auto-detects
JSON/CSV/table/log formats for everything else.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/AgusRdz/chop/main/install.sh | sh
```

Or with a specific version:

```bash
CHOP_VERSION=v0.8.0 curl -fsSL https://raw.githubusercontent.com/AgusRdz/chop/main/install.sh | sh
```

Or build from source (requires Docker):

```bash
git clone https://github.com/AgusRdz/chop.git
cd chop
make test       # run tests
make install    # build for your platform + copy to ~/bin/
```

## Usage

```bash
chop git status        # "modified(3): app.ts, login.ts, config.json"
chop kubectl get pods  # compact table, essential columns only
chop terraform plan    # resource summary, no attribute noise
chop curl https://api  # JSON auto-compressed to structure + types
chop anything          # auto-detects JSON/CSV/table/logs and compresses
```

## Agent Integration

chop works with any AI coding agent. Pick the integration that fits your setup:

### Direct (any agent)

Prefix commands with `chop` in your agent's instructions or prompt:

```
When running CLI commands, prefix with `chop` for read-only commands:
  chop git status, chop docker ps, chop npm test
```

### Shell integration (any agent)

Auto-wraps all supported commands in your shell:

```bash
# bash
echo 'eval "$(chop init bash)"' >> ~/.bashrc

# zsh
echo 'eval "$(chop init zsh)"' >> ~/.zshrc

# fish
chop init fish | source  # add to fish config
```

### Claude Code hook

Automatically rewrites Bash tool calls — zero config after install:

```bash
chop init --global       # register PreToolUse hook
chop init --uninstall    # remove hook
```

### File reading

Language-aware file compression — strips comments and blank lines:

```bash
chop read src/main.go              # minimal: remove comments, collapse blanks
chop read src/main.go --aggressive # also remove imports and all blank lines
chop read src/main.go --lines 50   # smart truncation to 50 lines
chop read src/main.go -n           # with line numbers
```

Supports Go, Rust, Python, JavaScript/TypeScript, C/C++, C#, Java, Ruby,
Shell, HTML/XML, CSS/SCSS, SQL, YAML, Markdown, and more.

## Supported Commands

| Category | Command | Subcommands | Savings |
|----------|---------|-------------|---------|
| **Version Control** | `git` | status, log, diff, branch | 60-90% |
| **Version Control** | `gh` | pr list/view/checks, issue list/view, run list/view | 50-87% |
| **JavaScript** | `npm` | install, list, test | 70-90% |
| **JavaScript** | `pnpm` | install, list | 70-90% |
| **JavaScript** | `yarn` | install | 70-90% |
| **JavaScript** | `bun` | install | 70-90% |
| **JavaScript** | `npx` | jest, vitest, mocha | 80-95% |
| **JavaScript** | `tsc` | (all) | 80-90% |
| **JavaScript** | `eslint` / `biome` | (all) | 80-90% |
| **Angular/Nx** | `ng` | build, test, serve | 70-90% |
| **Angular/Nx** | `nx` | build, test | 70-90% |
| **.NET** | `dotnet` | build, test | 70-90% |
| **Rust** | `cargo` | test, build, check, clippy | 70-90% |
| **Go** | `go` | test, build, vet | 75-90% |
| **Python** | `pytest` | (all) | 70-90% |
| **Python** | `pip` / `pip3` | install, list | 70-85% |
| **Python** | `uv` | install | 70-85% |
| **Python** | `mypy` | (all) | 70-85% |
| **Python** | `ruff` | (all) | 70-85% |
| **Python** | `pylint` | (all) | 70-85% |
| **Java** | `mvn` | compile, test, package, install, clean, verify, dependency:tree | 70-85% |
| **Java** | `gradle` / `gradlew` | build, test, dependencies, assemble, compileJava, compileKotlin, jar, war, clean | 70-85% |
| **Ruby** | `bundle` | install | 70-85% |
| **Ruby** | `rspec` | (all) | 70-90% |
| **Ruby** | `rubocop` | (all) | 70-85% |
| **PHP** | `composer` | install, update | 70-85% |
| **Containers** | `docker` | ps, build, images, logs, inspect, stats, top, diff, history, network ls, volume ls, system df | 60-85% |
| **Containers** | `docker compose` | ps, build, logs, images | 60-85% |
| **Kubernetes** | `kubectl` | get, describe, logs, top, apply, delete | 60-85% |
| **Kubernetes** | `helm` | install, list | 60-85% |
| **Infrastructure** | `terraform` | plan, apply, init | 70-90% |
| **Build Tools** | `make` | (all) | 60-80% |
| **Build Tools** | `cmake` | (all) | 60-80% |
| **Build Tools** | `gcc` / `g++` / `clang` | (all) | 60-80% |
| **Cloud** | `aws` | s3 ls, ec2 describe-instances, logs, (generic JSON) | 60-85% |
| **Cloud** | `az` | vm list, resource list, (generic JSON) | 60-85% |
| **Cloud** | `gcloud` | compute instances list, (generic) | 60-85% |
| **HTTP** | `curl` | (all) | 50-80% |
| **HTTP** | `http` (HTTPie) | (all) | 50-80% |
| **Search** | `grep` / `rg` | (all) | 50-70% |
| **System** | `ping` | (all) | 50-70% |
| **System** | `ps` | (all) | 60-80% |
| **System** | `ss` / `netstat` | (all) | 60-80% |
| **System** | `df` / `du` | (all) | 50-70% |

## Tracking

Track cumulative token savings:

```bash
chop gain              # summary stats
chop gain --history    # last 20 commands with per-command savings
chop gain --summary    # per-command breakdown
```

## Auto-detect

Any command not in the supported list still gets compressed. chop auto-detects:

- **JSON** — compressed to structure + types (arrays summarized)
- **CSV/TSV** — column headers + row count
- **Tables** — essential columns, aligned
- **Log lines** — deduplicated with counts, grouped by level

This means `chop <anything>` works. Known commands get purpose-built filters;
everything else gets generic compression.

## Other Subcommands

```bash
chop config            # show config file path and contents
chop discover          # scan Claude Code logs for missed chop opportunities
chop hook-audit        # show last 20 hook rewrite log entries
chop capture <cmd>     # run command and save raw + filtered output as fixtures
chop help              # show help
chop version           # show version
```

## Configuration

Config file at `~/.config/chop/config.yml`:

```yaml
# Save raw output for LLM re-read
tee: true

# Skip filtering for specific commands
disabled:
  - curl
  - grep
```

## Development

```bash
make test              # run tests
make build             # build (linux, in container)
make install           # build for your platform + install to ~/bin/
make cross             # build all platforms
make clean             # remove binaries
make release-patch     # tag + push next patch version
make release-minor     # tag + push next minor version
```

Version is injected automatically from git tags via `-ldflags`.

## License

MIT
