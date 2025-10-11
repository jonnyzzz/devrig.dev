---
title: "Download"
url: "/download/"
---

# Download devrig

Get the latest version of devrig for your platform.

## Latest Release

Download the appropriate binary for your operating system:

- [Linux x86-64](/download/devrig-linux-x86_64)
- [Linux ARM64](/download/devrig-linux-arm64)
- [macOS ARM64 (Apple Silicon)](/download/devrig-darwin-arm64)
- [Windows x86-64](/download/devrig-windows-x86_64.exe)
- [Windows ARM64](/download/devrig-windows-arm64.exe)

## Release Information

See [latest.json](/download/latest.json) for current release details including version numbers and checksums.

## Installation

After downloading, make the binary executable (Linux/macOS):

```bash
chmod +x devrig-*
```

Then run:

```bash
./devrig-<platform> start
```

Or use the bootstrap script in your repository to automate download and verification.
