# signal-aws

A lightweight CLI binary that enables CloudFormation-style signaling for Terraform deployments via AWS SQS.

## Overview

`signal-aws` bridges the gap between Terraform's infrastructure provisioning and application readiness by providing a CloudFormation `cfn-signal` equivalent for Terraform. It allows EC2 instances to signal their configuration status back to Terraform through SQS messages, enabling true infrastructure-application synchronization.

### How it fits in the ecosystem

- **Terraform Provider**: Waits for signals via SQS polling (separate component)
- **signal-aws Binary**: Runs on EC2 instances to send readiness signals
- **AWS SQS**: Message transport layer between instances and Terraform

## Quick Start

### Installation
```bash
# Download and install
curl -L https://github.com/terraconstructs/signal-aws/releases/latest/download/signal-aws_Linux_x86_64.tar.gz | tar xz
chmod +x tcsignal-aws
sudo mv tcsignal-aws /usr/local/bin/

# Or build from source
make build
```

### Basic Usage

```bash
# Signal success after running a command
tcsignal-aws --queue-url https://sqs.us-east-1.amazonaws.com/123456789/my-queue \
             --id deployment-123 \
             --exec "./install-app.sh"

# Manual status signaling
tcsignal-aws --queue-url https://sqs.us-east-1.amazonaws.com/123456789/my-queue \
             --id deployment-123 \
             --status SUCCESS
```

### Terraform Integration Example

```hcl
resource "aws_instance" "web" {
  # ... instance configuration ...
  
  user_data = <<-EOD
    #!/bin/bash
    # Download and install signal-aws
    curl -L "https://github.com/terraconstructs/signal-aws/releases/latest/download/signal-aws_Linux_x86_64.tar.gz" | tar xz
    chmod +x tcsignal-aws
    sudo mv tcsignal-aws /usr/local/bin/
    
    # Run application setup and signal completion
    /usr/local/bin/tcsignal-aws \
      --queue-url "${aws_sqs_queue.signals.url}" \
      --id "${local.deployment_id}" \
      --exec "./setup-application.sh"
  EOD
}

resource "tconsaws_signal" "web_ready" {
  queue_url      = aws_sqs_queue.signals.url
  signal_id      = local.deployment_id
  expected_count = length(aws_instance.web)
  timeout        = "10m"
  depends_on     = [aws_instance.web]
}
```

## Features

- **CLI Interface**: Full flag parsing with validation
- **Command Execution**: Wraps user commands and captures exit codes  
- **AWS Integration**: IMDS instance ID fetching + SQS publishing
- **Error Handling**: Proper exit codes (0=success, 1=child failed, 2=publish failed)
- **Testing**: Comprehensive mock-based testing covering all scenarios
- **Retry Logic**: Configurable retries with exponential backoff
- **Structured Logging**: JSON/console format with observability integration
- **Integration Testing**: Local ElasticMQ testing setup

## CLI Reference

```
USAGE:
  tcsignal-aws [flags]

FLAGS:
  -u, --queue-url string     (required) SQS queue URL
  -i, --id string            (required) unique signal ID for the deployment
  -e, --exec string          run this command and signal based on its exit code
  -s, --status string        shortcut: send "SUCCESS" or "FAILURE" without exec
  -n, --instance-id string   override instance ID (default: fetch from IMDS)
  --retries int              transient-error retries (default 3)
  --publish-timeout duration timeout per SendMessage (default 10s)
  --timeout duration         total operation timeout (default 30s)
  --log-format string        log format: json or console (default "console")
  --log-level string         log level: debug, info, warn, or error (default "info")
  --help                     show usage
```

## Local Testing & Development

### Testing Without EC2/IMDS

When developing or testing outside of EC2 environments, use the `--instance-id` flag to bypass IMDS:

```bash
# Test locally without IMDS dependency
tcsignal-aws --queue-url https://sqs.us-east-1.amazonaws.com/123456789/my-queue \
             --id test-signal-123 \
             --status SUCCESS \
             --instance-id i-local-test-12345

# Test with command execution
tcsignal-aws --queue-url https://sqs.us-east-1.amazonaws.com/123456789/my-queue \
             --id test-signal-456 \
             --exec "./my-test-script.sh" \
             --instance-id i-local-test-67890
```

### Integration Testing

Run the full integration test suite with local ElasticMQ and EC2 metadata mock:

```bash
# Start local test environment
make integration-up

# Run integration tests
make integration-test

# Stop test environment
make integration-down

# Or run the complete workflow in one command
make integration-test  # (automatically starts and stops environment)
```

The integration environment provides:
- **ElasticMQ**: Local SQS-compatible message queue
- **EC2 Metadata Mock**: Simulates AWS IMDS responses
- **End-to-end testing**: Validates complete binary workflow

## Tech Stack

- **Language**: Go (single static binary, no dependencies)
- **AWS SDK**: AWS SDK for Go v2
- **Testing**: Native Go testing with mocks
- **Build**: Standard Go toolchain + Makefile

## Development

```bash
# Build
make build

# Test
make test

# Test with coverage
make test-coverage

# Run all checks
make check

# Test with fixtures
make test-fixtures
```

## Architecture

### Message Format
```json
{
  "MessageAttributes": {
    "signal_id": {"DataType": "String", "StringValue": "deployment-123"},
    "instance_id": {"DataType": "String", "StringValue": "i-0abc123def456"},
    "status": {"DataType": "String", "StringValue": "SUCCESS"}
  }
}
```

### Exit Codes
- `0`: Success (command succeeded and signal sent)
- `1`: Command failed (signal sent with FAILURE status)
- `2`: Signal publishing failed

### AWS Permissions Required
The EC2 instance needs:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["sqs:SendMessage"],
      "Resource": "arn:aws:sqs:*:*:your-signal-queue"
    }
  ]
}
```

## Project Status

**Current**: Phase 1 Complete ✅  
**Next**: Phase 2 implementation with retry logic and structured logging

This is part of a larger Terraform provider ecosystem that enables CloudFormation-style resource signaling for Terraform deployments.