$ErrorActionPreference = "Stop"

$Repo = "AgusRdz/chop"
$InstallDir = if ($env:CHOP_INSTALL_DIR) { $env:CHOP_INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\chop" }

# Detect architecture
$Arch = if ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture -eq [System.Runtime.InteropServices.Architecture]::Arm64) {
    "arm64"
} else {
    "amd64"
}

$Binary = "chop-windows-$Arch.exe"

# Get latest version
if (-not $env:CHOP_VERSION) {
    $Release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $env:CHOP_VERSION = $Release.tag_name
}

if (-not $env:CHOP_VERSION) {
    Write-Error "failed to determine latest version"
    exit 1
}

$Url = "https://github.com/$Repo/releases/download/$($env:CHOP_VERSION)/$Binary"

Write-Host "installing chop $($env:CHOP_VERSION) (windows/$Arch)..."

# Create install dir
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# Download binary
$Destination = Join-Path $InstallDir "chop.exe"
Invoke-WebRequest -Uri $Url -OutFile $Destination

Write-Host "installed chop to $Destination"
Write-Host ""

# Add to user PATH if not already present
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
$CleanInstallDir = $InstallDir.TrimEnd("\")
$PathParts = $UserPath -split ";" | ForEach-Object { $_.TrimEnd("\") }

if ($PathParts -notcontains $CleanInstallDir) {
    $NewUserPath = "$InstallDir;$UserPath"
    [Environment]::SetEnvironmentVariable("PATH", $NewUserPath, "User")
    Write-Host "added $InstallDir to PATH"
}

# Update current session PATH so it can be used immediately
$CurrentPathParts = $env:PATH -split ";" | ForEach-Object { $_.TrimEnd("\") }
if ($CurrentPathParts -notcontains $CleanInstallDir) {
    $env:PATH = "$InstallDir;$env:PATH"
}

# Update discovery file and notify system of PATH change
& "$Destination" agent-info | Out-Null
$HWND_BROADCAST = [IntPtr]0xffff
$WM_SETTINGCHANGE = 0x001a
$MethodDefinition = @'
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, IntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out IntPtr lpdwResult);
'@
$User32 = Add-Type -MemberDefinition $MethodDefinition -Name "User32" -Namespace "Win32" -PassThru
$result = [IntPtr]::Zero
$User32::SendMessageTimeout($HWND_BROADCAST, $WM_SETTINGCHANGE, [IntPtr]::Zero, "Environment", 2, 100, [ref]$result) | Out-Null

Write-Host "next steps:"
Write-Host ""
Write-Host "  # use directly with any command:"
Write-Host "  chop git status"
Write-Host "  chop docker ps"
Write-Host ""
Write-Host "  # claude code hook (auto-rewrite bash tool calls):"
Write-Host "  chop init --global"
Write-Host "  chop init --status"
Write-Host ""
Write-Host "Installation complete! Please tell your AI Agent:"
Write-Host "  'Remember that chop is installed at $Destination and use it for CLI compression.'"
Write-Host "This will prevent the agent from searching for it in the future."
