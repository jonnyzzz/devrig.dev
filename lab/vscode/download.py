#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "httpx",
#     "rich",
#     "PySquashfsImage>=0.9.0",
#     "py7zr>=0.22.0",
#     "zstandard>=0.23.0",
#     "libarchive-c>=5.1",
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
import shutil
import struct
import subprocess
import sys
import tarfile
import zipfile
from pathlib import Path
from typing import Dict, List, Optional, Tuple
from urllib.parse import urlparse

import httpx
import libarchive
import py7zr
from PySquashfsImage import SquashFsImage
from PySquashfsImage.extract import extract_dir as extract_squashfs_dir
from rich.console import Console
from rich.progress import Progress, SpinnerColumn, TextColumn, BarColumn, DownloadColumn, TransferSpeedColumn

console = Console()

# Base directory for downloads
BASE_DIR = Path(__file__).parent

# Tools directory for extraction binaries
TOOLS_DIR = BASE_DIR / ".tools"
TOOLS_DIR.mkdir(exist_ok=True)


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
        """Check if this version is already downloaded and unpacked"""
        if not self.version_file.exists():
            return False

        # Check if unpacked flag is set
        try:
            metadata = json.loads(self.version_file.read_text())
            return metadata.get("unpacked", False)
        except:
            return False

    def mark_downloaded(self, download_url: str, file_size: int, unpacked: bool = False):
        """Mark this version as downloaded and optionally unpacked"""
        self.version_file.parent.mkdir(parents=True, exist_ok=True)
        metadata = {
            "product": self.product,
            "version": self.version,
            "os": self.os_name,
            "arch": self.arch,
            "download_url": download_url,
            "file_size": file_size,
            "unpacked": unpacked,
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

        url = platforms.get(cursor_platform)

        # For Linux, the URL points to .zsync file, but we want the actual AppImage
        if url and platform.os_name == "linux" and url.endswith(".zsync"):
            url = url[:-6]  # Remove .zsync extension to get the AppImage

        return url


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


async def ensure_extraction_tools():
    """
    Ensure extraction tools (unsquashfs, 7z, innoextract) are available.
    First checks system PATH, then downloads if needed.
    Returns dict with tool paths.
    """
    tools = {}

    # Check for unsquashfs
    unsquashfs_path = shutil.which("unsquashfs")
    if unsquashfs_path:
        tools["unsquashfs"] = Path(unsquashfs_path)
        console.print(f"[dim]Found unsquashfs: {unsquashfs_path}[/dim]")
    else:
        console.print("[yellow]unsquashfs not found in PATH[/yellow]")
        console.print("[yellow]Install with: brew install squashfs (macOS) or apt-get install squashfs-tools (Linux)[/yellow]")

    # Check for 7z/7za/7zz
    for cmd in ["7z", "7za", "7zz"]:
        tool_path = shutil.which(cmd)
        if tool_path:
            tools["7z"] = Path(tool_path)
            console.print(f"[dim]Found {cmd}: {tool_path}[/dim]")
            break
    else:
        console.print("[yellow]7z not found in PATH[/yellow]")
        console.print("[yellow]Install with: brew install p7zip (macOS) or apt-get install p7zip-full (Linux)[/yellow]")

    # Check for innoextract
    innoextract_path = shutil.which("innoextract")
    if innoextract_path:
        tools["innoextract"] = Path(innoextract_path)
        console.print(f"[dim]Found innoextract: {innoextract_path}[/dim]")
    else:
        console.print("[yellow]innoextract not found in PATH[/yellow]")
        console.print("[yellow]Install with: brew install innoextract (macOS) or apt-get install innoextract (Linux)[/yellow]")

    return tools


def extract_appimage_with_unsquashfs(appimage_path: Path, extract_dir: Path, unsquashfs_path: Path) -> bool:
    """
    Extract AppImage using unsquashfs binary.
    """
    try:
        console.print(f"    Extracting AppImage with unsquashfs: {appimage_path.name}")

        # Read the AppImage file to find SquashFS offset
        with open(appimage_path, 'rb') as f:
            data = f.read()

        # Find SquashFS magic signature
        # AppImages often have the magic bytes in the code section, so we need to find
        # the actual SquashFS header which is typically aligned on a reasonable boundary
        squashfs_magic = b'hsqs'

        # Find all occurrences of the magic bytes
        offsets = []
        start = 0
        while True:
            offset = data.find(squashfs_magic, start)
            if offset == -1:
                break
            offsets.append(offset)
            start = offset + 1

        if not offsets:
            console.print(f"    [yellow]Could not find SquashFS in AppImage[/yellow]")
            return False

        # Try offsets in reverse order (later ones are more likely to be the actual filesystem)
        # or offsets that are aligned on 4-byte boundaries
        offsets.sort(reverse=True)
        console.print(f"    Found {len(offsets)} potential SquashFS locations, trying in reverse order")

        # Try each offset until one works
        temp_squashfs = extract_dir.parent / f"{appimage_path.stem}.squashfs"
        for offset in offsets:
            # Extract the SquashFS portion to a temporary file
            with open(temp_squashfs, 'wb') as f:
                f.write(data[offset:])

            # Use unsquashfs to extract
            result = subprocess.run(
                [str(unsquashfs_path), "-f", "-d", str(extract_dir), str(temp_squashfs)],
                capture_output=True,
                text=True
            )

            if result.returncode == 0:
                # Success!
                temp_squashfs.unlink()
                console.print(f"    Successfully extracted AppImage with unsquashfs (offset: {offset})")
                return True

        # Clean up temp file
        if temp_squashfs.exists():
            temp_squashfs.unlink()

        console.print(f"    [red]unsquashfs failed for all {len(offsets)} potential offsets[/red]")
        return False

    except Exception as e:
        console.print(f"    [red]Failed to extract with unsquashfs: {e}[/red]")
        return False


def extract_appimage(appimage_path: Path, extract_dir: Path, tools: dict) -> bool:
    """
    Extract AppImage by finding and extracting the embedded SquashFS filesystem.
    AppImages are ELF binaries with a SquashFS filesystem appended.
    """
    # Try unsquashfs first if available
    if "unsquashfs" in tools:
        return extract_appimage_with_unsquashfs(appimage_path, extract_dir, tools["unsquashfs"])

    # Fallback to Python libraries
    try:
        console.print(f"    Extracting AppImage: {appimage_path.name}")

        # Read the AppImage file
        with open(appimage_path, 'rb') as f:
            data = f.read()

        # Find SquashFS magic signature: "hsqs" (0x68737173)
        squashfs_magic = b'hsqs'
        offset = data.find(squashfs_magic)

        if offset == -1:
            console.print(f"    [yellow]Could not find SquashFS in AppImage[/yellow]")
            return False

        console.print(f"    Found SquashFS at offset: {offset}")

        # Extract the SquashFS portion to a temporary file
        temp_squashfs = extract_dir.parent / f"{appimage_path.stem}.squashfs"
        with open(temp_squashfs, 'wb') as f:
            f.write(data[offset:])

        # Ensure extract directory doesn't exist yet
        if extract_dir.exists():
            shutil.rmtree(extract_dir)
        extract_dir.mkdir(parents=True)

        # Use PySquashfsImage to extract
        try:
            with SquashFsImage.from_file(str(temp_squashfs)) as image:
                # Use the library's extract function
                extract_squashfs_dir(image.root, str(extract_dir))

            # Clean up temp file
            temp_squashfs.unlink()
            console.print(f"    Successfully extracted AppImage")
            return True

        except Exception as e:
            console.print(f"    [yellow]PySquashfsImage failed: {e}[/yellow]")
            if temp_squashfs.exists():
                temp_squashfs.unlink()

            # Try direct extraction with libarchive on the temp squashfs file
            try:
                console.print(f"    Trying libarchive on SquashFS...")
                with libarchive.file_reader(str(temp_squashfs)) as archive:
                    entry_count = 0
                    for entry in archive:
                        # Extract each entry
                        dest_path = extract_dir / entry.pathname.lstrip('/')
                        dest_path.parent.mkdir(parents=True, exist_ok=True)

                        if entry.isdir:
                            dest_path.mkdir(parents=True, exist_ok=True)
                        elif entry.isfile or entry.islnk:
                            with open(dest_path, 'wb') as f:
                                for block in entry.get_blocks():
                                    f.write(block)

                            # Preserve permissions
                            if entry.mode:
                                try:
                                    dest_path.chmod(entry.mode & 0o777)
                                except:
                                    pass

                        entry_count += 1

                    if entry_count > 0:
                        console.print(f"    Successfully extracted {entry_count} files with libarchive")
                        return True
                    else:
                        console.print(f"    [yellow]No files found in SquashFS[/yellow]")
                        return False

            except Exception as e2:
                console.print(f"    [yellow]libarchive also failed: {e2}[/yellow]")
                # AppImage extraction failed, but keep the AppImage binary as fallback
                console.print(f"    [dim]Keeping AppImage binary (can be executed directly on Linux)[/dim]")
                return True  # Consider this a partial success

    except Exception as e:
        console.print(f"    [red]Failed to process AppImage: {e}[/red]")
        return False


def extract_with_innoextract(exe_path: Path, extract_dir: Path, tool_path: Path) -> bool:
    """
    Extract Inno Setup installer using innoextract binary.
    """
    try:
        console.print(f"    Extracting with innoextract: {exe_path.name}")

        result = subprocess.run(
            [str(tool_path), "-e", "-d", str(extract_dir), str(exe_path)],
            capture_output=True,
            text=True
        )

        if result.returncode == 0:
            # Check if files were extracted (innoextract creates an 'app' subdirectory)
            extracted_files = list(extract_dir.rglob("*"))
            if len(extracted_files) > 0:
                console.print(f"    Successfully extracted {len(extracted_files)} items with innoextract")
                return True

        console.print(f"    [yellow]innoextract failed: {result.stderr[:200]}[/yellow]")
        return False

    except Exception as e:
        console.print(f"    [yellow]Failed to extract with innoextract: {e}[/yellow]")
        return False


def extract_exe_with_7z(exe_path: Path, extract_dir: Path, tool_path: Path) -> bool:
    """
    Extract Windows installer using 7z binary.
    """
    try:
        console.print(f"    Extracting with 7z: {exe_path.name}")

        result = subprocess.run(
            [str(tool_path), "x", f"-o{extract_dir}", str(exe_path), "-y"],
            capture_output=True,
            text=True
        )

        if result.returncode == 0:
            # Check if files were extracted
            extracted_files = list(extract_dir.rglob("*"))
            if len(extracted_files) > 0:
                console.print(f"    Successfully extracted {len(extracted_files)} files with 7z")
                return True
            else:
                console.print(f"    [yellow]No files extracted[/yellow]")
                return False
        else:
            console.print(f"    [yellow]7z extraction failed: {result.stderr}[/yellow]")
            return False

    except Exception as e:
        console.print(f"    [red]Failed to extract with 7z: {e}[/red]")
        return False


def extract_nsis_installer(exe_path: Path, extract_dir: Path, tools: dict) -> bool:
    """
    Try to extract Windows installer using innoextract first, then 7z, then py7zr, then libarchive.
    Many Windows installers use Inno Setup, NSIS, or other formats that can be extracted.
    """
    # Try innoextract first if available (best for Inno Setup installers like VSCode)
    if "innoextract" in tools:
        if extract_with_innoextract(exe_path, extract_dir, tools["innoextract"]):
            return True

    # Try 7z binary if available (works for some installers, but limited for Inno Setup)
    if "7z" in tools:
        if extract_exe_with_7z(exe_path, extract_dir, tools["7z"]):
            return True

    # Try py7zr (works for some NSIS installers)
    try:
        console.print(f"    Extracting with py7zr: {exe_path.name}")

        with py7zr.SevenZipFile(exe_path, 'r') as archive:
            archive.extractall(path=extract_dir)

        console.print(f"    Successfully extracted with py7zr")
        return True

    except Exception as e:
        console.print(f"    [dim]py7zr failed: {e}[/dim]")

    # Try libarchive (supports more formats)
    try:
        console.print(f"    Extracting with libarchive: {exe_path.name}")

        with libarchive.file_reader(str(exe_path)) as archive:
            entry_count = 0
            for entry in archive:
                # Extract each entry
                dest_path = extract_dir / entry.pathname.lstrip('/')
                dest_path.parent.mkdir(parents=True, exist_ok=True)

                if entry.isdir:
                    dest_path.mkdir(parents=True, exist_ok=True)
                elif entry.isfile or entry.islnk:
                    with open(dest_path, 'wb') as f:
                        for block in entry.get_blocks():
                            f.write(block)

                    # Preserve permissions if available
                    if entry.mode:
                        try:
                            dest_path.chmod(entry.mode & 0o777)
                        except:
                            pass

                entry_count += 1

        if entry_count > 0:
            console.print(f"    Successfully extracted {entry_count} entries with libarchive")
            return True
        else:
            console.print(f"    [yellow]No files extracted from installer[/yellow]")
            return False

    except Exception as e:
        console.print(f"    [yellow]Could not extract installer: {e}[/yellow]")
        return False


def unpack_archive(archive_path: Path, extract_dir: Path, tools: dict) -> bool:
    """
    Unpack an archive file to the specified directory.
    Returns True if successful, False otherwise.
    Supports: .zip, .tar.gz, .tgz, .dmg (macOS only), .exe (NSIS), .AppImage (SquashFS extraction)
    """
    archive_path = Path(archive_path)
    extract_dir = Path(extract_dir)

    assert archive_path.exists(), f"Archive file not found: {archive_path}"

    suffix = archive_path.suffix.lower()
    name = archive_path.name.lower()

    try:
        # Check file magic bytes to determine actual type (some files have misleading extensions)
        with open(archive_path, 'rb') as f:
            magic_bytes = f.read(4)

        is_pe = magic_bytes[:2] == b'MZ'  # PE/EXE file
        is_zip = magic_bytes[:2] == b'PK'  # ZIP file

        # Handle different archive types
        if (suffix == ".zip" or name.endswith(".zip")) and is_zip:
            console.print(f"    Unpacking ZIP: {archive_path.name}")
            with zipfile.ZipFile(archive_path, 'r') as zip_ref:
                zip_ref.extractall(extract_dir)
            return True

        elif (suffix == ".zip" or name.endswith(".zip")) and is_pe:
            # File has .zip extension but is actually an EXE (VSCode Windows)
            console.print(f"    File has .zip extension but is actually an EXE installer")
            return extract_nsis_installer(archive_path, extract_dir, tools)

        elif suffix == ".gz" and (name.endswith(".tar.gz") or name.endswith(".tgz")):
            console.print(f"    Unpacking TAR.GZ: {archive_path.name}")
            with tarfile.open(archive_path, 'r:gz') as tar_ref:
                tar_ref.extractall(extract_dir, filter='data')
            return True

        elif suffix == ".dmg":
            # DMG files are disk images, we'll leave them as-is
            # They can be mounted on macOS but don't need extraction
            console.print(f"    DMG file (keeping as-is): {archive_path.name}")
            return True

        elif suffix == ".exe":
            # Try to extract as NSIS installer (VSCode, VSCodium use NSIS)
            return extract_nsis_installer(archive_path, extract_dir, tools)

        elif name.endswith(".appimage"):
            # Extract AppImage by finding and extracting embedded SquashFS
            archive_path.chmod(0o755)  # Make executable first
            return extract_appimage(archive_path, extract_dir, tools)

        else:
            console.print(f"    [yellow]Unknown archive type: {archive_path.name}[/yellow]")
            return False

    except Exception as e:
        console.print(f"    [red]Failed to unpack {archive_path.name}: {e}[/red]")
        return False


async def download_product(downloader: ProductDownloader, platforms: List[Platform], tools: dict):
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

                        file_size = dest_path.stat().st_size

                        # Unpack the downloaded archive
                        progress.update(task_id, description=f"  ⚙ {platform.os_name}-{platform.arch} (unpacking...)")

                        extract_dir = version_info.get_download_path() / "unpacked"
                        extract_dir.mkdir(exist_ok=True)

                        unpacked = unpack_archive(dest_path, extract_dir, tools)

                        # Mark as downloaded and unpacked
                        version_info.mark_downloaded(download_url, file_size, unpacked)

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

    # Ensure extraction tools are available
    console.print("\n[bold]Checking extraction tools...[/bold]")
    tools = await ensure_extraction_tools()

    if not tools:
        console.print("[yellow]No extraction tools found. Some archives may not be extractable.[/yellow]")
        console.print("[yellow]For best results, install:[/yellow]")
        console.print("[yellow]  - squashfs-tools (for AppImage extraction)[/yellow]")
        console.print("[yellow]  - p7zip (for Windows installer extraction)[/yellow]")

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
            await download_product(downloader, PLATFORMS, tools)
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
