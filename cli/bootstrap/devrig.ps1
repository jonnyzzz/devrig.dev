#
# see https://devrig.dev for more details
#

param(
    [Parameter(ValueFromRemainingArguments=$true)]
    [string[]]$Arguments
)

$ErrorActionPreference = "Stop"

# Determine script directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Configuration
$DEVRIG_CONFIG = if ($env:DEVRIG_CONFIG) { $env:DEVRIG_CONFIG } else { Join-Path $ScriptDir "devrig.yaml" }
$DEVRIG_HOME = if ($env:DEVRIG_HOME) { $env:DEVRIG_HOME } else { Join-Path $ScriptDir ".devrig" }

# Log configuration overrides
if ($DEVRIG_CONFIG -ne (Join-Path $ScriptDir "devrig.yaml")) {
    Write-Host "[INFO] Using custom config location: DEVRIG_CONFIG=$DEVRIG_CONFIG"
}

if ($DEVRIG_HOME -ne (Join-Path $ScriptDir ".devrig")) {
    Write-Host "[INFO] Using custom devrig home: DEVRIG_HOME=$DEVRIG_HOME"
}

# Check if config exists
if (-not (Test-Path $DEVRIG_CONFIG)) {
    Write-Host "[ERROR] Configuration file not found: $DEVRIG_CONFIG"
    exit 1
}

# Detect platform
if ($env:DEVRIG_OS) {
    $os = $env:DEVRIG_OS
    Write-Host "[INFO] Using custom OS: DEVRIG_OS=$os"
} else {
    if ($IsWindows -or (-not (Get-Variable IsWindows -ErrorAction SilentlyContinue))) {
        $os = "windows"
    } elseif ($IsLinux) {
        $os = "linux"
    } elseif ($IsMacOS) {
        $os = "darwin"
    } else {
        Write-Host "[ERROR] Unsupported OS"
        exit 1
    }
}

if ($env:DEVRIG_CPU) {
    $cpu = $env:DEVRIG_CPU
    Write-Host "[INFO] Using custom CPU: DEVRIG_CPU=$cpu"
} else {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { $cpu = "x86_64" }
        "ARM64" { $cpu = "arm64" }
        default {
            Write-Host "[ERROR] Unsupported CPU architecture: $arch"
            exit 1
        }
    }
}

# Parse YAML to get URL and hash for current platform
$content = Get-Content $DEVRIG_CONFIG -Raw
$lines = $content -split "`n"

$inDevrig = $false
$inBinaries = $false
$inPlatform = $false
$url = ""
$sha512 = ""

foreach ($line in $lines) {
    if ($url -and $sha512) {
        break
    }

    if ($line -match "^devrig:") {
        $inDevrig = $true
        continue
    }

    if ($inDevrig -and $line -match "^[a-z_]+:" -and $line -notmatch "^\s+") {
        break
    }

    if ($inDevrig -and $line -match "^\s+binaries:") {
        $inBinaries = $true
        continue
    }

    if ($inBinaries -and $line -match "^\s+$os-$cpu`:") {
        $inPlatform = $true
        continue
    }

    if ($inPlatform -and $line -match "^\s+[a-z_-]+:" -and $line -notmatch "^\s+(url|sha512):") {
        break
    }

    if ($inPlatform) {
        if (-not $url -and $line -match "^\s+url:\s*[`"']?([^`"']+)[`"']?") {
            $url = $matches[1].Trim()
        }
        elseif (-not $sha512 -and $line -match "^\s+sha512:\s*[`"']?([^`"']+)[`"']?") {
            $sha512 = $matches[1].Trim()
        }
    }
}

if (-not $url -or -not $sha512) {
    Write-Host "[ERROR] Could not find devrig binary configuration for platform: $os $cpu"
    Write-Host "[ERROR] Please check $DEVRIG_CONFIG"
    exit 1
}

if ($env:DEVRIG_DEBUG_YAML_DOWNLOAD -eq "1") {
    Write-Host $url
    Write-Host $sha512
    exit 44
}

# Create devrig home if it doesn't exist
if (-not (Test-Path $DEVRIG_HOME)) {
    New-Item -ItemType Directory -Path $DEVRIG_HOME -Force | Out-Null
}

# Construct binary path directly with hash (matching sh script)
$DEVRIG_BIN = Join-Path $DEVRIG_HOME "devrig-$os-$cpu-$sha512"
if ($os -eq "windows") {
    $DEVRIG_BIN = "$DEVRIG_BIN.exe"
}

$expectedHash = $sha512.ToLower()

# Helper function to check SHA512 sum
function Test-SHA512Sum {
    param([string]$FilePath)

    try {
        $actualHash = (Get-FileHash -Path $FilePath -Algorithm SHA512).Hash.ToLower()

        if ($actualHash -ne $expectedHash) {
            Write-Host "[ERROR] Downloaded binary checksum mismatch for $FilePath!"
            Write-Host "[ERROR] Expected: $expectedHash"
            Write-Host "[ERROR] Actual:   $actualHash"
            return $false
        }
        return $true
    }
    catch {
        Write-Host "[ERROR] Failed to compute hash: $_"
        return $false
    }
}

# Check if binary exists, if not download it
if (-not (Test-Path $DEVRIG_BIN)) {
    Write-Host "[INFO] Devrig binary not found, downloading..."

    # Create temporary file for download
    $tempBinary = "$DEVRIG_BIN-downloading"

    # Download binary (no retries like sh script)
    try {
        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($url, $tempBinary)
        $webClient.Dispose()
    }
    catch {
        Write-Host "[ERROR] Failed to download devrig binary: $_"
        if (Test-Path $tempBinary) {
            Remove-Item $tempBinary -Force
        }
        exit 1
    }

    if (-not (Test-Path $tempBinary)) {
        Write-Host "[ERROR] Failed to download devrig binary"
        exit 1
    }

    # Verify downloaded binary hash
    Write-Host "[INFO] Verifying downloaded binary checksum..."
    if (-not (Test-SHA512Sum -FilePath $tempBinary)) {
        Remove-Item $tempBinary -Force
        exit 7
    }

    # Unblock file (Windows security feature, only on Windows)
    if ($os -eq "windows") {
        Unblock-File -Path $tempBinary -ErrorAction SilentlyContinue
    }

    # Move to production location
    Write-Host "[INFO] Installing devrig binary..."
    if (Test-Path $DEVRIG_BIN) {
        Remove-Item $DEVRIG_BIN -Force
    }
    Move-Item $tempBinary $DEVRIG_BIN -Force

    Write-Host "[INFO] Devrig binary installed successfully"
}

# Verify the binary hash before execution (matching sh script)
if (-not (Test-SHA512Sum -FilePath $DEVRIG_BIN)) {
    exit 7
}

if ($env:DEVRIG_DEBUG_NO_EXEC -eq "1") {
    Write-Host $url
    Write-Host $sha512
    Write-Host $DEVRIG_BIN
    exit 45
}

# Set DEVRIG_CONFIG environment variable for the tool to use
$env:DEVRIG_CONFIG = $DEVRIG_CONFIG

# Execute devrig binary with all passed arguments
Write-Host "[INFO] Executing devrig..."

# Pass all arguments and exit with the same exit code
$process = Start-Process -FilePath $DEVRIG_BIN -ArgumentList $Arguments -NoNewWindow -Wait -PassThru
exit $process.ExitCode
