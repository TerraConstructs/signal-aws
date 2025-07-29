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

func TestMockIMDSClient_SetInstanceIDError(t *testing.T) {
	mock := NewMockIMDSClient()
	expectedErr := fmt.Errorf("mock IMDS error")
	mock.SetInstanceIDError(expectedErr)

	instanceID, err := mock.GetInstanceID(context.Background())
	if err != expectedErr {
		t.Errorf("Expected mock error, got: %v", err)
	}

	// Instance ID should be empty on error
	if instanceID != "" {
		t.Errorf("Expected empty instance ID on error, got: %s", instanceID)
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

	// Should be able to call GetRegion through interface
	region, err := client.GetRegion(context.Background())
	if err != nil {
		t.Errorf("Expected no error through interface, got: %v", err)
	}

	if region == "" {
		t.Error("Expected non-empty region through interface")
	}

	// Test that DefaultIMDSClient implements IMDSClient interface
	var defaultClient IMDSClient = NewDefaultIMDSClient()
	if defaultClient == nil {
		t.Error("Expected DefaultIMDSClient to implement IMDSClient interface")
	}
}

// Test region functionality
func TestMockIMDSClient_BasicRegion(t *testing.T) {
	mock := NewMockIMDSClient()

	// Test default behavior (should return default fake region)
	region, err := mock.GetRegion(context.Background())
	if err != nil {
		t.Errorf("Expected no error from mock IMDS client, got: %v", err)
	}

	expectedRegion := "us-east-1"
	if region != expectedRegion {
		t.Errorf("Expected default region %s, got: %s", expectedRegion, region)
	}

	// Verify call count is incremented
	if mock.CallCount() != 1 {
		t.Errorf("Expected call count 1, got: %d", mock.CallCount())
	}
}

func TestMockIMDSClient_SetRegion(t *testing.T) {
	mock := NewMockIMDSClient()
	customRegion := "eu-west-1"
	mock.SetRegion(customRegion)

	region, err := mock.GetRegion(context.Background())
	if err != nil {
		t.Errorf("Expected no error from mock IMDS client, got: %v", err)
	}

	if region != customRegion {
		t.Errorf("Expected custom region %s, got: %s", customRegion, region)
	}
}

func TestMockIMDSClient_SetRegionError(t *testing.T) {
	mock := NewMockIMDSClient()
	expectedErr := fmt.Errorf("mock IMDS region error")
	mock.SetRegionError(expectedErr)

	region, err := mock.GetRegion(context.Background())
	if err != expectedErr {
		t.Errorf("Expected mock error, got: %v", err)
	}

	// Region should be empty on error
	if region != "" {
		t.Errorf("Expected empty region on error, got: %s", region)
	}

	// Call should still be counted
	if mock.CallCount() != 1 {
		t.Errorf("Expected call count 1 even on error, got: %d", mock.CallCount())
	}
}

func TestMockIMDSClient_BothInstanceIDAndRegion(t *testing.T) {
	mock := NewMockIMDSClient()
	customID := "i-custom123456789"
	customRegion := "ap-southeast-1"

	mock.SetInstanceID(customID)
	mock.SetRegion(customRegion)

	// Test instance ID
	instanceID, err := mock.GetInstanceID(context.Background())
	if err != nil {
		t.Errorf("Expected no error for instance ID, got: %v", err)
	}
	if instanceID != customID {
		t.Errorf("Expected instance ID %s, got: %s", customID, instanceID)
	}

	// Test region
	region, err := mock.GetRegion(context.Background())
	if err != nil {
		t.Errorf("Expected no error for region, got: %v", err)
	}
	if region != customRegion {
		t.Errorf("Expected region %s, got: %s", customRegion, region)
	}

	// Should have 2 calls total
	if mock.CallCount() != 2 {
		t.Errorf("Expected call count 2, got: %d", mock.CallCount())
	}
}

func TestMockIMDSClient_DifferentRegions(t *testing.T) {
	testCases := []string{
		"us-east-1",
		"us-west-2",
		"eu-west-1",
		"ap-southeast-1",
	}

	for _, expectedRegion := range testCases {
		t.Run("Region_"+expectedRegion, func(t *testing.T) {
			mock := NewMockIMDSClient()
			mock.SetRegion(expectedRegion)

			region, err := mock.GetRegion(context.Background())
			if err != nil {
				t.Errorf("Expected no error for region %s, got: %v", expectedRegion, err)
			}

			if region != expectedRegion {
				t.Errorf("Expected region %s, got: %s", expectedRegion, region)
			}
		})
	}
}
