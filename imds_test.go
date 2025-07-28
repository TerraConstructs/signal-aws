package signal

import (
	"context"
	"fmt"
	"testing"
)

func TestMockIMDSClient_Basic(t *testing.T) {
	mock := NewMockIMDSClient()

	// Test default behavior (should return default fake instance ID)
	instanceID, err := mock.GetInstanceID(context.Background())
	if err != nil {
		t.Errorf("Expected no error from mock IMDS client, got: %v", err)
	}

	expectedID := "i-1234567890abcdef0"
	if instanceID != expectedID {
		t.Errorf("Expected default instance ID %s, got: %s", expectedID, instanceID)
	}

	// Verify call count
	if mock.CallCount() != 1 {
		t.Errorf("Expected call count 1, got: %d", mock.CallCount())
	}
}

func TestMockIMDSClient_SetInstanceID(t *testing.T) {
	mock := NewMockIMDSClient()
	customID := "i-abcdef1234567890"
	mock.SetInstanceID(customID)

	instanceID, err := mock.GetInstanceID(context.Background())
	if err != nil {
		t.Errorf("Expected no error from mock IMDS client, got: %v", err)
	}

	if instanceID != customID {
		t.Errorf("Expected custom instance ID %s, got: %s", customID, instanceID)
	}
}

func TestMockIMDSClient_SetError(t *testing.T) {
	mock := NewMockIMDSClient()
	expectedErr := fmt.Errorf("mock IMDS error")
	mock.SetError(expectedErr)

	instanceID, err := mock.GetInstanceID(context.Background())
	if err != expectedErr {
		t.Errorf("Expected mock error, got: %v", err)
	}

	// Instance ID should still be returned even on error (depends on implementation)
	// In this case, we return both the ID and error
	expectedID := "i-1234567890abcdef0"
	if instanceID != expectedID {
		t.Errorf("Expected instance ID %s even with error, got: %s", expectedID, instanceID)
	}

	// Call should still be counted
	if mock.CallCount() != 1 {
		t.Errorf("Expected call count 1 even on error, got: %d", mock.CallCount())
	}
}

func TestMockIMDSClient_MultipleCalls(t *testing.T) {
	mock := NewMockIMDSClient()

	// Make multiple calls
	for i := 0; i < 5; i++ {
		instanceID, err := mock.GetInstanceID(context.Background())
		if err != nil {
			t.Errorf("Expected no error on call %d, got: %v", i+1, err)
		}

		expectedID := "i-1234567890abcdef0"
		if instanceID != expectedID {
			t.Errorf("Expected instance ID %s on call %d, got: %s", expectedID, i+1, instanceID)
		}
	}

	// Verify call count
	if mock.CallCount() != 5 {
		t.Errorf("Expected call count 5, got: %d", mock.CallCount())
	}
}

func TestMockIMDSClient_ThreadSafety(t *testing.T) {
	mock := NewMockIMDSClient()

	// Run multiple goroutines to test thread safety
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			instanceID, err := mock.GetInstanceID(context.Background())
			if err != nil {
				t.Errorf("Expected no error in goroutine %d, got: %v", id, err)
			}

			expectedID := "i-1234567890abcdef0"
			if instanceID != expectedID {
				t.Errorf("Expected instance ID %s in goroutine %d, got: %s", expectedID, id, instanceID)
			}

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

func TestMockIMDSClient_DifferentInstanceIDs(t *testing.T) {
	testCases := []string{
		"i-1234567890abcdef0",
		"i-abcdef1234567890",
		"i-0987654321fedcba",
		"i-fedcba0987654321",
	}

	for _, expectedID := range testCases {
		t.Run("InstanceID_"+expectedID, func(t *testing.T) {
			mock := NewMockIMDSClient()
			mock.SetInstanceID(expectedID)

			instanceID, err := mock.GetInstanceID(context.Background())
			if err != nil {
				t.Errorf("Expected no error for instance ID %s, got: %v", expectedID, err)
			}

			if instanceID != expectedID {
				t.Errorf("Expected instance ID %s, got: %s", expectedID, instanceID)
			}
		})
	}
}

func TestMockIMDSClient_ContextHandling(t *testing.T) {
	mock := NewMockIMDSClient()

	// Test with background context
	instanceID, err := mock.GetInstanceID(context.Background())
	if err != nil {
		t.Errorf("Expected no error with background context, got: %v", err)
	}
	if instanceID == "" {
		t.Error("Expected non-empty instance ID with background context")
	}

	// Test with context with timeout
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	instanceID, err = mock.GetInstanceID(ctx)
	if err != nil {
		t.Errorf("Expected no error with timeout context, got: %v", err)
	}
	if instanceID == "" {
		t.Error("Expected non-empty instance ID with timeout context")
	}
}

// Note: We cannot easily test DefaultIMDSClient without running on EC2
// or mocking the AWS SDK, so we focus on testing the interface and mocks.
// Integration tests will use mocks to verify IMDS integration.
func TestDefaultIMDSClient_Creation(t *testing.T) {
	// Test that we can create a DefaultIMDSClient instance
	client := NewDefaultIMDSClient()
	if client == nil {
		t.Error("Expected DefaultIMDSClient instance, got nil")
	}
}

func TestIMDSClient_Interface(t *testing.T) {
	// Test that MockIMDSClient implements IMDSClient interface
	var client IMDSClient = NewMockIMDSClient()

	// Should be able to call GetInstanceID through interface
	instanceID, err := client.GetInstanceID(context.Background())
	if err != nil {
		t.Errorf("Expected no error through interface, got: %v", err)
	}

	if instanceID == "" {
		t.Error("Expected non-empty instance ID through interface")
	}

	// Test that DefaultIMDSClient implements IMDSClient interface
	var defaultClient IMDSClient = NewDefaultIMDSClient()
	if defaultClient == nil {
		t.Error("Expected DefaultIMDSClient to implement IMDSClient interface")
	}
}
