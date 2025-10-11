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

    echo "Using SSH agent: $SSH_AUTH_SOCK"
    ssh-add -L 2>/dev/null || {
        echo "Error: No keys found in ssh-agent" >&2
        exit 1
    }
}

sign_string() {
    local string_to_sign="$1"
    local key_identifier="$2"

    [ -z "$string_to_sign" ] && { echo "Error: No string provided" >&2; exit 1; }

    resolve_ssh_agent || { echo "Error: Could not find SSH agent" >&2; exit 1; }

    local available_keys=$(ssh-add -L 2>/dev/null)
    [ -z "$available_keys" ] && { echo "Error: No keys in agent" >&2; exit 1; }

    local selected_key
    if [ -n "$key_identifier" ]; then
        selected_key=$(echo "$available_keys" | grep -F "$key_identifier" | head -1)
        if [ -z "$selected_key" ]; then
            echo "Error: Key '$key_identifier' not found" >&2
            echo "Available keys:" >&2
            echo "$available_keys" | awk '{print $NF}' >&2
            exit 1
        fi
    else
        selected_key=$(echo "$available_keys" | head -1)
    fi

    local key_comment=$(echo "$selected_key" | awk '{for(i=3;i<=NF;i++) printf $i" "; print ""}' | sed 's/ $//')
    echo "Using key: $key_comment" >&2

    local temp_key=$(mktemp)
    local temp_data=$(mktemp)

    echo "$selected_key" > "$temp_key"
    echo -n "$string_to_sign" > "$temp_data"

    ssh-keygen -Y sign -f "$temp_key" -n file < "$temp_data" || {
        rm -f "$temp_key" "$temp_data"
        echo "Error: Failed to sign" >&2
        exit 1
    }

    rm -f "$temp_key" "$temp_data"
}

case "${1:-}" in
    -l|--list)
        list_keys
        ;;
    -s|--sign)
        [ -z "$2" ] && { echo "Usage: $0 --sign <string> [key_identifier]" >&2; exit 1; }
        sign_string "$2" "$3"
        ;;
    -h|--help)
        cat << EOF
Usage: $0 [OPTIONS]

Options:
  -l, --list                    List available SSH keys in agent
  -s, --sign <string> [key]     Sign a string with specified key
  -h, --help                    Show this help

Examples:
  $0 --list
  $0 --sign "Hello World" "devrig key"
  $0 --sign "Hello World"
EOF
        ;;
    *)
        echo "Error: Invalid option. Use --help for usage" >&2
        exit 1
        ;;
esac
