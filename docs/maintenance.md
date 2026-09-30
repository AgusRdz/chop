# Maintenance

## Diagnostics

`chop doctor` runs a full health check and auto-fixes what it can:

```bash
chop doctor
```

Checks performed:

- Hook is installed and points to the correct binary (auto-fixes path mismatches)
- Binary is not in the legacy `~/bin` location
- chop is not globally disabled
- Tracking database is accessible and healthy
- Global `config.yml` has valid syntax
- `filters.yml` has valid regex patterns and accessible exec scripts

## Hook audit log

Every command rewrite performed by the hook is logged. Useful for debugging unexpected hook behavior:

```bash
chop hook-audit          # show last 20 hook rewrite log entries
chop hook-audit --clear  # clear the audit log
```

## Updates

```bash
chop update              # update to the latest version manually
chop auto-update         # show current auto-update status
chop auto-update on      # download updates found by the check (applies on next run)
chop auto-update off     # disable auto-updates (you'll get a notification instead)
```

The version check runs only during `chop doctor` and `chop gain`, at most once per 24 hours, with a 3-second lookup timeout. It never runs from wrapped commands or the hook, and chop never spawns a background process for it. With auto-update on, a download prints `chop: downloading update <old> -> <new>...` to stderr and is applied on the next `chop doctor` or `chop gain` run.

## Enable / Disable

Temporarily bypass chop without uninstalling:

```bash
chop disable   # hook passes all commands through uncompressed
chop enable    # resume compression
```

## Uninstall & Reset

```bash
chop uninstall                # remove hook, tracking data, config, and binary
chop uninstall --keep-data    # uninstall but preserve tracking history
chop reset                    # clear tracking data and audit log, keep installation
```

## Migrating from ~/bin

Versions before v0.14.4 (pre v1.0.0) installed the binary to `~/bin`. Run the migration script to move it to the standard location and update your shell config automatically.

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/AgusRdz/chop/main/migrate.sh | sh
source ~/.zshrc  # or ~/.bashrc
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/AgusRdz/chop/main/migrate.ps1 | iex
```

Or manually:

**macOS / Linux:**

```bash
mkdir -p ~/.local/bin
mv ~/bin/chop ~/.local/bin/chop
# remove ~/bin from ~/.zshrc or ~/.bashrc, then add:
export PATH="$HOME/.local/bin:$PATH"
```

**Windows:**

```powershell
New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\Programs\chop"
Move-Item "$env:USERPROFILE\bin\chop.exe" "$env:LOCALAPPDATA\Programs\chop\chop.exe"
[Environment]::SetEnvironmentVariable("PATH", "$env:LOCALAPPDATA\Programs\chop;" + [Environment]::GetEnvironmentVariable("PATH","User"), "User")
```

Restart your terminal after migrating.
