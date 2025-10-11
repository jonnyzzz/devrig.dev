---
title: "Download"
url: "/download/"
---

# Download devrig

Get the latest version of devrig for your platform.

## Latest Release

**Version:** 0.0.1
**Release Date:** January 1, 2025

Download the appropriate binary for your operating system:

- [Linux x86-64](/download/devrig-linux-x86_64)
- [Linux ARM64](/download/devrig-linux-arm64)
- [macOS ARM64](/download/devrig-darwin-arm64)
- [Windows x86-64](/download/devrig-windows-x86_64.exe)
- [Windows ARM64](/download/devrig-windows-arm64.exe)

## Release Information

See [latest.json](/download/latest.json) for current release details including checksums.

### Checksums (SHA-512)

**Linux x86-64:**
```
sha-512:placeholder-hash-linux-x86_64
```

**Linux ARM64:**
```
sha-512:placeholder-hash-linux-arm64
```

**macOS ARM64:**
```
sha-512:placeholder-hash-darwin-arm64
```

**Windows x86-64:**
```
sha-512:placeholder-hash-windows-x86_64
```

**Windows ARM64:**
```
sha-512:placeholder-hash-windows-arm64
```

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

## Quick Start

Place `devrig.cmd` in your project root and run:

```bash
./devrig.cmd start
```

The bootstrap script will automatically download and verify the correct binary for your platform.
