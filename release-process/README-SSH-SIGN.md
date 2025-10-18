# SSH Sign - File Signing and Verification Tool

A bash script for signing files and verifying signatures using SSH keys stored in ssh-agent.

## Features

- Sign files using SSH keys from ssh-agent
- Verify signatures with public keys or allowed_signers files
- List available SSH keys
- Automatic SSH agent detection
- Support for Ed25519 and RSA keys

## Prerequisites

- SSH agent running with loaded keys
- `ssh-keygen` command available
- `ssh-add` command available

## Usage

### List Available Keys

Display all SSH keys currently loaded in ssh-agent:

```bash
./ssh-sign.sh --list
```

Output example:
```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDIPpXgnYpUQnJaaGkVfqLtoZVGjsmnphxI9EZB/P0Fq devrig key 1
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQD... devrig key 2
```

### Sign a File

Sign a file and write the signature to an output file. The key parameter is required and must be a full key line from `--list` output (e.g., "ssh-ed25519 AAAA... devrig key 1").

```bash
./ssh-sign.sh --sign <input_file> <output_file> <key>
```

**Example:**

```bash
KEY=$(./ssh-sign.sh --list | head -1)
./ssh-sign.sh --sign myfile.txt myfile.txt.sig "$KEY"
```

The signature will be written in OpenSSH signature format:
```
-----BEGIN SSH SIGNATURE-----
...base64-encoded signature...
-----END SSH SIGNATURE-----
```

### Verify a Signature

Verify a file signature using either a public key string or an allowed_signers file:

```bash
./ssh-sign.sh --verify <file> <signature_file> <key_or_allowed_signers>
```

**Examples:**

Verify with a public key string (from `--list` output):
```bash
KEY=$(./ssh-sign.sh --list | head -1)
./ssh-sign.sh --verify myfile.txt myfile.txt.sig "$KEY"
```

Verify with an allowed_signers file:
```bash
./ssh-sign.sh --verify myfile.txt myfile.txt.sig allowed_signers
```

The tool automatically detects whether the third argument is a file path or a public key string.

### allowed_signers File Format

The `allowed_signers` file follows the OpenSSH format:

```
principal key_type public_key [comment]
```

**Example:**
```
* ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDIPpXgnYpUQnJaaGkVfqLtoZVGjsmnphxI9EZB/P0Fq devrig key 1
* ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQD... devrig key 2
```

The principal (`*` in this case) matches any identity. You can also use specific email addresses or identities:

```
user@example.com ssh-ed25519 AAAAC3... user key
admin@example.com ssh-rsa AAAAB3... admin key
```

**Creating allowed_signers from ssh-agent:**
```bash
./ssh-sign.sh --list | while IFS= read -r key; do
    echo "* $key"
done > allowed_signers
```

### SSH Agent Detection

The script automatically detects SSH agent in the following order:

1. `IdentityAgent` from SSH config (`~/.ssh/config`)
2. `$SSH_AUTH_SOCK` environment variable
3. Common default paths:
   - `/run/user/$(id -u)/keyring/ssh`
   - `$HOME/.ssh/agent.sock`

If no agent is found, the script will exit with an error.

## Command Reference

### Options

| Option | Arguments | Description |
|--------|-----------|-------------|
| `-l, --list` | None | List available SSH keys in agent |
| `-s, --sign` | `<input> <output> <key>` | Sign a file (key must be full line from --list) |
| `-v, --verify` | `<file> <sig> <key_or_file>` | Verify a signature with key or file |
| `-h, --help` | None | Show help message |

### Exit Codes

- `0` - Success
- `1` - Error (agent not found, file missing, verification failed, etc.)

## Examples

### Complete Workflow

1. **List available keys:**
   ```bash
   ./ssh-sign.sh --list
   ```

2. **Sign a file:**
   ```bash
   KEY=$(./ssh-sign.sh --list | head -1)
   ./ssh-sign.sh --sign release.json release.json.sig "$KEY"
   ```
   Output:
   ```
   Using key: ssh-ed25519 AAAA... devrig key 1
   Signing file: release.json
   Output file: release.json.sig
   ✓ Signature written to: release.json.sig
   ```

3. **Create allowed_signers file:**
   ```bash
   ./ssh-sign.sh --list | while IFS= read -r key; do
       echo "* $key"
   done > allowed_signers
   ```

4. **Verify with allowed_signers file:**
   ```bash
   ./ssh-sign.sh --verify release.json release.json.sig allowed_signers
   ```
   Output:
   ```
   Verifying signature...
     File: release.json
     Signature: release.json.sig
     Allowed signers: allowed_signers
   ✓ Signature verified successfully
   ```

5. **Verify with public key directly:**
   ```bash
   KEY=$(./ssh-sign.sh --list | head -1)
   ./ssh-sign.sh --verify release.json release.json.sig "$KEY"
   ```
   Output:
   ```
   Verifying signature...
     File: release.json
     Signature: release.json.sig
     Public key: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDIP...
   ✓ Signature verified successfully
   ```

## Testing

Run the test suite to verify all functionality:

```bash
./test-ssh-sign.sh
```

The test suite includes:
- Listing keys
- Signing with default and specific keys
- Creating allowed_signers files
- Verifying valid signatures (file and key-based)
- Detecting tampered data
- Detecting invalid signatures
- Error handling tests

## Troubleshooting

### "Error: Could not find SSH agent"

**Solution:**
- Ensure SSH agent is running: `eval $(ssh-agent)`
- Add your key: `ssh-add ~/.ssh/id_ed25519`
- Verify: `ssh-add -L`

### "Error: No keys found in ssh-agent"

**Solution:**
- Add your SSH key: `ssh-add ~/.ssh/id_ed25519`
- Verify keys are loaded: `ssh-add -L`

### "Error: Key must be a full key line from --list output"

**Solution:**
- List available keys: `./ssh-sign.sh --list`
- Use the complete key line from the output (e.g., "ssh-ed25519 AAAA... comment")
- Example: `KEY=$(./ssh-sign.sh --list | head -1)`

### "Error: Signature verification failed"

Possible causes:
- File has been modified after signing
- Signature file is corrupted
- Wrong public key used for verification
- Signature was created with a different key

**Solution:**
- Verify the file hasn't been modified
- Check the signature file integrity
- Ensure you're using the correct public key
- Re-sign the file if needed

## Security Notes

- Keep your SSH private keys secure and encrypted
- Only add trusted keys to `allowed_signers` files
- Verify signatures before trusting file contents
- Use strong SSH keys (Ed25519 or RSA 4096-bit recommended)
- Regularly rotate signing keys
- Store `allowed_signers` files in version control for auditability
