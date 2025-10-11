#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// Read latest.json
const latestJsonPath = path.join(__dirname, '../static/download/latest.json');
const latestData = JSON.parse(fs.readFileSync(latestJsonPath, 'utf-8'));

// OS and architecture display names
const osNames = {
  'linux': 'Linux',
  'darwin': 'macOS',
  'windows': 'Windows'
};

const archNames = {
  'x86_64': 'x86-64',
  'arm64': 'ARM64'
};

// Generate download links
const downloadLinks = latestData.releases.map(release => {
  const osName = osNames[release.os] || release.os;
  const archName = archNames[release.arch] || release.arch;
  const fileName = release.url.split('/').pop();

  return `- [${osName} ${archName}](${release.url.replace('https://devrig.dev', '')})`;
}).join('\n');

// Generate download.md content
const downloadContent = `---
title: "Download"
url: "/download/"
---

# Download devrig

Get the latest version of devrig for your platform.

## Latest Release

**Version:** ${latestData.version}
**Release Date:** ${new Date(latestData.releaseDate).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })}

Download the appropriate binary for your operating system:

${downloadLinks}

## Release Information

See [latest.json](/download/latest.json) for current release details including checksums.

### Checksums (SHA-512)

${latestData.releases.map(release => {
  const osName = osNames[release.os] || release.os;
  const archName = archNames[release.arch] || release.arch;
  return `**${osName} ${archName}:**\n\`\`\`\n${release.sha512}\n\`\`\``;
}).join('\n\n')}

## Installation

After downloading, make the binary executable (Linux/macOS):

\`\`\`bash
chmod +x devrig-*
\`\`\`

Then run:

\`\`\`bash
./devrig-<platform> start
\`\`\`

Or use the bootstrap script in your repository to automate download and verification.

## Quick Start

Place \`devrig.cmd\` in your project root and run:

\`\`\`bash
./devrig.cmd start
\`\`\`

The bootstrap script will automatically download and verify the correct binary for your platform.
`;

// Write download.md
const outputPath = path.join(__dirname, '../content/download.md');
fs.writeFileSync(outputPath, downloadContent, 'utf-8');

console.log('✓ Generated download.md from latest.json');
