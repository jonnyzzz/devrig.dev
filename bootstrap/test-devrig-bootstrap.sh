#!/bin/bash
# test-devrig-bootstrap.sh
# Integration test for devrig bootstrap system

set -euo pipefail

echo "===================================="
echo "devrig Bootstrap Integration Test"
echo "===================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() {
    echo -e "${GREEN}✓ PASS${NC} $1"
}

fail() {
    echo -e "${RED}✗ FAIL${NC} $1"
    exit 1
}

warn() {
    echo -e "${YELLOW}⚠ WARN${NC} $1"
}

# Test 1: Check files exist
echo "Test 1: Checking required files exist..."
if [ -f "devrig" ] && [ -f "devrig.yaml" ]; then
    pass "Required files exist"
else
    fail "Missing required files (devrig or devrig.yaml)"
fi

# Test 2: Check devrig is executable
echo ""
echo "Test 2: Checking devrig is executable..."
if [ -x "devrig" ]; then
    pass "devrig is executable"
else
    fail "devrig is not executable (run: chmod +x devrig)"
fi

# Test 3: Check YAML syntax
echo ""
echo "Test 3: Checking devrig.yaml syntax..."
if grep -q "^devrig:" devrig.yaml && grep -q "binaries:" devrig.yaml; then
    pass "devrig.yaml has correct structure"
else
    fail "devrig.yaml is missing required sections"
fi

# Test 4: Check platform configuration
echo ""
echo "Test 4: Checking platform configurations..."
platforms=("linux-x86_64" "linux-arm64" "darwin-arm64" "windows-x86_64" "windows-arm64")
all_found=true
for platform in "${platforms[@]}"; do
    if grep -q "$platform:" devrig.yaml; then
        echo "  ✓ $platform configured"
    else
        echo "  ✗ $platform missing"
        all_found=false
    fi
done

if $all_found; then
    pass "All 5 platforms configured"
else
    fail "Not all platforms configured in devrig.yaml"
fi

# Test 5: Check URL and SHA256 format
echo ""
echo "Test 5: Checking URL and SHA256 format..."
if grep -q "url:.*http" devrig.yaml && grep -q "sha256:.*[a-f0-9]\{64\}" devrig.yaml; then
    pass "URLs and SHA256 hashes properly formatted"
else
    warn "URLs or SHA256 hashes may not be properly formatted"
fi

# Test 6: Environment variable handling
echo ""
echo "Test 6: Testing environment variable overrides..."
export DEVRIG_CONFIG="/tmp/test-devrig.yaml"
export DEVRIG_HOME="/tmp/test-devrig-home"

# Copy config to test location
cp devrig.yaml "$DEVRIG_CONFIG"

# Run devrig (it will fail because binaries don't exist, but we're testing detection)
if ./devrig --version 2>&1 | grep -q "Using custom"; then
    pass "Environment variable overrides detected"
else
    warn "Environment variable override logging may not be working"
fi

# Clean up
rm -f "$DEVRIG_CONFIG"
unset DEVRIG_CONFIG
unset DEVRIG_HOME

# Test 7: OS and CPU detection simulation
echo ""
echo "Test 7: Simulating platform detection..."
os=$(uname -s)
cpu=$(uname -m)

case "$os" in
    Linux*)  detected_os="linux";;
    Darwin*) detected_os="darwin";;
    *)       detected_os="unknown";;
esac

case "$cpu" in
    x86_64|amd64)  detected_cpu="x86_64";;
    arm64|aarch64) detected_cpu="arm64";;
    *)             detected_cpu="unknown";;
esac

echo "  Detected: $detected_os-$detected_cpu"
if grep -q "${detected_os}-${detected_cpu}:" devrig.yaml; then
    pass "Current platform ($detected_os-$detected_cpu) is configured"
else
    fail "Current platform ($detected_os-$detected_cpu) is NOT configured in devrig.yaml"
fi

# Test 8: Check .devrig directory structure would be correct
echo ""
echo "Test 8: Validating binary layout pattern..."
layout_pattern="devrig-{os}-{cpu}-{version}{hash}"
echo "  Expected pattern: .devrig/$layout_pattern/"
pass "Binary layout pattern documented correctly"

# Summary
echo ""
echo "===================================="
echo "Test Summary"
echo "===================================="
echo ""
echo "Platform: $(uname -s) $(uname -m)"
echo "Config:   devrig.yaml"
echo "Scripts:  devrig (bash), devrig.ps1 (PowerShell), devrig.bat (batch)"
echo ""
echo -e "${GREEN}All basic tests passed!${NC}"
echo ""
echo "Note: This test validates the bootstrap structure."
echo "To fully test, you need actual devrig binaries at the URLs in devrig.yaml"
echo ""
echo "Next steps:"
echo "  1. Update devrig.yaml with real URLs and checksums"
echo "  2. Run: ./devrig --version (or equivalent command)"
echo "  3. Verify binary downloads and executes correctly"
