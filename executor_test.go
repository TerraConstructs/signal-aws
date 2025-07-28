package signal

import (
	"fmt"
	"testing"
)

func TestDefaultExecutor_Success(t *testing.T) {
	executor := NewDefaultExecutor(false)

	// Test with success.sh fixture
	exitCode, err := executor.Run("./test/fixtures/success.sh")
	if err != nil {
		t.Fatalf("Expected no error for success.sh, got: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for success.sh, got: %d", exitCode)
	}
}

func TestDefaultExecutor_Failure(t *testing.T) {
	executor := NewDefaultExecutor(false)

	// Test with fail.sh fixture
	exitCode, err := executor.Run("./test/fixtures/fail.sh")
	if err != nil {
		t.Fatalf("Expected no error executing fail.sh, got: %v", err)
	}

	if exitCode != 2 {
		t.Errorf("Expected exit code 2 for fail.sh, got: %d", exitCode)
	}
}

func TestDefaultExecutor_InvalidCommand(t *testing.T) {
	executor := NewDefaultExecutor(false)

	// Test with non-existent command
	exitCode, err := executor.Run("this-command-does-not-exist-12345")

	// sh -c will run but the command inside will fail with exit code 127 (command not found)
	if err != nil {
		t.Fatalf("Expected no error from sh -c execution, got: %v", err)
	}

	// Command not found typically returns exit code 127
	if exitCode != 127 {
		t.Errorf("Expected exit code 127 for command not found, got: %d", exitCode)
	}
}

func TestDefaultExecutor_Verbose(t *testing.T) {
	// Test that verbose mode doesn't break execution
	executor := NewDefaultExecutor(true)

	exitCode, err := executor.Run("echo 'verbose test'")
	if err != nil {
		t.Fatalf("Expected no error with verbose mode, got: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("Expected exit code 0 with verbose mode, got: %d", exitCode)
	}
}

func TestDefaultExecutor_ExitCodeHandling(t *testing.T) {
	executor := NewDefaultExecutor(false)

	testCases := []struct {
		name         string
		command      string
		expectedCode int
	}{
		{"exit_0", "exit 0", 0},
		{"exit_1", "exit 1", 1},
		{"exit_5", "exit 5", 5},
		{"exit_127", "exit 127", 127},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exitCode, err := executor.Run(tc.command)
			if err != nil {
				t.Fatalf("Expected no error for '%s', got: %v", tc.command, err)
			}

			if exitCode != tc.expectedCode {
				t.Errorf("Expected exit code %d for '%s', got: %d",
					tc.expectedCode, tc.command, exitCode)
			}
		})
	}
}

func TestMockExecutor_Basic(t *testing.T) {
	mock := NewMockExecutor()

	// Test default behavior (should return 0, nil)
	exitCode, err := mock.Run("test-command")
	if err != nil {
		t.Errorf("Expected no error from mock, got: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected default exit code 0, got: %d", exitCode)
	}

	// Verify call was recorded
	calls := mock.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 call recorded, got: %d", len(calls))
	}
	if calls[0] != "test-command" {
		t.Errorf("Expected call to be 'test-command', got: %s", calls[0])
	}
}

func TestMockExecutor_SetExitCode(t *testing.T) {
	mock := NewMockExecutor()
	mock.SetExitCode(42)

	exitCode, err := mock.Run("test-command")
	if err != nil {
		t.Errorf("Expected no error from mock, got: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("Expected exit code 42, got: %d", exitCode)
	}
}

func TestMockExecutor_SetError(t *testing.T) {
	mock := NewMockExecutor()
	expectedErr := fmt.Errorf("mock error")
	mock.SetError(expectedErr)

	exitCode, err := mock.Run("test-command")
	if err != expectedErr {
		t.Errorf("Expected mock error, got: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0 (default), got: %d", exitCode)
	}
}

func TestMockExecutor_CustomResults(t *testing.T) {
	mock := NewMockExecutor()

	// Set custom result for specific command
	mock.SetResultForCommand("special-command", 5, nil)
	mock.SetExitCode(1) // This should be overridden for special-command

	// Test special command
	exitCode, err := mock.Run("special-command")
	if err != nil {
		t.Errorf("Expected no error for special command, got: %v", err)
	}
	if exitCode != 5 {
		t.Errorf("Expected exit code 5 for special command, got: %d", exitCode)
	}

	// Test regular command (should use default)
	exitCode, err = mock.Run("regular-command")
	if err != nil {
		t.Errorf("Expected no error for regular command, got: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for regular command, got: %d", exitCode)
	}

	// Verify call count
	if mock.CallCount() != 2 {
		t.Errorf("Expected 2 calls, got: %d", mock.CallCount())
	}
}

func TestMockExecutor_ThreadSafety(t *testing.T) {
	mock := NewMockExecutor()

	// Run multiple goroutines to test thread safety
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			mock.Run(fmt.Sprintf("command-%d", id))
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	if mock.CallCount() != 10 {
		t.Errorf("Expected 10 calls after concurrent execution, got: %d", mock.CallCount())
	}
}
