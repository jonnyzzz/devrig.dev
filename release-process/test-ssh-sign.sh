#!/bin/bash
set -e

# Test script for ssh-sign.sh
# This script tests signing and verification functionality

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SSH_SIGN="$SCRIPT_DIR/ssh-sign.sh"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local test_name="$1"
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -e "${YELLOW}[TEST $TESTS_RUN]${NC} $test_name"
}

pass_test() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo -e "${GREEN}✓ PASS${NC}"
    echo
}

fail_test() {
    local message="$1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo -e "${RED}✗ FAIL${NC}: $message"
    echo
}

# Create temporary directory for tests
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT

cd "$TEST_DIR"

# Test 1: List keys
run_test "List available SSH keys"
if "$SSH_SIGN" --list > keys.txt 2>&1; then
    if [ -s keys.txt ]; then
        pass_test
    else
        fail_test "No keys found"
    fi
else
    fail_test "Failed to list keys"
fi

# Test 2: Sign a file with full key line
run_test "Sign a file with full key line from --list"
echo "Hello, World!" > test.txt
FULL_KEY_LINE=$("$SSH_SIGN" --list 2>/dev/null | head -1)
if [ -n "$FULL_KEY_LINE" ]; then
    if "$SSH_SIGN" --sign test.txt test.txt.sig "$FULL_KEY_LINE" 2>sign.log; then
        if [ -s test.txt.sig ]; then
            if grep -q "BEGIN SSH SIGNATURE" test.txt.sig; then
                pass_test
            else
                fail_test "Signature file doesn't contain SSH signature"
            fi
        else
            fail_test "Signature file is empty"
        fi
    else
        fail_test "Failed to sign file"
        cat sign.log
    fi
else
    fail_test "No keys available"
fi

# Test 3: Create allowed signers file
run_test "Create allowed signers file from available keys"
"$SSH_SIGN" --list 2>/dev/null | while IFS= read -r key; do
    echo "* $key"
done > allowed_signers

if [ -s allowed_signers ]; then
    pass_test
else
    fail_test "Failed to create allowed signers file"
fi

# Test 4: Verify a valid signature with allowed_signers file
run_test "Verify a valid signature with allowed_signers file"
if [ -f test.txt.sig ] && [ -s allowed_signers ]; then
    if "$SSH_SIGN" --verify test.txt test.txt.sig allowed_signers > verify.log 2>&1; then
        if grep -q "Signature verified successfully" verify.log; then
            pass_test
        else
            fail_test "Verification succeeded but no success message"
        fi
    else
        fail_test "Signature verification failed"
        cat verify.log
    fi
else
    fail_test "Prerequisites missing (signature or allowed_signers)"
fi

# Test 5: Verify with public key string
run_test "Verify signature with public key string (from --list)"
if [ -f test.txt.sig ]; then
    # Get first key from --list output
    FIRST_KEY=$("$SSH_SIGN" --list 2>/dev/null | head -1)
    if [ -n "$FIRST_KEY" ]; then
        if "$SSH_SIGN" --verify test.txt test.txt.sig "$FIRST_KEY" > verify_key.log 2>&1; then
            if grep -q "Signature verified successfully" verify_key.log; then
                pass_test
            else
                fail_test "Verification succeeded but no success message"
            fi
        else
            fail_test "Signature verification with key failed"
            cat verify_key.log
        fi
    else
        fail_test "No keys available for testing"
    fi
else
    fail_test "Prerequisites missing (signature)"
fi

# Test 6: Verify should fail with wrong data
run_test "Verify should fail with tampered data"
echo "Tampered data" > test_tampered.txt
if "$SSH_SIGN" --verify test_tampered.txt test.txt.sig allowed_signers > verify_fail.log 2>&1; then
    fail_test "Verification should have failed for tampered data"
else
    if grep -q "Signature verification failed" verify_fail.log; then
        pass_test
    else
        fail_test "Expected verification failure message not found"
    fi
fi

# Test 7: Verify should fail with invalid signature
run_test "Verify should fail with invalid signature"
echo "INVALID SIGNATURE" > invalid.sig
if "$SSH_SIGN" --verify test.txt invalid.sig allowed_signers > verify_invalid.log 2>&1; then
    fail_test "Verification should have failed for invalid signature"
else
    pass_test
fi

# Test 8: Help command
run_test "Display help message"
if "$SSH_SIGN" --help > help.txt 2>&1; then
    if grep -q "Usage:" help.txt && grep -q "Examples:" help.txt; then
        pass_test
    else
        fail_test "Help message incomplete"
    fi
else
    fail_test "Failed to display help"
fi

# Test 9: Error handling - missing key parameter
run_test "Error handling: Sign without key parameter"
echo "test" > test_nokey.txt
if "$SSH_SIGN" --sign test_nokey.txt output.sig > /dev/null 2>&1; then
    fail_test "Should have failed for missing key parameter"
else
    pass_test
fi

# Test 10: Error handling - invalid key format
run_test "Error handling: Sign with invalid key format (not a full key line)"
echo "test" > test_badkey.txt
if "$SSH_SIGN" --sign test_badkey.txt output.sig "not-a-key" > /dev/null 2>&1; then
    fail_test "Should have failed for invalid key format"
else
    pass_test
fi

# Test 11: Error handling - missing arguments
run_test "Error handling: Verify with missing arguments"
if "$SSH_SIGN" --verify test.txt > /dev/null 2>&1; then
    fail_test "Should have failed for missing arguments"
else
    pass_test
fi

# Print summary
echo "================================"
echo "TEST SUMMARY"
echo "================================"
echo "Total tests run: $TESTS_RUN"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
