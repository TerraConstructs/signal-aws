# tcons-signal

A lightweight CLI binary that enables CloudFormation-style signaling for Terraform deployments via AWS SQS.

## Overview

`tcons-signal` bridges the gap between Terraform's infrastructure provisioning and application readiness by providing a CloudFormation `cfn-signal` equivalent for Terraform. It allows EC2 instances to signal their configuration status back to Terraform through SQS messages, enabling true infrastructure-application synchronization.

### How it fits in the ecosystem

- **Terraform Provider**: Waits for signals via SQS polling (separate component)
- **tcons-signal Binary**: Runs on EC2 instances to send readiness signals
- **AWS SQS**: Message transport layer between instances and Terraform

## Quick Start

### Installation
```bash
# Download and install
curl -L https://github.com/your-org/tcons-signal/releases/latest/download/tcons-signal-linux-amd64 -o /usr/local/bin/tcons-signal
chmod +x /usr/local/bin/tcons-signal

# Or build from source
make build
```

### Basic Usage

```bash
# Signal success after running a command
tcons-signal --queue-url https://sqs.us-east-1.amazonaws.com/123456789/my-queue \
             --id deployment-123 \
             --exec "./install-app.sh"

# Manual status signaling
tcons-signal --queue-url https://sqs.us-east-1.amazonaws.com/123456789/my-queue \
             --id deployment-123 \
             --status SUCCESS
```

### Terraform Integration Example

```hcl
resource "aws_instance" "web" {
  # ... instance configuration ...
  
  user_data = <<-EOD
    #!/bin/bash
    # Download and install tcons-signal
    curl -L https://releases.example.com/tcons-signal -o /usr/local/bin/tcons-signal
    chmod +x /usr/local/bin/tcons-signal
    
    # Run application setup and signal completion
    /usr/local/bin/tcons-signal \
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

### ✅ Phase 1 (Current)
- **CLI Interface**: Full flag parsing with validation
- **Command Execution**: Wraps user commands and captures exit codes  
- **AWS Integration**: IMDS instance ID fetching + SQS publishing
- **Error Handling**: Proper exit codes (0=success, 1=child failed, 2=publish failed)
- **Testing**: Comprehensive mock-based testing covering all scenarios

### 🔄 Phase 2 (Planned)
- **Retry Logic**: Configurable retries with exponential backoff
- **Structured Logging**: JSON/text format with observability integration
- **Integration Testing**: Local ElasticMQ testing setup
- **Enhanced Timeouts**: Per-operation and overall timeout controls

## CLI Reference

```
USAGE:
  tcons-signal [flags]

FLAGS:
  -u, --queue-url string     (required) SQS queue URL
  -i, --id string            (required) unique signal ID for the deployment
  -e, --exec string          run this command and signal based on its exit code
  -s, --status string        shortcut: send "SUCCESS" or "FAILURE" without exec
  -v, --verbose bool         basic log verbosity
  --retries int              transient-error retries (default 3)
  --publish-timeout duration timeout per SendMessage (default 10s)
  --timeout duration         total operation timeout (default 30s)
  --help                     show usage
```

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