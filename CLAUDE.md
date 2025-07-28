# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`tcons-signal` is a Go CLI binary that enables CloudFormation-style signaling for Terraform deployments via AWS SQS. It acts as a bridge between EC2 instance initialization and Terraform orchestration, allowing instances to signal their readiness status back to Terraform through SQS messages.

## Development Commands

### Core Development Tasks
```bash
# Build the binary
make build

# Run all unit tests
make test

# Run tests with coverage report (generates coverage.html)
make test-coverage

# Run all quality checks (fmt, vet, lint, test)
make check

# Test binary with fixture scripts (expects failures on non-EC2 environments)
make test-fixtures

# Show binary usage
make usage
```

### Integration Testing
```bash
# Start integration test environment (ElasticMQ + EC2 metadata mock)
make integration-up

# Stop integration test environment
make integration-down

# Run complete integration test suite (includes build, setup, test, cleanup)
make integration-test

# Manual integration test with environment variables
AWS_EC2_METADATA_SERVICE_ENDPOINT=http://localhost:1338 \
AWS_ENDPOINT_URL_SQS=http://localhost:9324 \
AWS_REGION=us-east-1 \
go test -v ./cmd -tags=integration
```

### Running Specific Tests
```bash
# Run tests for a specific package
go test -v ./cmd

# Run a specific test function
go test -v ./cmd -run TestRun_ExecSuccess

# Run tests with verbose output and coverage for specific package
go test -v -coverprofile=coverage.out ./publisher.go ./publisher_test.go
```

## High-Level Architecture

### Core Application Flow
1. **Configuration Parsing** (`config.go`) - CLI flags with validation
2. **Command Execution** (`executor.go`) - Wraps user commands and captures exit codes
3. **Instance Metadata** (`imds.go`) - Fetches EC2 instance ID from AWS IMDS
4. **Signal Publishing** (`publisher.go`, `sqs_publisher.go`) - Sends status to SQS
5. **Main Orchestration** (`cmd/main.go`) - Wires components with dependency injection

### Interface-Based Design
All major components use interfaces for clean separation and testability:

- **`Executor`** - Command execution abstraction
- **`Publisher`** - Message publishing abstraction  
- **`IMDSClient`** - EC2 metadata service abstraction

This enables comprehensive mock-based testing without AWS dependencies.

### Exit Code Strategy
- `0`: Success (command succeeded and signal sent)
- `1`: Command failed (signal sent with FAILURE status)
- `2`: Signal publishing failed (infrastructure error)

### SQS Message Format
Messages use empty body with attributes:
```
MessageAttributes:
  signal_id: unique deployment identifier
  instance_id: EC2 instance ID from IMDS
  status: "SUCCESS" or "FAILURE"
```

## Testing Architecture

### Unit Testing with Mocks (`mocks.go`)
Comprehensive mock implementations for isolated testing:
- **MockExecutor**: Simulates command execution with configurable exit codes and errors
- **MockPublisher**: Records SQS calls, simulates failures/retries, supports retry configuration
- **MockIMDSClient**: Provides fake instance IDs, simulates IMDS errors

**Key Features:**
- Thread-safe implementations with proper synchronization
- Configurable failure scenarios for testing edge cases
- Call count tracking and input recording for assertions
- Support for retry logic testing

### Integration Testing Environment
Complete local AWS simulation for end-to-end testing:

**Docker Services:**
- **ElasticMQ** (`softwaremill/elasticmq:1.6.14`): SQS-compatible message queue
- **EC2 Metadata Mock** (`public.ecr.aws/aws-ec2/amazon-ec2-metadata-mock:v1.13.0`): IMDS simulation

**Environment Configuration:**
```bash
AWS_EC2_METADATA_SERVICE_ENDPOINT=http://localhost:1338  # IMDS mock endpoint
AWS_ENDPOINT_URL_SQS=http://localhost:9324               # ElasticMQ endpoint  
AWS_REGION=us-east-1                                     # Required region
AWS_ACCESS_KEY_ID=test                                   # Dummy credentials
AWS_SECRET_ACCESS_KEY=test                               # Dummy credentials
```

**Integration Test Helpers:**
- `test/helpers.go`: CLI wrapper for integration commands (`make integration-up/down/test`)
- `test/integration/helpers.go`: Shared package with common integration functions:
  - `StartEnvironment()`: Starts ElasticMQ and EC2 metadata mock with health checking
  - `StopEnvironment()`: Stops and cleans up services
  - `RunFullTest()`: Complete test suite orchestration
  - `WaitForService()`, `IsElasticMQHealthy()`, `IsEC2MockHealthy()`: Health check utilities

### Test Fixtures (`test/fixtures/`)
- `success.sh`: Returns exit 0 for success scenarios
- `fail.sh`: Returns exit 2 for failure scenarios
- Used by both unit tests and integration tests

### Test Coverage Levels

**Unit Tests:** Mock-based testing covering:
1. ✅ Exec success (MockPublisher + exit 0)
2. ✅ Explicit failure status (`--status FAILURE`)
3. ✅ Exec failure (fail.sh exit 2)
4. ✅ Retry configuration passing (AWS SDK handles actual retries)
5. ✅ Publish timeout simulation
6. ✅ Missing required flags validation
7. ✅ Invalid command execution

**Integration Tests:** End-to-end testing covering:
1. ✅ SQS message format validation against ElasticMQ
2. ✅ AWS SDK endpoint override functionality
3. ✅ Complete binary workflow (config → exec → IMDS → SQS)
4. ✅ Cross-platform Docker environment
5. ⚠️ Retry logic against real SQS responses (minor ElasticMQ compatibility issue)

### Testing Strategy Notes
- **Unit tests** validate logic and error handling using mocks
- **Integration tests** validate AWS SDK integration and message formats
- **No AWS credentials required** for any testing
- **Cross-platform support** with ARM64 Docker platform specification
- **Automated service orchestration** via shared integration helper package

## Project Structure

```
/                           # Root package: interfaces and core types
├── cmd/                   # Main binary entry point
│   ├── main.go           # Binary orchestration and dependency injection
│   ├── main_test.go      # Unit tests for main application logic
│   └── integration_test.go # End-to-end integration tests (build tag: integration)
├── test/                  # Integration testing infrastructure
│   ├── fixtures/         # Test scripts (success.sh, fail.sh)
│   ├── elasticmq.conf    # ElasticMQ queue configuration
│   ├── helpers.go        # Integration environment helper commands
│   └── integration/      # Shared integration helper functions
│       └── helpers.go    # Common integration test utilities
├── docker-compose.yml     # ElasticMQ + EC2 metadata mock services
├── go.mod                 # Go module with AWS SDK v2 dependencies
├── Makefile              # Development automation
├── PHASE2_PLAN.md        # Implementation roadmap and status
├── CLAUDE.md             # This file - development guidance
└── README.md             # User documentation
```

## Key Implementation Notes

### AWS Integration
- Uses AWS SDK for Go v2 for SQS and IMDS services
- Leverages built-in retry and backoff mechanisms
- Proper credential chain handling (instance profiles)

### Timeout Handling
- Overall operation timeout via `--timeout` flag
- Per-SQS-call timeout via `--publish-timeout` flag
- Context-based cancellation throughout the application

### Phase 1 vs Phase 2
Current implementation is Phase 1: core functionality with mock-based testing.
Phase 2 (planned): retry logic implementation, structured logging, ElasticMQ integration testing.

### Development Notes
- All external dependencies are abstracted behind interfaces
- Thread-safe mocks with proper synchronization
- `make test-fixtures` will show IMDS timeout errors when run outside EC2 (this is expected)
- Coverage target is 78%+ (currently achieved)