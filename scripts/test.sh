#!/bin/bash

# Test Script for McMocknald Order Kiosk System
# Runs scenario tests with the "scenario" build tag
# Usage: ./test.sh [ci]
#   - Default mode: Runs original 3-minute tests (TestSmallLoad, TestLargeLoad)
#   - CI mode: ./test.sh ci - Runs 1-minute CI tests (TestSmallLoadCI, TestLargeLoadCI)

set -e  # Exit on error

# Parse command-line arguments
CI_MODE=false
if [ "$1" == "ci" ]; then
    CI_MODE=true
fi

echo "============================================================"
echo "McMocknald Order Kiosk - Scenario Test Script"
echo "============================================================"
echo ""

# Get the project root (parent of scripts directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Project Root: $PROJECT_ROOT"
echo "Go Version: $(go version)"
echo "Test Mode: $([ "$CI_MODE" = true ] && echo "CI (1-minute tests)" || echo "Standard (3-minute tests)")"
echo ""

# Navigate to project root
cd "$PROJECT_ROOT"

if [ "$CI_MODE" = true ]; then
    # CI Mode: Run 1-minute tests for faster CI/CD feedback
    echo "============================================================"
    echo "Running CI Scenario Tests (Optimized for CI/CD)"
    echo "============================================================"
    echo ""
    echo "Tests will run with the following parameters:"
    echo ""
    echo "Small Load CI Test:"
    echo "  - 100 Regular Customers"
    echo "  - 50 VIP Customers"
    echo "  - 25 Cook Bots"
    echo "  - Duration: 1 minute"
    echo "  - Report Interval: 10 seconds"
    echo ""
    echo "Large Load CI Test:"
    echo "  - 10,000 Regular Customers"
    echo "  - 5,000 VIP Customers"
    echo "  - 1,250 Cook Bots"
    echo "  - Duration: 1 minute"
    echo "  - Report Interval: 10 seconds"
    echo ""
    echo "============================================================"
    echo ""

    # Run the CI scenario tests with proper flags
    # -tags scenario: Include files with //go:build scenario tag
    # -v: Verbose output to see test progress
    # -timeout: Set timeout for long-running tests (5m should be sufficient for CI tests)
    # ./test/scenario: Run only tests in the scenario directory

    echo "Starting Small Load CI Test..."
    echo "------------------------------------------------------------"
    go test -tags scenario -v -timeout 5m ./test/scenario -run TestSmallLoadCI

    echo ""
    echo "============================================================"
    echo ""
    echo "Starting Large Load CI Test..."
    echo "------------------------------------------------------------"
    go test -tags scenario -v -timeout 5m ./test/scenario -run TestLargeLoadCI

    echo ""
    echo "============================================================"
    echo "All CI Scenario Tests Completed Successfully!"
    echo "============================================================"
    echo ""

    # Summary
    echo "Test Summary (CI Mode):"
    echo "  - Small Load CI Test: Completed"
    echo "  - Large Load CI Test: Completed"
    echo ""
    echo "Check the output above for detailed statistics including:"
    echo "  - Completion rates at 10-second intervals"
    echo "  - Final completion percentages"
    echo "  - Queue sizes throughout the test"
    echo "  - Order processing performance"
    echo ""
else
    # Standard Mode: Run full 3-minute tests
    echo "============================================================"
    echo "Running Scenario Tests (Standard Mode)"
    echo "============================================================"
    echo ""
    echo "Tests will run with the following parameters:"
    echo ""
    echo "Small Load Test:"
    echo "  - 100 Regular Customers"
    echo "  - 50 VIP Customers"
    echo "  - 25 Cook Bots"
    echo "  - Duration: 3 minutes"
    echo "  - Report Interval: 20 seconds"
    echo ""
    echo "Large Load Test:"
    echo "  - 10,000 Regular Customers"
    echo "  - 5,000 VIP Customers"
    echo "  - 1,250 Cook Bots"
    echo "  - Duration: 3 minutes"
    echo "  - Report Interval: 20 seconds"
    echo ""
    echo "============================================================"
    echo ""

    # Run the scenario tests with proper flags
    # -tags scenario: Include files with //go:build scenario tag
    # -v: Verbose output to see test progress
    # -timeout: Set timeout for long-running tests (default is 10m, we set 15m for safety)
    # ./test/scenario: Run only tests in the scenario directory

    echo "Starting Small Load Test..."
    echo "------------------------------------------------------------"
    go test -tags scenario -v -timeout 15m ./test/scenario -run TestSmallLoad

    echo ""
    echo "============================================================"
    echo ""
    echo "Starting Large Load Test..."
    echo "------------------------------------------------------------"
    go test -tags scenario -v -timeout 15m ./test/scenario -run TestLargeLoad

    echo ""
    echo "============================================================"
    echo "All Scenario Tests Completed Successfully!"
    echo "============================================================"
    echo ""

    # Summary
    echo "Test Summary:"
    echo "  - Small Load Test: Completed"
    echo "  - Large Load Test: Completed"
    echo ""
    echo "Check the output above for detailed statistics including:"
    echo "  - Completion rates at 20-second intervals"
    echo "  - Final completion percentages"
    echo "  - Queue sizes throughout the test"
    echo "  - Order processing performance"
    echo ""
fi

echo "============================================================"
