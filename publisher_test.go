package signal

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMockPublisher_Basic(t *testing.T) {
	mock := NewMockPublisher()

	input := PublishInput{
		QueueURL:       "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
		SignalID:       "test-signal-123",
		InstanceID:     "i-1234567890abcdef0",
		Status:         "SUCCESS",
		PublishTimeout: 10 * time.Second,
		Retries:        3,
	}

	// Test successful publish
	err := mock.Publish(context.Background(), input)
	if err != nil {
		t.Errorf("Expected no error from mock publisher, got: %v", err)
	}

	// Verify call was recorded
	calls := mock.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 call recorded, got: %d", len(calls))
	}

	if calls[0] != input {
		t.Errorf("Expected call to match input")
	}

	// Test call count
	if mock.CallCount() != 1 {
		t.Errorf("Expected call count 1, got: %d", mock.CallCount())
	}
}

func TestMockPublisher_GetLastCall(t *testing.T) {
	mock := NewMockPublisher()

	input1 := PublishInput{
		QueueURL: "queue1",
		SignalID: "signal1",
		Status:   "SUCCESS",
		Retries:  3,
	}

	input2 := PublishInput{
		QueueURL: "queue2",
		SignalID: "signal2",
		Status:   "FAILURE",
		Retries:  5,
	}

	// Make first call
	mock.Publish(context.Background(), input1)

	lastCall := mock.GetLastCall()
	if lastCall == nil {
		t.Fatal("Expected last call to be recorded")
	}
	if lastCall.SignalID != "signal1" {
		t.Errorf("Expected last call SignalID to be 'signal1', got: %s", lastCall.SignalID)
	}

	// Make second call
	mock.Publish(context.Background(), input2)

	lastCall = mock.GetLastCall()
	if lastCall == nil {
		t.Fatal("Expected last call to be recorded")
	}
	if lastCall.SignalID != "signal2" {
		t.Errorf("Expected last call SignalID to be 'signal2', got: %s", lastCall.SignalID)
	}
}

func TestMockPublisher_SetError(t *testing.T) {
	mock := NewMockPublisher()
	expectedErr := fmt.Errorf("mock publish error")
	mock.SetError(expectedErr)

	input := PublishInput{
		QueueURL: "test-queue",
		SignalID: "test-signal",
		Status:   "SUCCESS",
		Retries:  3,
	}

	err := mock.Publish(context.Background(), input)
	if err != expectedErr {
		t.Errorf("Expected mock error, got: %v", err)
	}

	// Call should still be recorded even on error
	if mock.CallCount() != 1 {
		t.Errorf("Expected call count 1 even on error, got: %d", mock.CallCount())
	}
}

func TestMockPublisher_RetryLogic(t *testing.T) {
	mock := NewMockPublisher()

	// Set to fail first 2 calls, succeed on 3rd
	mock.SetFailFirstNCalls(2)

	input := PublishInput{
		QueueURL: "test-queue",
		SignalID: "test-signal",
		Status:   "SUCCESS",
		Retries:  3,
	}

	// First call should fail
	err := mock.Publish(context.Background(), input)
	if err == nil {
		t.Error("Expected first call to fail")
	}
	if mock.CallCount() != 1 {
		t.Errorf("Expected call count 1 after first call, got: %d", mock.CallCount())
	}

	// Second call should fail
	err = mock.Publish(context.Background(), input)
	if err == nil {
		t.Error("Expected second call to fail")
	}
	if mock.CallCount() != 2 {
		t.Errorf("Expected call count 2 after second call, got: %d", mock.CallCount())
	}

	// Third call should succeed
	err = mock.Publish(context.Background(), input)
	if err != nil {
		t.Errorf("Expected third call to succeed, got: %v", err)
	}
	if mock.CallCount() != 3 {
		t.Errorf("Expected call count 3 after third call, got: %d", mock.CallCount())
	}
}

func TestMockPublisher_MessageAttributes(t *testing.T) {
	mock := NewMockPublisher()

	input := PublishInput{
		QueueURL:       "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
		SignalID:       "deployment-abc-123",
		InstanceID:     "i-0123456789abcdef0",
		Status:         "FAILURE",
		PublishTimeout: 5 * time.Second,
		Retries:        2,
	}

	err := mock.Publish(context.Background(), input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	lastCall := mock.GetLastCall()
	if lastCall == nil {
		t.Fatal("Expected call to be recorded")
	}

	// Verify all message attributes are captured correctly
	if lastCall.QueueURL != input.QueueURL {
		t.Errorf("Expected QueueURL %s, got %s", input.QueueURL, lastCall.QueueURL)
	}

	if lastCall.SignalID != input.SignalID {
		t.Errorf("Expected SignalID %s, got %s", input.SignalID, lastCall.SignalID)
	}

	if lastCall.InstanceID != input.InstanceID {
		t.Errorf("Expected InstanceID %s, got %s", input.InstanceID, lastCall.InstanceID)
	}

	if lastCall.Status != input.Status {
		t.Errorf("Expected Status %s, got %s", input.Status, lastCall.Status)
	}

	if lastCall.PublishTimeout != input.PublishTimeout {
		t.Errorf("Expected PublishTimeout %v, got %v", input.PublishTimeout, lastCall.PublishTimeout)
	}

	if lastCall.Retries != input.Retries {
		t.Errorf("Expected Retries %d, got %d", input.Retries, lastCall.Retries)
	}
}

func TestMockPublisher_StatusValues(t *testing.T) {
	mock := NewMockPublisher()

	testCases := []string{"SUCCESS", "FAILURE"}

	for _, status := range testCases {
		t.Run("Status_"+status, func(t *testing.T) {
			input := PublishInput{
				QueueURL: "test-queue",
				SignalID: "test-signal",
				Status:   status,
			}

			err := mock.Publish(context.Background(), input)
			if err != nil {
				t.Errorf("Expected no error for status %s, got: %v", status, err)
			}

			lastCall := mock.GetLastCall()
			if lastCall.Status != status {
				t.Errorf("Expected status %s, got %s", status, lastCall.Status)
			}
		})
	}
}

func TestMockPublisher_ThreadSafety(t *testing.T) {
	mock := NewMockPublisher()

	// Run multiple goroutines to test thread safety
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			input := PublishInput{
				QueueURL: "test-queue",
				SignalID: fmt.Sprintf("signal-%d", id),
				Status:   "SUCCESS",
			}
			mock.Publish(context.Background(), input)
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

func TestPublishInput_Struct(t *testing.T) {
	// Test that PublishInput struct contains all required fields
	input := PublishInput{
		QueueURL:       "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue",
		SignalID:       "test-signal-123",
		InstanceID:     "i-1234567890abcdef0",
		Status:         "SUCCESS",
		PublishTimeout: 10 * time.Second,
		Retries:        3,
	}

	// Verify all fields are set
	if input.QueueURL == "" {
		t.Error("QueueURL should not be empty")
	}
	if input.SignalID == "" {
		t.Error("SignalID should not be empty")
	}
	if input.InstanceID == "" {
		t.Error("InstanceID should not be empty")
	}
	if input.Status == "" {
		t.Error("Status should not be empty")
	}
	if input.PublishTimeout == 0 {
		t.Error("PublishTimeout should not be zero")
	}
	if input.Retries < 0 {
		t.Error("Retries should not be negative")
	}
}

// Note: We cannot easily test SQSPublisher without real AWS credentials
// and SQS service, so we focus on testing the interface and mocks.
// Integration tests will use mocks to verify the SQS message format.
func TestSQSPublisher_Creation(t *testing.T) {
	// Test that we can create an SQSPublisher instance
	publisher := NewSQSPublisher(createTestLogger())
	if publisher == nil {
		t.Error("Expected SQSPublisher instance, got nil")
	}

	// Test with verbose mode
	verbosePublisher := NewSQSPublisher(createTestLogger())
	if verbosePublisher == nil {
		t.Error("Expected verbose SQSPublisher instance, got nil")
	}
}

func TestMockPublisher_RetryConfiguration(t *testing.T) {
	mock := NewMockPublisher()

	testCases := []struct {
		name    string
		retries int
	}{
		{"Default retries", 3},
		{"No retries", 0},
		{"High retries", 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := PublishInput{
				QueueURL: "test-queue",
				SignalID: "test-signal",
				Status:   "SUCCESS",
				Retries:  tc.retries,
			}

			err := mock.Publish(context.Background(), input)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			lastCall := mock.GetLastCall()
			if lastCall == nil {
				t.Fatal("Expected call to be recorded")
			}

			if lastCall.Retries != tc.retries {
				t.Errorf("Expected retries %d, got %d", tc.retries, lastCall.Retries)
			}
		})
	}
}
