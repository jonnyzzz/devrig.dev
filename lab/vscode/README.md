# VSCode Forks Downloader

This tool downloads the latest versions of popular VS Code forks for all major platforms.

## Supported Products

- **VSCode** - Official Microsoft Visual Studio Code
- **VSCodium** - Open source build without telemetry
- **Cursor** - AI-powered code editor
- **Windsurf** - Codeium's AI IDE

## Supported Platforms

For each product, downloads are available for:
- Windows (x64, ARM64)
- Linux (x64, ARM64)
- macOS/Darwin (x64, ARM64)

## Prerequisites

You need `uv` installed. Install it with:

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

Or on macOS with Homebrew:

```bash
brew install uv
```

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

### Incremental Downloads

The script tracks downloaded versions using `.version` files in each download directory. If a version is already downloaded, it will be skipped. This means:

- Re-running the script is fast and safe
- Only new versions are downloaded
- No bandwidth wasted on re-downloads

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
- The downloaded archive (`.zip`, `.tar.gz`, `.dmg`, etc.)
- A `.version` file with metadata about the download

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

## Directory Structure

After running, you'll have:

```
vscode/
├── download.py          # This script
├── README.md           # This file
├── vscode/            # Official VSCode downloads
│   ├── vscode-1.95.0-windows-arm64/
│   ├── vscode-1.95.0-windows-x64/
│   ├── vscode-1.95.0-linux-arm64/
│   └── ...
├── vscodium/          # VSCodium downloads
│   └── ...
├── cursor/            # Cursor downloads
│   └── ...
└── windsurf/          # Windsurf downloads
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
