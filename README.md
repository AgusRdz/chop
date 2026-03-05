# chop

CLI output compressor for AI coding assistants.

Reduces LLM token consumption 50-98% by filtering and compressing CLI output.
Proxies any command, applies smart filtering for known tools, and auto-detects
JSON/CSV/table/log formats for everything else.

## Install

```bash
# From source (requires Go 1.24+ or Docker)
git clone <repo-url>
cd chop
docker compose run --rm dev go build -o bin/chop .

# Copy binary to PATH
cp bin/chop /usr/local/bin/
```

## Usage

```bash
chop git status        # "modified(3): app.ts, login.ts, config.json"
chop kubectl get pods  # compact table, essential columns only
chop terraform plan    # resource summary, no attribute noise
chop curl https://api  # JSON auto-compressed to structure + types
chop anything          # auto-detects JSON/CSV/table/logs and compresses
```

## Supported Commands

| Category | Command | Subcommands | Savings |
|----------|---------|-------------|---------|
| **Version Control** | `git` | status, log, diff, branch | 60-90% |
| **Version Control** | `gh` | pr list/view/checks, issue list/view, run list/view | 50-87% |
| **JavaScript** | `npm` | install, list, test | 70-90% |
| **JavaScript** | `npx` | jest, vitest, mocha | 80-95% |
| **JavaScript** | `tsc` | (all) | 80-90% |
| **JavaScript** | `eslint` / `biome` | (all) | 80-90% |
| **.NET** | `dotnet` | build, test | 70-90% |
| **Rust** | `cargo` | test, build, check, clippy | 70-90% |
| **Go** | `go` | test, build, vet | 75-90% |
| **Java** | `mvn` | compile, test, package, install, clean, verify, dependency:tree | 70-85% |
| **Java** | `gradle` / `gradlew` | build, test, dependencies, assemble, compileJava, compileKotlin, jar, war, clean | 70-85% |
| **Containers** | `docker` | ps, build, images | 60-80% |
| **Kubernetes** | `kubectl` | get, describe, logs | 60-85% |
| **Infrastructure** | `terraform` | plan, apply, init | 70-90% |
| **Cloud** | `aws` | s3 ls, ec2 describe-instances, logs, (generic JSON) | 60-85% |
| **Cloud** | `az` | vm list, resource list, (generic JSON) | 60-85% |
| **Cloud** | `gcloud` | compute instances list, (generic) | 60-85% |
| **HTTP** | `curl` | (all) | 50-80% |
| **HTTP** | `http` (HTTPie) | (all) | 50-80% |
| **Search** | `grep` / `rg` | (all) | 50-70% |

## chop gain

Track cumulative token savings across all commands:

```bash
chop gain              # summary stats
chop gain --history    # last 20 commands with per-command savings
```

All commands are tracked in a local SQLite database. Use `chop gain` to see
how many tokens you've saved over time.

## Auto-detect

Any command not in the supported list still gets compressed. chop auto-detects:

- **JSON** -- compressed to structure + types (arrays summarized)
- **CSV/TSV** -- column headers + row count
- **Tables** -- essential columns, aligned
- **Log lines** -- deduplicated with counts, grouped by level

This means `chop <anything>` works. Known commands get purpose-built filters;
everything else gets generic compression.

## Development

```bash
docker compose run --rm dev go test ./... -v   # run tests
docker compose run --rm dev go build -o bin/chop .  # build
docker compose run --rm dev go vet ./...       # vet
```

## License

MIT
