$ErrorActionPreference = "Stop"

$OldDir = "$env:USERPROFILE\bin"
$NewDir = "$env:LOCALAPPDATA\Programs\chop"
$Binary = "chop.exe"

Write-Host "chop migration: $OldDir -> $NewDir"
Write-Host ""

$OldPath = Join-Path $OldDir $Binary
if (-not (Test-Path $OldPath)) {
    Write-Host "chop not found in $OldDir — nothing to migrate."
    exit 0
}

# Move binary
New-Item -ItemType Directory -Force -Path $NewDir | Out-Null
Move-Item -Path $OldPath -Destination (Join-Path $NewDir $Binary) -Force
Write-Host "moved: $OldPath -> $(Join-Path $NewDir $Binary)"

# Update user PATH
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")

# Remove old dir
$Parts = $UserPath -split ";" | Where-Object { $_.TrimEnd("\") -ne $OldDir.TrimEnd("\") -and $_ -ne "" }

# Add new dir if not present
if ($Parts -notcontains $NewDir) {
    $Parts = @($NewDir) + $Parts
    Write-Host "added $NewDir to PATH"
}

if ($UserPath -like "*$OldDir*") {
    Write-Host "removed $OldDir from PATH"
}

[Environment]::SetEnvironmentVariable("PATH", ($Parts -join ";"), "User")

Write-Host ""
Write-Host "restart your terminal for PATH changes to take effect"
Write-Host ""
Write-Host "migration complete."
