#!/bin/bash

# Script to run tunnel tests

# Set up colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Running TunnelBroker Tests${NC}"
echo "========================================"

# Run unit tests
echo -e "${YELLOW}Running Unit Tests${NC}"
go test -v ./internal/tunnels/...

# Check if unit tests passed
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Unit Tests Passed!${NC}"
else
    echo -e "${RED}Unit Tests Failed!${NC}"
    exit 1
fi

# Run tests with coverage
echo -e "\n${YELLOW}Running Tests with Coverage${NC}"
go test -coverprofile=coverage.out ./internal/tunnels/...

# Display coverage
go tool cover -func=coverage.out

# Generate HTML coverage report
echo -e "\n${YELLOW}Generating HTML Coverage Report${NC}"
go tool cover -html=coverage.out -o coverage.html
echo -e "${GREEN}Coverage report generated: coverage.html${NC}"

# Run integration tests if requested
if [ "$1" == "--integration" ]; then
    echo -e "\n${YELLOW}Running Integration Tests${NC}"
    echo "Make sure your test database is running!"
    
    # Set environment variables for integration tests
    export INTEGRATION_TEST=true
    export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/tunnelbroker_test"
    
    # Run integration tests
    go test -v -tags=integration ./internal/tunnels/...
    
    # Check if integration tests passed
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Integration Tests Passed!${NC}"
    else
        echo -e "${RED}Integration Tests Failed!${NC}"
        exit 1
    fi
fi

echo -e "\n${GREEN}All tests completed successfully!${NC}"
