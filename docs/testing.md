# TunnelBroker Testing Guide

This document provides instructions for running and maintaining the test suite for the TunnelBroker application.

## Test Structure

The test suite is organized into the following components:

1. **Unit Tests**: Test individual functions and components in isolation
2. **Integration Tests**: Test the interaction between components and with the database
3. **Coverage Reports**: Measure and visualize test coverage

## Test Files

The test files are organized as follows:

- `service_test.go`: Tests for service layer functions (CreateTunnelService, DeleteTunnel, etc.)
- `repository_test.go`: Tests for database operations (InsertTunnel, GetTunnelByID, etc.)
- `handler_test.go`: Tests for HTTP handlers (CreateTunnelHandler, GetTunnelHandler, etc.)
- `model_test.go`: Tests for data models and validation functions
- `integration_test.go`: End-to-end tests for the complete tunnel lifecycle

## Running Tests

### Running Unit Tests

To run all unit tests:

```bash
go test -v ./internal/tunnels/...
```

To run a specific test:

```bash
go test -v -run TestCreateTunnelService ./internal/tunnels/...
```

### Running Tests with Coverage

To run tests and generate a coverage report:

```bash
go test -coverprofile=coverage.out ./internal/tunnels/...
go tool cover -html=coverage.out
```

### Running Integration Tests

Integration tests require a test database. To run integration tests:

```bash
export INTEGRATION_TEST=true
export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/tunnelbroker_test"
go test -v -tags=integration ./internal/tunnels/...
```

### Using the Test Script

A convenience script is provided to run all tests:

```bash
# Run unit tests only
./scripts/run_tests.sh

# Run unit tests and integration tests
./scripts/run_tests.sh --integration
```

## Test Cases

### Tunnel Creation Tests

- Creating the first tunnel for a user
- Creating the second tunnel for a user
- Attempting to create a third tunnel (should fail)
- Creating tunnels with different types (SIT, GRE)
- Validating generated prefixes
- Checking command generation

### Tunnel Deletion Tests

- Deleting an existing tunnel
- Attempting to delete a non-existent tunnel
- Verifying resources are properly cleaned up

### Prefix Tests

- Validating prefix uniqueness
- Verifying prefix format
- Testing third prefix generation from dedicated /48

### Error Handling Tests

- Invalid input validation
- Database errors
- Authorization errors

## Mocking

The tests use mocking to isolate components:

- Database operations are mocked to avoid actual database access in unit tests
- Service functions are mocked in handler tests
- Integration tests use a real database but in a controlled environment

## Test Coverage

The test suite aims for >80% code coverage. Coverage reports are generated as HTML files for easy visualization.

## Maintaining Tests

When making changes to the codebase:

1. Update existing tests to reflect the changes
2. Add new tests for new functionality
3. Run the full test suite to ensure nothing breaks
4. Check the coverage report to identify untested code

## Troubleshooting

If tests fail:

1. Check the error messages for specific failures
2. Verify that the test database is running (for integration tests)
3. Ensure that the test environment variables are set correctly
4. Check for recent code changes that might affect the tests
