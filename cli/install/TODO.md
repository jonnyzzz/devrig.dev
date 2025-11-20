# TODO: Security Enhancements for Install Command

## SHA-sum Validation for Font Downloads

### Current Status
The JetBrains Mono font installer currently downloads fonts directly from the official GitHub repository without cryptographic verification.

### Problem
Without checksum validation, there is a potential security risk:
- Man-in-the-middle attacks could substitute malicious files
- Corrupted downloads might not be detected
- No guarantee of file integrity and authenticity

### Research Findings
Based on investigation of the JetBrains Mono GitHub repository:
- **No GPG/PGP signatures** are provided by JetBrains for font releases
- **No SHA-256/SHA-512 checksums** are published alongside releases
- The font is released under the SIL Open Font License 1.1 (OFL-1.1)

### Proposed Solutions

#### Option 1: Maintain Our Own Checksums (Recommended)
1. Create a checksums file in our repository with verified hashes
2. Update this file with each new JetBrains Mono release
3. Validate downloaded files against our checksums before installation

**Implementation Steps:**
- [ ] Create `cli/install/checksums/jetbrains-mono.json` with structure:
  ```json
  {
    "versions": {
      "v2.304": {
        "zip_sha512": "abc123...",
        "release_url": "https://github.com/JetBrains/JetBrainsMono/releases/tag/v2.304"
      }
    }
  }
  ```
- [ ] Add function `validateDownloadChecksum(filePath, expectedHash string) error`
- [ ] Integrate validation into download process
- [ ] Add CI workflow to alert when new JetBrains Mono version is released

#### Option 2: Content-Based Validation
- Validate the ZIP archive structure
- Verify expected font files are present
- Check font file headers for TTF format markers

#### Option 3: Request JetBrains to Provide Checksums
- Open an issue/PR on JetBrains/JetBrainsMono repository
- Request official checksums or signatures for releases
- This would benefit the entire community

### Code Locations
The following locations need updates for SHA-sum validation:

1. **`cli/install/jetbrains_mono.go:31-34`**
   ```go
   // TODO: Validate SHA-sum of the downloaded font
   // Issue: Add checksum validation for downloaded fonts
   // Reference: https://github.com/jonnyzzz/devrig.dev/issues/TBD
   ```

2. **`cli/install/jetbrains_mono.go:126-129`**
   ```go
   // TODO: Validate SHA-sum after download
   // Issue: Add checksum validation for downloaded font archive
   // Reference: https://github.com/jonnyzzz/devrig.dev/issues/TBD
   ```

### Testing Requirements
Once implemented, we need tests for:
- [ ] Valid checksum passes validation
- [ ] Invalid checksum fails validation
- [ ] Missing checksum file is handled gracefully
- [ ] Corrupted download is detected

### Priority
**Medium** - While important for security, the current implementation downloads from the official GitHub repository over HTTPS, which provides some level of security.

### References
- JetBrains Mono Repository: https://github.com/JetBrains/JetBrainsMono
- GitHub API for releases: https://api.github.com/repos/JetBrains/JetBrainsMono/releases/latest
- Similar implementations in package managers: apt, yum, cargo, npm
