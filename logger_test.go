package signal

import (
	"testing"
)

// TestNewLogger_InvalidFormat tests our error handling for invalid log formats
func TestNewLogger_InvalidFormat(t *testing.T) {
	// Test with invalid format
	logger, err := NewLogger("xml", "info")
	if err == nil {
		t.Error("Expected error for invalid format, got nil")
	}
	if logger != nil {
		t.Error("Expected nil logger for invalid format")
	}

	// Test error message contains useful information
	expectedError := "invalid log format: xml (must be 'json' or 'console')"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// TestNewLogger_InvalidLevel tests our error handling for invalid log levels
func TestNewLogger_InvalidLevel(t *testing.T) {
	// Invalid levels should default to info level and not error
	logger, err := NewLogger("console", "invalid-level")
	if err != nil {
		t.Errorf("Expected no error for invalid level (should default), got: %v", err)
	}
	if logger == nil {
		t.Error("Expected logger instance even with invalid level, got nil")
	}
}

// TestNewLogger_ValidFormats tests that valid formats don't error
func TestNewLogger_ValidFormats(t *testing.T) {
	testCases := []struct {
		format string
		level  string
	}{
		{"json", "info"},
		{"console", "debug"},
		{"json", "error"},
		{"console", "warn"},
	}

	for _, tc := range testCases {
		t.Run(tc.format+"_"+tc.level, func(t *testing.T) {
			logger, err := NewLogger(tc.format, tc.level)
			if err != nil {
				t.Errorf("Expected no error for valid format/level %s/%s, got: %v", tc.format, tc.level, err)
			}
			if logger == nil {
				t.Errorf("Expected logger instance for valid format/level %s/%s, got nil", tc.format, tc.level)
			}
		})
	}
}

// Note: We don't test zap's internal functionality (JSON output format, sync behavior, etc.)
// as that's zap's responsibility. Our config integration is tested in config_test.go.
// This approach reduces maintenance burden and focuses tests on our business logic.
