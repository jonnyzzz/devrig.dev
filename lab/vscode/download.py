#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "httpx",
#     "rich",
# ]
# ///

"""
VSCode Forks Downloader
Downloads latest versions of VSCode, VSCodium, Cursor, and Windsurf for all platforms.
Uses incremental downloads with version tracking via touch files.
"""

import asyncio
import hashlib
import json
import re
import sys
from pathlib import Path
from typing import Dict, List, Optional, Tuple
from urllib.parse import urlparse

import httpx
from rich.console import Console
from rich.progress import Progress, SpinnerColumn, TextColumn, BarColumn, DownloadColumn, TransferSpeedColumn

console = Console()

# Base directory for downloads
BASE_DIR = Path(__file__).parent


class Platform:
    """Platform configuration for downloads"""
    def __init__(self, os_name: str, arch: str, prefer_arm: bool = True):
        self.os_name = os_name
        self.arch = arch
        self.prefer_arm = prefer_arm

    def __repr__(self):
        return f"Platform({self.os_name}, {self.arch})"


# Platforms to download
PLATFORMS = [
    Platform("windows", "arm64"),
    Platform("windows", "x64"),
    Platform("linux", "arm64"),
    Platform("linux", "x64"),
    Platform("darwin", "arm64"),  # macOS
    Platform("darwin", "x64"),
]


class VersionInfo:
    """Tracks version information for incremental downloads"""
    def __init__(self, product: str, version: str, os_name: str, arch: str):
        self.product = product
        self.version = version
        self.os_name = os_name
        self.arch = arch
        self.folder_name = f"{product}-{version}-{os_name}-{arch}"
        self.version_file = BASE_DIR / product / self.folder_name / ".version"

    def is_downloaded(self) -> bool:
        """Check if this version is already downloaded"""
        return self.version_file.exists()

    def mark_downloaded(self, download_url: str, file_size: int):
        """Mark this version as downloaded"""
        self.version_file.parent.mkdir(parents=True, exist_ok=True)
        metadata = {
            "product": self.product,
            "version": self.version,
            "os": self.os_name,
            "arch": self.arch,
            "download_url": download_url,
            "file_size": file_size,
        }
        self.version_file.write_text(json.dumps(metadata, indent=2))

    def get_download_path(self) -> Path:
        """Get the directory for this download"""
        return self.version_file.parent


class ProductDownloader:
    """Base class for product downloaders"""

    def __init__(self, name: str):
        self.name = name
        self.client = httpx.AsyncClient(timeout=30.0, follow_redirects=True)

    async def close(self):
        await self.client.aclose()

    async def get_latest_version(self) -> str:
        """Get the latest version number"""
        raise NotImplementedError

    async def get_download_url(self, version: str, platform: Platform) -> Optional[str]:
        """Get download URL for specific version and platform"""
        raise NotImplementedError

    async def download_file(self, url: str, dest_path: Path, progress: Progress, task_id):
        """Download a file with progress tracking"""
        dest_path.parent.mkdir(parents=True, exist_ok=True)

        async with self.client.stream("GET", url) as response:
            assert response.status_code == 200, f"Failed to download {url}: HTTP {response.status_code}"

            total_size = int(response.headers.get("content-length", 0))
            progress.update(task_id, total=total_size)

            with open(dest_path, "wb") as f:
                downloaded = 0
                async for chunk in response.aiter_bytes(chunk_size=8192):
                    f.write(chunk)
                    downloaded += len(chunk)
                    progress.update(task_id, advance=len(chunk))

        return dest_path


class VSCodeDownloader(ProductDownloader):
    """Downloader for official Microsoft VSCode"""

    def __init__(self):
        super().__init__("vscode")
        self.base_url = "https://update.code.visualstudio.com"

    async def get_latest_version(self) -> str:
        """Get latest VSCode version"""
        # Use the update API to get latest version
        url = f"{self.base_url}/api/update/linux-x64/stable/latest"
        response = await self.client.get(url)
        assert response.status_code == 200, f"Failed to get VSCode version: HTTP {response.status_code}"
        data = response.json()
        version = data.get("productVersion")
        assert version, "No version found in VSCode API response"
        return version

    async def get_download_url(self, version: str, platform: Platform) -> Optional[str]:
        """Get VSCode download URL"""
        # Map platform to VSCode platform names
        platform_map = {
            ("windows", "x64"): "win32-x64",
            ("windows", "arm64"): "win32-arm64",
            ("linux", "x64"): "linux-x64",
            ("linux", "arm64"): "linux-arm64",
            ("darwin", "x64"): "darwin",
            ("darwin", "arm64"): "darwin-arm64",
        }

        vscode_platform = platform_map.get((platform.os_name, platform.arch))
        if not vscode_platform:
            return None

        # VSCode uses archive for some platforms
        if platform.os_name == "linux":
            url = f"{self.base_url}/{version}/{vscode_platform}/stable"
        else:
            url = f"{self.base_url}/{version}/{vscode_platform}/stable"

        return url


class VSCodiumDownloader(ProductDownloader):
    """Downloader for VSCodium"""

    def __init__(self):
        super().__init__("vscodium")
        self.repo = "VSCodium/vscodium"

    async def get_latest_version(self) -> str:
        """Get latest VSCodium version from GitHub"""
        url = f"https://api.github.com/repos/{self.repo}/releases/latest"
        response = await self.client.get(url)
        assert response.status_code == 200, f"Failed to get VSCodium version: HTTP {response.status_code}"
        data = response.json()
        version = data.get("tag_name", "").lstrip("v")
        assert version, "No version found in VSCodium release"
        return version

    async def get_download_url(self, version: str, platform: Platform) -> Optional[str]:
        """Get VSCodium download URL from GitHub releases"""
        url = f"https://api.github.com/repos/{self.repo}/releases/tags/{version}"
        response = await self.client.get(url)
        assert response.status_code == 200, f"Failed to get VSCodium release: HTTP {response.status_code}"

        data = response.json()
        assets = data.get("assets", [])

        # Look for appropriate asset
        patterns = {
            ("windows", "x64"): r"VSCodium.*win32-x64.*\.zip$",
            ("windows", "arm64"): r"VSCodium.*win32-arm64.*\.zip$",
            ("linux", "x64"): r"VSCodium.*linux-x64.*\.tar\.gz$",
            ("linux", "arm64"): r"VSCodium.*linux-arm64.*\.tar\.gz$",
            ("darwin", "x64"): r"VSCodium.*darwin-x64.*\.zip$",
            ("darwin", "arm64"): r"VSCodium.*darwin-arm64.*\.zip$",
        }

        pattern = patterns.get((platform.os_name, platform.arch))
        if not pattern:
            return None

        for asset in assets:
            name = asset.get("name", "")
            if re.search(pattern, name, re.IGNORECASE):
                return asset.get("browser_download_url")

        return None


class CursorDownloader(ProductDownloader):
    """Downloader for Cursor IDE"""

    def __init__(self):
        super().__init__("cursor")
        # Use the version history from GitHub repo
        self.version_url = "https://raw.githubusercontent.com/accesstechnology-mike/cursor-downloads/main/version-history.json"
        self._version_data = None

    async def _get_version_data(self):
        """Fetch and cache version data"""
        if self._version_data is None:
            response = await self.client.get(self.version_url)
            assert response.status_code == 200, f"Failed to get Cursor versions: HTTP {response.status_code}"
            data = response.json()
            versions = data.get("versions", [])
            assert len(versions) > 0, "No versions found in Cursor version history"
            self._version_data = versions
        return self._version_data

    async def get_latest_version(self) -> str:
        """Get latest Cursor version"""
        versions = await self._get_version_data()
        latest = versions[0]
        version = latest.get("version")
        assert version, "No version found in Cursor version data"
        return version

    async def get_download_url(self, version: str, platform: Platform) -> Optional[str]:
        """Get Cursor download URL"""
        versions = await self._get_version_data()

        # Find the version
        version_data = None
        for v in versions:
            if v.get("version") == version:
                version_data = v
                break

        if not version_data:
            return None

        platforms = version_data.get("platforms", {})

        # Map our platform to Cursor's platform names
        platform_map = {
            ("windows", "x64"): "win32-x64-user",
            ("windows", "arm64"): "win32-arm64-user",
            ("linux", "x64"): "linux-x64",
            ("linux", "arm64"): "linux-arm64",
            ("darwin", "x64"): "darwin-x64",
            ("darwin", "arm64"): "darwin-arm64",
        }

        cursor_platform = platform_map.get((platform.os_name, platform.arch))
        if not cursor_platform:
            return None

        return platforms.get(cursor_platform)


class WindsurfDownloader(ProductDownloader):
    """Downloader for Windsurf Editor"""

    def __init__(self):
        super().__init__("windsurf")
        self.releases_url = "https://windsurf.com/windsurf/releases"
        self._version_info = None

    async def _get_version_info(self):
        """Fetch version info by scraping releases page"""
        if self._version_info is None:
            # Fetch the releases page
            response = await self.client.get(self.releases_url)
            assert response.status_code == 200, f"Failed to get Windsurf releases page: HTTP {response.status_code}"

            html = response.text

            # Extract a download URL to parse version and commit
            # Pattern: https://windsurf-stable.codeiumdata.com/{platform}/stable/{commit}/{filename}
            import re
            match = re.search(
                r'https://windsurf-stable\.codeiumdata\.com/[^/]+/stable/([a-f0-9]+)/[^-]+-[^-]+-[^-]+-(\d+\.\d+\.\d+)\.',
                html
            )

            assert match, "Could not find version and commit in Windsurf releases page"

            commit_hash = match.group(1)
            version = match.group(2)

            self._version_info = {
                "version": version,
                "commit": commit_hash,
            }

        return self._version_info

    async def get_latest_version(self) -> str:
        """Get latest Windsurf version"""
        info = await self._get_version_info()
        return info["version"]

    async def get_download_url(self, version: str, platform: Platform) -> Optional[str]:
        """Get Windsurf download URL"""
        info = await self._get_version_info()
        commit_hash = info["commit"]

        # Map our platform to Windsurf's platform names and filenames
        # Format: (path_platform, filename_platform, extension)
        platform_map = {
            ("windows", "x64"): ("win32-x64-archive", "win32-x64", "zip"),
            ("windows", "arm64"): ("win32-arm64-archive", "win32-arm64", "zip"),
            ("linux", "x64"): ("linux-x64", "linux-x64", "tar.gz"),
            ("linux", "arm64"): None,  # Not available
            ("darwin", "x64"): ("darwin-x64", "darwin-x64", "zip"),
            ("darwin", "arm64"): ("darwin-arm64", "darwin-arm64", "zip"),
        }

        platform_info = platform_map.get((platform.os_name, platform.arch))
        if not platform_info:
            return None

        path_platform, filename_platform, ext = platform_info
        base = "https://windsurf-stable.codeiumdata.com"

        # Construct download URL
        return f"{base}/{path_platform}/stable/{commit_hash}/Windsurf-{filename_platform}-{version}.{ext}"


async def download_product(downloader: ProductDownloader, platforms: List[Platform]):
    """Download all versions of a product for specified platforms"""
    try:
        console.print(f"\n[bold cyan]Processing {downloader.name}...[/bold cyan]")

        # Get latest version
        version = await downloader.get_latest_version()
        console.print(f"Latest version: [green]{version}[/green]")

        # Check and download for each platform
        downloads = []
        for platform in platforms:
            version_info = VersionInfo(downloader.name, version, platform.os_name, platform.arch)

            if version_info.is_downloaded():
                console.print(f"  ✓ {platform.os_name}-{platform.arch}: [dim]already downloaded[/dim]")
                continue

            download_url = await downloader.get_download_url(version, platform)
            if not download_url:
                console.print(f"  ⚠ {platform.os_name}-{platform.arch}: [yellow]not available[/yellow]")
                continue

            downloads.append((version_info, download_url, platform))

        # Download files with progress tracking
        if downloads:
            with Progress(
                SpinnerColumn(),
                TextColumn("[progress.description]{task.description}"),
                BarColumn(),
                DownloadColumn(),
                TransferSpeedColumn(),
                console=console,
            ) as progress:
                for version_info, download_url, platform in downloads:
                    # Get filename from URL
                    parsed = urlparse(download_url)
                    filename = Path(parsed.path).name
                    if not filename or filename == "stable" or filename == "latest":
                        # Generate filename based on product and platform
                        ext_map = {
                            "windows": ".zip",
                            "linux": ".tar.gz",
                            "darwin": ".zip",
                        }
                        filename = f"{downloader.name}-{platform.os_name}-{platform.arch}{ext_map.get(platform.os_name, '')}"

                    dest_path = version_info.get_download_path() / filename

                    task_id = progress.add_task(
                        f"  ↓ {platform.os_name}-{platform.arch}",
                        total=None
                    )

                    try:
                        await downloader.download_file(download_url, dest_path, progress, task_id)

                        # Mark as downloaded
                        file_size = dest_path.stat().st_size
                        version_info.mark_downloaded(download_url, file_size)

                        progress.update(task_id, description=f"  ✓ {platform.os_name}-{platform.arch}")
                    except Exception as e:
                        progress.update(task_id, description=f"  ✗ {platform.os_name}-{platform.arch}")
                        console.print(f"    [red]Error: {e}[/red]")

        console.print(f"[green]✓ {downloader.name} complete[/green]")

    except Exception as e:
        console.print(f"[red]✗ Failed to process {downloader.name}: {e}[/red]")
        raise
    finally:
        await downloader.close()


async def main():
    """Main entry point"""
    console.print("[bold]VSCode Forks Downloader[/bold]")
    console.print("=" * 50)

    # Create downloaders
    downloaders = [
        VSCodeDownloader(),
        VSCodiumDownloader(),
        CursorDownloader(),
        WindsurfDownloader(),
    ]

    # Download each product
    for downloader in downloaders:
        try:
            await download_product(downloader, PLATFORMS)
        except Exception as e:
            console.print(f"[red]Fatal error with {downloader.name}: {e}[/red]")
            # Continue with other products

    console.print("\n[bold green]All downloads complete![/bold green]")


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        console.print("\n[yellow]Download interrupted by user[/yellow]")
        sys.exit(1)
    except Exception as e:
        console.print(f"\n[red]Fatal error: {e}[/red]")
        sys.exit(1)
