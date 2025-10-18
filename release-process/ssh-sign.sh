#!/bin/bash
set -e

resolve_ssh_agent() {
    local identity_agent=$(ssh -G localhost 2>/dev/null | grep "^identityagent" | sed 's/^identityagent //')

    if [ -n "$identity_agent" ] && [ "$identity_agent" != "none" ]; then
        identity_agent="${identity_agent/#\~/$HOME}"
        if [ -S "$identity_agent" ]; then
            export SSH_AUTH_SOCK="$identity_agent"
            return 0
        fi
    fi

    [ -n "$SSH_AUTH_SOCK" ] && [ -S "$SSH_AUTH_SOCK" ] && return 0

    local default_paths=(
        "/run/user/$(id -u)/keyring/ssh"
        "$HOME/.ssh/agent.sock"
    )

    for path in "${default_paths[@]}"; do
        if [ -S "$path" ]; then
            export SSH_AUTH_SOCK="$path"
            return 0
        fi
    done

    return 1
}

list_keys() {
    resolve_ssh_agent || {
        echo "Error: Could not find SSH agent" >&2
        exit 1
    }

    echo "Using SSH agent: $SSH_AUTH_SOCK" >&2
    ssh-add -L 2>/dev/null || {
        echo "Error: No keys found in ssh-agent" >&2
        exit 1
    }
}

sign_file() {
    local file_to_sign="$1"
    local output_file="$2"
    local key_identifier="$3"

    [ -z "$file_to_sign" ] && { echo "Error: No input file provided" >&2; exit 1; }
    [ -z "$output_file" ] && { echo "Error: No output file provided" >&2; exit 1; }
    [ -z "$key_identifier" ] && { echo "Error: Key identifier is required" >&2; exit 1; }
    [ ! -f "$file_to_sign" ] && { echo "Error: Input file not found: $file_to_sign" >&2; exit 1; }

    resolve_ssh_agent || { echo "Error: Could not find SSH agent" >&2; exit 1; }

    local available_keys=$(ssh-add -L 2>/dev/null)
    [ -z "$available_keys" ] && { echo "Error: No keys in agent" >&2; exit 1; }

    # Key must be a full key line from --list (starts with ssh-)
    if [[ "$key_identifier" != ssh-* ]]; then
        echo "Error: Key must be a full key line from --list output" >&2
        echo "Example: ssh-ed25519 AAAA... comment" >&2
        exit 1
    fi

    local selected_key=$(echo "$available_keys" | grep -F "$key_identifier")
    if [ -z "$selected_key" ]; then
        echo "Error: Provided key not found in agent" >&2
        exit 1
    fi

    echo "Using key: $selected_key" >&2
    echo "Signing file: $file_to_sign" >&2
    echo "Output file: $output_file" >&2

    local temp_key=$(mktemp)
    echo "$selected_key" > "$temp_key"

    ssh-keygen -Y sign -f "$temp_key" -n file < "$file_to_sign" > "$output_file" || {
        rm -f "$temp_key"
        echo "Error: Failed to sign" >&2
        exit 1
    }

    rm -f "$temp_key"
    echo "✓ Signature written to: $output_file" >&2
}

verify_signature_with_key() {
    local file_to_verify="$1"
    local signature_file="$2"
    local public_key="$3"

    [ -z "$file_to_verify" ] && { echo "Error: No file provided" >&2; exit 1; }
    [ -z "$signature_file" ] && { echo "Error: No signature file provided" >&2; exit 1; }
    [ -z "$public_key" ] && { echo "Error: No public key provided" >&2; exit 1; }

    [ ! -f "$file_to_verify" ] && { echo "Error: File not found: $file_to_verify" >&2; exit 1; }
    [ ! -f "$signature_file" ] && { echo "Error: Signature file not found: $signature_file" >&2; exit 1; }

    echo "Verifying signature..." >&2
    echo "  File: $file_to_verify" >&2
    echo "  Signature: $signature_file" >&2
    echo "  Public key: ${public_key:0:50}..." >&2

    # Create temporary allowed_signers file
    local temp_allowed=$(mktemp)
    echo "* $public_key" > "$temp_allowed"

    ssh-keygen -Y verify -f "$temp_allowed" -I '*' -n file -s "$signature_file" < "$file_to_verify" || {
        rm -f "$temp_allowed"
        echo "Error: Signature verification failed" >&2
        exit 1
    }

    rm -f "$temp_allowed"
    echo "✓ Signature verified successfully" >&2
}

verify_signature_with_file() {
    local file_to_verify="$1"
    local signature_file="$2"
    local allowed_signers_file="$3"

    [ -z "$file_to_verify" ] && { echo "Error: No file provided" >&2; exit 1; }
    [ -z "$signature_file" ] && { echo "Error: No signature file provided" >&2; exit 1; }
    [ -z "$allowed_signers_file" ] && { echo "Error: No allowed signers file provided" >&2; exit 1; }

    [ ! -f "$file_to_verify" ] && { echo "Error: File not found: $file_to_verify" >&2; exit 1; }
    [ ! -f "$signature_file" ] && { echo "Error: Signature file not found: $signature_file" >&2; exit 1; }
    [ ! -f "$allowed_signers_file" ] && { echo "Error: Allowed signers file not found: $allowed_signers_file" >&2; exit 1; }

    echo "Verifying signature..." >&2
    echo "  File: $file_to_verify" >&2
    echo "  Signature: $signature_file" >&2
    echo "  Allowed signers: $allowed_signers_file" >&2

    ssh-keygen -Y verify -f "$allowed_signers_file" -I '*' -n file -s "$signature_file" < "$file_to_verify" || {
        echo "Error: Signature verification failed" >&2
        exit 1
    }

    echo "✓ Signature verified successfully" >&2
}

case "${1:-}" in
    -l|--list)
        list_keys
        ;;
    -s|--sign)
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: $0 --sign <input_file> <output_file> <key>" >&2
            exit 1
        fi
        sign_file "$2" "$3" "$4"
        ;;
    -v|--verify)
        if [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ]; then
            echo "Usage: $0 --verify <file> <signature_file> <key_or_allowed_signers>" >&2
            exit 1
        fi
        # Check if the third argument is a file or a key string
        if [ -f "$4" ]; then
            verify_signature_with_file "$2" "$3" "$4"
        else
            verify_signature_with_key "$2" "$3" "$4"
        fi
        ;;
    -h|--help)
        cat << EOF
Usage: $0 [OPTIONS]

Options:
  -l, --list                                           List available SSH keys in agent
  -s, --sign <input> <output> <key>                    Sign a file (key must be full line from --list)
  -v, --verify <file> <sig> <key_or_allowed_signers>  Verify a signature with key or file
  -h, --help                                           Show this help

Examples:
  # List available keys
  $0 --list

  # Sign a file with full key line from --list
  KEY=\$($0 --list | head -1)
  $0 --sign myfile.txt myfile.txt.sig "\$KEY"

  # Verify with a specific key (from --list output)
  KEY=\$($0 --list | head -1)
  $0 --verify myfile.txt myfile.txt.sig "\$KEY"

  # Verify with allowed_signers file
  $0 --verify myfile.txt myfile.txt.sig allowed_signers
EOF
        ;;
    *)
        echo "Error: Invalid option. Use --help for usage" >&2
        exit 1
        ;;
esac
