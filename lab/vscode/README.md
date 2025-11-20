# VSCode Forks Downloader

This tool downloads and unpacks the latest versions of popular VS Code forks for all major platforms.

## Supported Products

- **VSCode** - Official Microsoft Visual Studio Code
- **VSCodium** - Open source build without telemetry
- **Cursor** - AI-powered code editor (including correct AppImage for Linux)
- **Windsurf** - Codeium's AI IDE

## Supported Platforms

For each product, downloads are available for:
- Windows (x64, ARM64) - Installers (.exe) and archives (.zip)
- Linux (x64, ARM64) - TAR.GZ archives and AppImages
- macOS/Darwin (x64, ARM64) - ZIP archives and DMG images

## Prerequisites

### Required: uv

You need `uv` installed. Install it with:

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

Or on macOS with Homebrew:

```bash
brew install uv
```

### Recommended: Extraction Tools

For complete extraction of all archive formats, install these tools:

**macOS (Homebrew):**
```bash
brew install squashfs p7zip innoextract
```

**Ubuntu/Debian:**
```bash
sudo apt-get install squashfs-tools p7zip-full innoextract
```

**Fedora/RHEL:**
```bash
sudo dnf install squashfs-tools p7zip innoextract
```

**Arch Linux:**
```bash
sudo pacman -S squashfs-tools p7zip innoextract
```

**What these tools do:**
- `squashfs-tools` (provides `unsquashfs`) - Extracts AppImage files (Cursor Linux)
- `p7zip` (provides `7z`) - Extracts Windows installers
- `innoextract` - Extracts Inno Setup installers (limited support for newer versions)

**Extraction Compatibility:**
- ✅ **Fully Working**: VSCodium, Windsurf (Windows/macOS/Linux), Cursor & VSCode (macOS/Linux)
- ⚠️  **Limited**: VSCode & Cursor Windows use Inno Setup 6.4.0.1 which `innoextract 1.9` doesn't support yet. The script falls back to `7z` which extracts installer metadata but not full application files.

**Without these tools:** The script will still download all files, but AppImages and Windows installers will be kept as executable binaries rather than being fully extracted. ZIP and TAR.GZ files extract fine without additional tools.

## Usage

Simply run the script:

```bash
./download.py
```

Or with uv explicitly:

```bash
uv run download.py
```

## Features

### Automatic Unpacking

The script automatically unpacks downloaded archives into an `unpacked/` subdirectory:

- **ZIP files** - Extracted to `unpacked/`
- **TAR.GZ files** - Extracted to `unpacked/`
- **AppImage files** - Made executable (no extraction needed)
- **EXE files** - No extraction needed
- **DMG files** - Left as-is (can be mounted on macOS)

### Incremental Downloads

The script tracks downloaded and unpacked versions using `.version` files. If a version is already downloaded and unpacked, it will be skipped. This means:

- Re-running the script is fast and safe
- Only new versions are downloaded
- Only new archives are unpacked
- No bandwidth or time wasted on re-processing

### Version Tracking

Each download is stored in a directory named:
```
{product}-{version}-{os}-{arch}/
```

For example:
```
vscode-1.95.0-darwin-arm64/
cursor-0.42.3-windows-x64/
```

Inside each directory, you'll find:
- The downloaded archive (`.zip`, `.tar.gz`, `.dmg`, `.AppImage`, `.exe`)
- An `unpacked/` directory with extracted contents (for applicable formats)
- A `.version` file with metadata about the download

The `.version` file tracks:
```json
{
  "product": "cursor",
  "version": "2.0.77",
  "os": "linux",
  "arch": "x64",
  "download_url": "https://...",
  "file_size": 238700000,
  "unpacked": true
}
```

### Error Handling

The script includes comprehensive assertions that will fail with clear error messages if:
- API responses are unexpected
- Download URLs are missing
- Network requests fail
- File operations fail

### Progress Tracking

Rich progress bars show:
- Which product is being processed
- Download progress for each file
- Transfer speeds
- Overall status

## Cursor Linux Downloads

The script correctly downloads Cursor for Linux as **AppImage** files, not zsync metadata files. The AppImages are:
- Fully self-contained executables
- Made executable automatically (chmod +x)
- Ready to run on any Linux distribution

Example Linux downloads:
- `Cursor-2.0.77-x86_64.AppImage` (238 MB)
- `Cursor-2.0.77-aarch64.AppImage` (218 MB)

## Directory Structure

After running, you'll have:

```
vscode/
├── download.py          # This script
├── README.md           # This file
├── vscode/            # Official VSCode downloads
│   ├── vscode-1.106.1-windows-arm64/
│   │   ├── VSCodeSetup-arm64.exe
│   │   ├── unpacked/
│   │   └── .version
│   ├── vscode-1.106.1-darwin-arm64/
│   │   ├── vscode-darwin-arm64.zip
│   │   ├── unpacked/
│   │   │   └── Visual Studio Code.app
│   │   └── .version
│   └── ...
├── vscodium/          # VSCodium downloads
│   ├── vscodium-1.105.17075-darwin-arm64/
│   │   ├── VSCodium-darwin-arm64-1.105.17075.zip
│   │   ├── unpacked/
│   │   │   └── VSCodium.app
│   │   └── .version
│   └── ...
├── cursor/            # Cursor downloads
│   ├── cursor-2.0.77-linux-x64/
│   │   ├── Cursor-2.0.77-x86_64.AppImage (executable)
│   │   ├── unpacked/ (empty - AppImage is self-contained)
│   │   └── .version
│   ├── cursor-2.0.77-darwin-arm64/
│   │   ├── Cursor-darwin-arm64.zip
│   │   ├── unpacked/
│   │   │   ├── Cursor.app
│   │   │   └── resources/
│   │   └── .version
│   └── ...
└── windsurf/          # Windsurf downloads
    ├── windsurf-1.12.32-linux-x64/
    │   ├── Windsurf-linux-x64-1.12.32.tar.gz
    │   ├── unpacked/
    │   │   └── Windsurf/
    │   └── .version
    └── ...
```

## Troubleshooting

### "Failed to get X version"

This usually means the API endpoint has changed or is temporarily unavailable. Check your internet connection and try again.

### "not available" for certain platforms

Some products don't provide builds for all platforms. This is expected and not an error.

### Downloads are slow

The script downloads large files (100+ MB each). Consider running on a machine with good bandwidth.

### Want to force re-download

Delete the `.version` file in the specific version directory, or delete the entire product directory.

## Development

The script is designed to be:
- **Self-contained**: Uses uv's inline script metadata (PEP 723)
- **Type-safe**: Uses type hints throughout
- **Maintainable**: Clear class structure for each product
- **Extensible**: Easy to add new products by inheriting from `ProductDownloader`

To add a new product:

1. Create a new class inheriting from `ProductDownloader`
2. Implement `get_latest_version()` and `get_download_url()`
3. Add an instance to the `downloaders` list in `main()`

## License

This tool is provided as-is for downloading publicly available software. Please respect the licenses of the downloaded products.
