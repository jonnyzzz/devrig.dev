# SSH Agent Signing

Sign strings using SSH keys from ssh-agent on macOS.

## Features

- Automatically resolves SSH agent from SSH config (`IdentityAgent`)
- Works with 1Password SSH agent and other agents
- Supports key selection by identifier or uses first available key
- Generates SSH signatures using `ssh-keygen -Y sign`

## Usage

### List available keys

```bash
./ssh-sign.sh --list
```

### Sign a string

```bash
# Sign with first available key
./ssh-sign.sh --sign "message to sign"

# Sign with specific key (matches key comment)
./ssh-sign.sh --sign "message to sign" "devrig key"
```

The signature is output to stdout in SSH signature format.

## Requirements

- macOS
- SSH agent running (1Password, ssh-agent, etc.)
- Keys loaded in the agent
- `ssh-keygen` with `-Y sign` support (OpenSSH 8.0+)

## How it works

The script:
1. Resolves the SSH agent socket from:
   - SSH config `IdentityAgent` directive
   - `SSH_AUTH_SOCK` environment variable
   - Default paths (1Password, system keyring)
2. Lists keys from the agent using `ssh-add -L`
3. Selects the appropriate key
4. Signs the data using `ssh-keygen -Y sign`

## Configuration

To use with 1Password or custom agent, configure in `~/.ssh/config`:

```
Host *
  IdentityAgent "~/Library/Group Containers/2BUA8C4S2C.com.1password/t/agent.sock"
```
