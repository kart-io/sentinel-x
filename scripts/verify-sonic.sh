#!/usr/bin/env bash

# Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
# Use of this source code is governed by a MIT style
# license that can be found in the LICENSE file.

# Sonic JSON Integration - Verification Script
# This script verifies the sonic integration is working correctly

PROJ_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
source "${PROJ_ROOT_DIR}/scripts/lib/init.sh"

onex::log::info "================================================"
onex::log::info "Sonic JSON Integration Verification"
onex::log::info "================================================"
onex::log::info ""

check_pass() {
    onex::color::green "✓ $1"
}

check_fail() {
    onex::color::red "✗ $1"
    exit 1
}

# 1. Check if json package builds
onex::log::info "1. Building JSON package..."
if go build ./pkg/utils/json 2>&1 > /dev/null; then
    check_pass "JSON package builds successfully"
else
    check_fail "JSON package failed to build"
fi

# 2. Run unit tests
onex::log::info ""
onex::log::info "2. Running unit tests..."
if go test ./pkg/utils/json -v > /tmp/test_output.txt 2>&1; then
    TESTS_PASSED=$(grep -c "PASS:" /tmp/test_output.txt || true)
    check_pass "All unit tests passed ($TESTS_PASSED tests)"
else
    check_fail "Unit tests failed"
fi

# 3. Check HTTP transport integration
onex::log::info ""
onex::log::info "3. Verifying HTTP transport integration..."
if go build ./pkg/infra/server/transport/http 2>&1 > /dev/null; then
    check_pass "HTTP transport builds with sonic integration"
else
    check_fail "HTTP transport failed to build"
fi

# 4. Run HTTP transport tests
onex::log::info ""
onex::log::info "4. Running HTTP transport tests..."
if go test ./pkg/infra/server/transport/http > /dev/null 2>&1; then
    check_pass "HTTP transport tests passed"
else
    check_fail "HTTP transport tests failed"
fi

# 5. Check if sonic is being used
onex::log::info ""
onex::log::info "5. Checking sonic activation..."
mkdir -p cmd/verify_sonic_tmp
cat <<EOF > cmd/verify_sonic_tmp/main.go
package main

import (
	"fmt"
	"github.com/kart-io/sentinel-x/pkg/utils/json"
)

func main() {
	if json.IsUsingSonic() {
		fmt.Println("high-performance sonic")
	} else {
		fmt.Println("standard library fallback")
	}
}
EOF

# Build the verification tool
if go build -o /tmp/verify_sonic_check ./cmd/verify_sonic_tmp; then
    if /tmp/verify_sonic_check | grep -q "high-performance sonic"; then
        check_pass "Sonic is active and working"
    else
        # Check if we are on a compatible architecture
        ARCH=$(go env GOARCH)
        if [[ "$ARCH" == "amd64" ]] || [[ "$ARCH" == "arm64" ]]; then
           onex::log::info "Output: $(/tmp/verify_sonic_check)"
           check_fail "Sonic is not active on supported architecture ($ARCH)"
        else
           onex::log::info "Sonic not active on $ARCH (expected)"
           check_pass "Sonic is not active (fallback on $ARCH)"
        fi
    fi
    rm /tmp/verify_sonic_check
    rm -rf cmd/verify_sonic_tmp
else
    check_fail "Failed to build verification tool"
    rm -rf cmd/verify_sonic_tmp
fi

# 6. Run quick benchmark
onex::log::info ""
onex::log::info "6. Running performance benchmark..."
BENCH_OUTPUT=$(go test -bench=BenchmarkRoundTripAPIResponse -benchtime=500ms -run=^$ ./pkg/utils/json 2>&1)

SONIC_TIME=$(echo "$BENCH_OUTPUT" | grep "Sonic-" | awk '{print $3}')
STDLIB_TIME=$(echo "$BENCH_OUTPUT" | grep "Stdlib-" | awk '{print $3}')

if [ ! -z "$SONIC_TIME" ] && [ ! -z "$STDLIB_TIME" ]; then
    check_pass "Benchmark completed"
    onex::log::info "   Sonic:  $SONIC_TIME"
    onex::log::info "   Stdlib: $STDLIB_TIME"
else
    check_fail "Benchmark failed to run"
fi

# 7. Verify files exist
onex::log::info ""
onex::log::info "7. Verifying all required files..."
FILES=(
    "pkg/utils/json/json.go"
    "pkg/utils/json/json_test.go"
    "pkg/utils/json/benchmark_test.go"
    "pkg/utils/json/README.md"
    "pkg/utils/json/PERFORMANCE.md"
    "docs/SONIC_INTEGRATION.md"
)

for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        check_pass "$file exists"
    else
        check_fail "$file is missing"
    fi
done

onex::log::info ""
onex::log::info "================================================"
onex::color::green "All verification checks passed!"
onex::log::info "================================================"
onex::log::info ""
onex::log::info "The sonic integration is working correctly and ready for deployment."
onex::log::info ""
onex::log::info "Next steps:"
onex::log::info "1. Review documentation in pkg/utils/json/README.md"
onex::log::info "2. Review performance report in pkg/utils/json/PERFORMANCE.md"
onex::log::info "3. Deploy to staging environment"
onex::log::info "4. Monitor metrics for 24-48 hours"
onex::log::info "5. Deploy to production"
onex::log::info ""
