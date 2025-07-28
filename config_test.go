package signal

import (
	"flag"
	"os"
	"testing"
	"time"
)

func TestParseConfig_Success(t *testing.T) {
	// Reset flag set for testing
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	os.Args = []string{
		"tcsignal-aws",
		"--queue-url", "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
		"--id", "test-signal-123",
		"--exec", "echo hello",
		"--log-level", "debug",
		"--log-format", "json",
		"--retries", "5",
		"--publish-timeout", "15s",
		"--timeout", "60s",
	}

	cfg, err := ParseConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.QueueURL != "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue" {
		t.Errorf("Expected QueueURL to be set correctly, got: %s", cfg.QueueURL)
	}

	if cfg.ID != "test-signal-123" {
		t.Errorf("Expected ID to be set correctly, got: %s", cfg.ID)
	}

	if cfg.Exec != "echo hello" {
		t.Errorf("Expected Exec to be set correctly, got: %s", cfg.Exec)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel to be debug, got: %s", cfg.LogLevel)
	}

	if cfg.LogFormat != "json" {
		t.Errorf("Expected LogFormat to be json, got: %s", cfg.LogFormat)
	}

	if cfg.Retries != 5 {
		t.Errorf("Expected Retries to be 5, got: %d", cfg.Retries)
	}

	if cfg.PublishTimeout != 15*time.Second {
		t.Errorf("Expected PublishTimeout to be 15s, got: %v", cfg.PublishTimeout)
	}

	if cfg.Timeout != 60*time.Second {
		t.Errorf("Expected Timeout to be 60s, got: %v", cfg.Timeout)
	}
}

func TestParseConfig_MissingQueueURL(t *testing.T) {
	// Reset flag set for testing
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Args = []string{
		"tcsignal-aws",
		"--id", "test-signal-123",
		"--exec", "echo hello",
	}

	_, err := ParseConfig()
	if err == nil {
		t.Fatal("Expected error for missing queue-url, got nil")
	}

	if err.Error() != "--queue-url is required" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestParseConfig_MissingID(t *testing.T) {
	// Reset flag set for testing
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Args = []string{
		"tcsignal-aws",
		"--queue-url", "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
		"--exec", "echo hello",
	}

	_, err := ParseConfig()
	if err == nil {
		t.Fatal("Expected error for missing id, got nil")
	}

	if err.Error() != "--id is required" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestParseConfig_MissingExecAndStatus(t *testing.T) {
	// Reset flag set for testing
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Args = []string{
		"tcsignal-aws",
		"--queue-url", "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
		"--id", "test-signal-123",
	}

	_, err := ParseConfig()
	if err == nil {
		t.Fatal("Expected error for missing exec and status, got nil")
	}

	if err.Error() != "either --exec or --status must be provided" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestParseConfig_InvalidStatus(t *testing.T) {
	// Reset flag set for testing
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Args = []string{
		"tcsignal-aws",
		"--queue-url", "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
		"--id", "test-signal-123",
		"--status", "INVALID",
	}

	_, err := ParseConfig()
	if err == nil {
		t.Fatal("Expected error for invalid status, got nil")
	}

	if err.Error() != "--status must be either SUCCESS or FAILURE" {
		t.Errorf("Expected specific error message, got: %s", err.Error())
	}
}

func TestParseConfig_ValidStatus(t *testing.T) {
	testCases := []string{"SUCCESS", "FAILURE"}

	for _, status := range testCases {
		t.Run("Status_"+status, func(t *testing.T) {
			// Reset flag set for testing
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			os.Args = []string{
				"tcsignal-aws",
				"--queue-url", "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
				"--id", "test-signal-123",
				"--status", status,
			}

			cfg, err := ParseConfig()
			if err != nil {
				t.Fatalf("Expected no error for valid status %s, got: %v", status, err)
			}

			if cfg.Status != status {
				t.Errorf("Expected Status to be %s, got: %s", status, cfg.Status)
			}
		})
	}
}

func TestParseConfig_ShortFlags(t *testing.T) {
	// Reset flag set for testing
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Args = []string{
		"tcsignal-aws",
		"-u", "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
		"-i", "test-signal-123",
		"-e", "echo hello",
	}

	cfg, err := ParseConfig()
	if err != nil {
		t.Fatalf("Expected no error with short flags, got: %v", err)
	}

	if cfg.QueueURL != "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue" {
		t.Errorf("Expected QueueURL to be set correctly with short flag, got: %s", cfg.QueueURL)
	}

	if cfg.ID != "test-signal-123" {
		t.Errorf("Expected ID to be set correctly with short flag, got: %s", cfg.ID)
	}

	if cfg.Exec != "echo hello" {
		t.Errorf("Expected Exec to be set correctly with short flag, got: %s", cfg.Exec)
	}

}

func TestParseConfig_Defaults(t *testing.T) {
	// Reset flag set for testing
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Args = []string{
		"tcsignal-aws",
		"--queue-url", "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
		"--id", "test-signal-123",
		"--exec", "echo hello",
	}

	cfg, err := ParseConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check defaults
	if cfg.Retries != 3 {
		t.Errorf("Expected default Retries to be 3, got: %d", cfg.Retries)
	}

	if cfg.PublishTimeout != 10*time.Second {
		t.Errorf("Expected default PublishTimeout to be 10s, got: %v", cfg.PublishTimeout)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected default Timeout to be 30s, got: %v", cfg.Timeout)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected default LogLevel to be info, got: %s", cfg.LogLevel)
	}

	if cfg.LogFormat != "console" {
		t.Errorf("Expected default LogFormat to be console, got: %s", cfg.LogFormat)
	}
}
