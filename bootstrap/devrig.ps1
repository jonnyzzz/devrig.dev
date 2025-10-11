# devrig.ps1 - Bootstrap script for Windows
# This script downloads, verifies, and executes the devrig binary

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

try {
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

    # Get version from config
    $version = ""

    # Compute short hash (first 8 characters)
    $expectedHash = $sha512.ToLower()
    $shortHash = $expectedHash.Substring(0, 8)

    # Construct binary directory path
    $binaryName = "devrig.exe"
    $binaryDir = Join-Path $DEVRIG_HOME "devrig-$os-$cpu-$version$shortHash"
    $binaryPath = Join-Path $binaryDir $binaryName

    # Create devrig home if it doesn't exist
    if (-not (Test-Path $DEVRIG_HOME)) {
        New-Item -ItemType Directory -Path $DEVRIG_HOME -Force | Out-Null
    }

    # Check if binary exists and is valid
    if (Test-Path $binaryPath) {
        Write-Host "[INFO] Found existing devrig binary: $binaryPath"

        # Verify hash
        $actualHash = (Get-FileHash -Path $binaryPath -Algorithm SHA512).Hash.ToLower()

        if ($actualHash -eq $expectedHash) {
            Write-Host "[INFO] Binary checksum verified: $shortHash"
        }
        else {
            Write-Host "[ERROR] Binary checksum mismatch!"
            Write-Host "[ERROR] Expected: $expectedHash"
            Write-Host "[ERROR] Actual:   $actualHash"
            Write-Host "[ERROR] Removing corrupted binary and re-downloading..."
            Remove-Item -Path $binaryDir -Recurse -Force

            # Restart the script
            & $MyInvocation.MyCommand.Path @Arguments
            exit
        }
    }
    else {
        Write-Host "[INFO] Devrig binary not found, downloading..."

        # Create temporary directory for download
        $tempDir = Join-Path $DEVRIG_HOME "devrig-$os-$cpu-$version$shortHash-downloading"
        $tempBinary = Join-Path $tempDir $binaryName

        # Clean up any previous failed downloads
        if (Test-Path $tempDir) {
            Remove-Item -Path $tempDir -Recurse -Force
        }
        New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

        # Download binary with retries
        $maxRetries = 3
        $attempt = 0
        $downloaded = $false

        while ($attempt -lt $maxRetries) {
            $attempt++
            Write-Host "[INFO] Downloading devrig binary (attempt $attempt/$maxRetries)..."

            try {
                $webClient = New-Object System.Net.WebClient
                $webClient.DownloadFile($url, $tempBinary)
                $webClient.Dispose()
                $downloaded = $true
                break
            }
            catch {
                Write-Host "[WARNING] Download failed: $_"

                if ($attempt -lt $maxRetries) {
                    Write-Host "[INFO] Retrying in 2 seconds..."
                    Start-Sleep -Seconds 2
                }
            }
        }

        if (-not $downloaded) {
            Write-Host "[ERROR] Failed to download devrig binary after $maxRetries attempts"
            Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
            exit 1
        }

        # Verify downloaded binary hash
        Write-Host "[INFO] Verifying downloaded binary checksum..."
        $actualHash = (Get-FileHash -Path $tempBinary -Algorithm SHA512).Hash.ToLower()

        if ($actualHash -ne $expectedHash) {
            Write-Host "[ERROR] Downloaded binary checksum mismatch!"
            Write-Host "[ERROR] Expected: $expectedHash"
            Write-Host "[ERROR] Actual:   $actualHash"
            Remove-Item -Path $tempDir -Recurse -Force
            exit 1
        }

        Write-Host "[INFO] Binary checksum verified: $shortHash"

        # Unblock file (Windows security feature)
        Unblock-File -Path $tempBinary -ErrorAction SilentlyContinue

        # Move to production location
        Write-Host "[INFO] Installing devrig binary..."
        Move-Item -Path $tempDir -Destination $binaryDir -Force

        Write-Host "[INFO] Devrig binary installed successfully"
    }

    # Execute devrig binary with all passed arguments
    Write-Host "[INFO] Executing devrig..."

    # Pass all arguments and exit with the same exit code
    $process = Start-Process -FilePath $binaryPath -ArgumentList $Arguments -NoNewWindow -Wait -PassThru
    exit $process.ExitCode
}
catch {
    Write-Host "[ERROR] An unexpected error occurred: $_"
    exit 1
}
