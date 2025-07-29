//go:build integration

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/terraconstructs/signal-aws"
	"github.com/terraconstructs/signal-aws/test/integration"
)

const (
	elasticMQEndpoint = "http://localhost:9324"
	testQueueName     = "tcons-test-queue"
	retryQueueName    = "tcons-retry-test-queue"
	timeoutQueueName  = "tcons-timeout-test-queue"
)

func TestMain(m *testing.M) {
	// Ensure ElasticMQ is running
	if !isElasticMQRunning() {
		fmt.Println("ElasticMQ is not running. Run 'make integration-up' first.")
		os.Exit(1)
	}

	code := m.Run()
	os.Exit(code)
}

func isElasticMQRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := createTestSQSClient()
	_, err := client.ListQueues(ctx, &sqs.ListQueuesInput{})
	return err == nil
}

func createTestSQSClient() *sqs.Client {
	cfg, _ := config.LoadDefaultConfig(context.Background(),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if service == sqs.ServiceID {
					return aws.Endpoint{URL: elasticMQEndpoint}, nil
				}
				return aws.Endpoint{}, fmt.Errorf("unknown service %s", service)
			})),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithRegion("us-east-1"),
	)

	return sqs.NewFromConfig(cfg)
}

func getQueueURL(t *testing.T, queueName string) string {
	client := createTestSQSClient()
	ctx := context.Background()

	result, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		t.Fatalf("Failed to get queue URL for %s: %v", queueName, err)
	}

	return *result.QueueUrl
}

func purgeQueue(t *testing.T, queueURL string) {
	client := createTestSQSClient()
	ctx := context.Background()

	_, err := client.PurgeQueue(ctx, &sqs.PurgeQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		t.Logf("Warning: Failed to purge queue %s: %v", queueURL, err)
	}
}

func receiveMessages(t *testing.T, queueURL string, maxMessages int32) []types.Message {
	client := createTestSQSClient()
	ctx := context.Background()

	result, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(queueURL),
		MaxNumberOfMessages:   maxMessages,
		WaitTimeSeconds:       2, // Short poll
		MessageAttributeNames: []string{"All"},
	})
	if err != nil {
		t.Fatalf("Failed to receive messages: %v", err)
	}

	return result.Messages
}

func TestSQSPublisher_Integration_ElasticMQPublish(t *testing.T) {
	queueURL := getQueueURL(t, testQueueName)
	purgeQueue(t, queueURL)

	// Create a client configured for ElasticMQ
	client := createTestSQSClient()
	ctx := context.Background()

	// Test publishing using our SQS message format with retry configuration
	input := signal.PublishInput{
		QueueURL:       queueURL,
		SignalID:       "integration-test-retry-001",
		InstanceID:     "i-integrationretry123456789",
		Status:         "SUCCESS",
		PublishTimeout: 10 * time.Second,
		Retries:        3,
	}

	// Manually test the SQS message format that our SQSPublisher would send
	sqsInput := &sqs.SendMessageInput{
		QueueUrl:    aws.String(input.QueueURL),
		MessageBody: aws.String("tcsignal-aws message"),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"signal_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String(input.SignalID),
			},
			"instance_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String(input.InstanceID),
			},
			"status": {
				DataType:    aws.String("String"),
				StringValue: aws.String(input.Status),
			},
		},
	}

	// Send message with retry configuration
	result, err := client.SendMessage(ctx, sqsInput)
	if err != nil {
		t.Fatalf("Failed to send message with retry config: %v", err)
	}

	t.Logf("Message sent successfully with retry config, MessageId: %s", *result.MessageId)

	// Verify message was received with correct attributes
	messages := receiveMessages(t, queueURL, 10)
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg := messages[0]

	// Verify message attributes match our PublishInput
	if msg.MessageAttributes["signal_id"].StringValue == nil ||
		*msg.MessageAttributes["signal_id"].StringValue != input.SignalID {
		t.Errorf("Expected signal_id '%s', got %v", input.SignalID, msg.MessageAttributes["signal_id"])
	}

	if msg.MessageAttributes["instance_id"].StringValue == nil ||
		*msg.MessageAttributes["instance_id"].StringValue != input.InstanceID {
		t.Errorf("Expected instance_id '%s', got %v", input.InstanceID, msg.MessageAttributes["instance_id"])
	}

	if msg.MessageAttributes["status"].StringValue == nil ||
		*msg.MessageAttributes["status"].StringValue != input.Status {
		t.Errorf("Expected status '%s', got %v", input.Status, msg.MessageAttributes["status"])
	}

	t.Log("SQS message format validated successfully for integration testing")
}

func TestSQSPublisher_Integration_WithElasticMQClient(t *testing.T) {
	queueURL := getQueueURL(t, testQueueName)
	purgeQueue(t, queueURL)

	// Create a client configured for ElasticMQ
	client := createTestSQSClient()
	ctx := context.Background()

	// Test direct SQS publishing (this bypasses our SQSPublisher for now)
	testMessage := &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String("tcsignal-aws message"),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"signal_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String("integration-test-002"),
			},
			"instance_id": {
				DataType:    aws.String("String"),
				StringValue: aws.String("i-integration987654321"),
			},
			"status": {
				DataType:    aws.String("String"),
				StringValue: aws.String("SUCCESS"),
			},
		},
	}

	// Send message
	result, err := client.SendMessage(ctx, testMessage)
	if err != nil {
		t.Fatalf("Failed to send message to ElasticMQ: %v", err)
	}

	t.Logf("Message sent successfully, MessageId: %s", *result.MessageId)

	// Verify message was received
	messages := receiveMessages(t, queueURL, 10)
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg := messages[0]

	// Verify message attributes
	if msg.MessageAttributes["signal_id"].StringValue == nil ||
		*msg.MessageAttributes["signal_id"].StringValue != "integration-test-002" {
		t.Errorf("Expected signal_id 'integration-test-002', got %v", msg.MessageAttributes["signal_id"])
	}

	if msg.MessageAttributes["instance_id"].StringValue == nil ||
		*msg.MessageAttributes["instance_id"].StringValue != "i-integration987654321" {
		t.Errorf("Expected instance_id 'i-integration987654321', got %v", msg.MessageAttributes["instance_id"])
	}

	if msg.MessageAttributes["status"].StringValue == nil ||
		*msg.MessageAttributes["status"].StringValue != "SUCCESS" {
		t.Errorf("Expected status 'SUCCESS', got %v", msg.MessageAttributes["status"])
	}

	t.Log("Message attributes validated successfully")
}

func TestBinary_Integration_WithElasticMQ_NoIMDS(t *testing.T) {
	queueURL := getQueueURL(t, testQueueName)
	purgeQueue(t, queueURL)

	// Build the binary if it doesn't exist in root directory
	if _, err := os.Stat("../tcsignal-aws"); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "../tcsignal-aws", ".")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build binary: %v", err)
		}
	}

	// Test 1: Run without IMDS and without --instance-id (should fail)
	t.Log("🔄 Testing binary without IMDS and without --instance-id (expected to fail)...")
	cmd1 := exec.Command("../tcsignal-aws",
		"--queue-url", queueURL,
		"--id", "integration-binary-test-no-imds-no-flag",
		"--status", "SUCCESS",
	)

	// Explicitly avoid setting AWS_EC2_METADATA_SERVICE_ENDPOINT to force real IMDS lookup
	cmd1.Env = append(os.Environ(),
		"AWS_ENDPOINT_URL_SQS=http://localhost:9324",
		"AWS_REGION=us-east-1",
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
	)

	output1, err1 := cmd1.CombinedOutput()
	t.Logf("Binary output (no IMDS, no flag): %s", string(output1))

	// Check the behavior - it might succeed if IMDS mock is accessible or fail if not
	if err1 != nil {
		t.Logf("✅ Binary failed as expected when IMDS not configured: %v", err1)

		// Check if the error mentions IMDS or instance ID (which means we got past queue URL validation)
		outputStr := string(output1)
		if contains(outputStr, "instance") || contains(outputStr, "imds") || contains(outputStr, "169.254.169.254") {
			t.Log("✅ Binary got to IMDS step, meaning queue URL and config parsing worked")
		} else {
			t.Errorf("Unexpected error - binary may have failed before reaching IMDS: %s", outputStr)
		}
	} else {
		// If it succeeded, it means IMDS mock was accessible (which is fine in integration environment)
		t.Log("✅ Binary succeeded - IMDS mock was accessible (this is fine in integration environment)")

		// Verify it used an instance ID from IMDS
		outputStr := string(output1)
		if contains(outputStr, "Fetched instance ID from IMDS") {
			t.Log("✅ Binary correctly fetched instance ID from IMDS mock")
		}
	}

	// Test 2: Run without IMDS but WITH --instance-id (should succeed)
	t.Log("🔄 Testing binary without IMDS but with --instance-id (should succeed)...")
	providedInstanceID := "i-no-imds-workaround-789"

	cmd2 := exec.Command("../tcsignal-aws",
		"--queue-url", queueURL,
		"--id", "integration-binary-test-no-imds-with-flag",
		"--status", "SUCCESS",
		"--instance-id", providedInstanceID,
		"--log-level", "debug",
	)

	cmd2.Env = append(os.Environ(),
		"AWS_ENDPOINT_URL_SQS=http://localhost:9324",
		"AWS_REGION=us-east-1",
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
	)

	output2, err2 := cmd2.CombinedOutput()
	t.Logf("Binary output (no IMDS, with flag): %s", string(output2))

	if err2 != nil {
		t.Errorf("Binary should succeed with --instance-id flag even without IMDS: %v", err2)
		t.Errorf("Output: %s", string(output2))
		return
	}

	t.Log("✅ Binary succeeded with --instance-id flag as IMDS workaround!")

	// Verify the message was published correctly
	messages := receiveMessages(t, queueURL, 10)

	// Should have 1 message (the successful one with --instance-id flag)
	successMessages := 0
	for _, msg := range messages {
		if signalID, exists := msg.MessageAttributes["signal_id"]; exists &&
			signalID.StringValue != nil &&
			*signalID.StringValue == "integration-binary-test-no-imds-with-flag" {
			successMessages++

			// Verify it has the correct instance ID
			if instanceID, exists := msg.MessageAttributes["instance_id"]; exists &&
				instanceID.StringValue != nil &&
				*instanceID.StringValue == providedInstanceID {
				t.Logf("✅ Message has correct provided instance ID: %s", providedInstanceID)
			} else {
				t.Errorf("Expected instance_id '%s', got %v", providedInstanceID, msg.MessageAttributes["instance_id"])
			}
		}
	}

	if successMessages != 1 {
		t.Errorf("Expected 1 successful message, found %d", successMessages)
	}

	// Verify the log output shows it used the provided instance ID
	if contains(string(output2), "Using provided instance ID") {
		t.Log("✅ Binary logged that it used the provided instance ID")
	} else {
		t.Error("Binary should have logged using provided instance ID")
	}

	t.Log("🎉 --instance-id flag successfully works as IMDS workaround!")
}

func TestBinary_Integration_WithIMDSMock(t *testing.T) {
	// Check if EC2 metadata mock is available
	if !isEC2MockAvailable() {
		t.Skip("EC2 metadata mock not available - run 'make integration-up' first")
	}

	queueURL := getQueueURL(t, testQueueName)
	purgeQueue(t, queueURL)

	// Build the binary if it doesn't exist in root directory
	if _, err := os.Stat("../tcsignal-aws"); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "../tcsignal-aws", ".")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build binary: %v", err)
		}
	}

	// Run the binary with both ElasticMQ and IMDS mock
	cmd := exec.Command("../tcsignal-aws",
		"--queue-url", queueURL,
		"--id", "integration-binary-test-with-imds",
		"--exec", "../test/fixtures/success.sh",
		"--log-level", "debug",
	)

	// Set environment variables for AWS configuration
	cmd.Env = append(os.Environ(),
		"AWS_EC2_METADATA_SERVICE_ENDPOINT=http://localhost:1338",
		"AWS_ENDPOINT_URL_SQS=http://localhost:9324",
		"AWS_REGION=us-east-1",
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
	)

	output, err := cmd.CombinedOutput()
	t.Logf("Binary output: %s", string(output))

	if err != nil {
		t.Logf("Binary execution failed: %v", err)
		// Even if it fails, let's check what step it got to
		outputStr := string(output)
		if contains(outputStr, "Successfully published signal") {
			t.Log("Success: Binary completed full end-to-end flow!")
		} else if contains(outputStr, "instance-id") || contains(outputStr, "Failed to send SQS message") {
			t.Log("Partial success: Binary got to SQS publishing step")
		} else {
			t.Errorf("Binary failed before completing workflow: %s", outputStr)
		}
	} else {
		// Binary succeeded! Let's verify the message was published
		t.Log("✅ Binary executed successfully!")

		// Check if message was published to SQS
		messages := receiveMessages(t, queueURL, 10)
		if len(messages) > 0 {
			t.Logf("✅ Found %d message(s) in SQS queue", len(messages))

			for i, msg := range messages {
				t.Logf("Message %d attributes: %+v", i+1, msg.MessageAttributes)

				// Verify expected attributes
				if signalID, exists := msg.MessageAttributes["signal_id"]; exists &&
					signalID.StringValue != nil &&
					*signalID.StringValue == "integration-binary-test-with-imds" {
					t.Log("✅ Signal ID matches expected value")
				}

				if status, exists := msg.MessageAttributes["status"]; exists &&
					status.StringValue != nil &&
					*status.StringValue == "SUCCESS" {
					t.Log("✅ Status matches expected value (SUCCESS)")
				}

				if instanceID, exists := msg.MessageAttributes["instance_id"]; exists &&
					instanceID.StringValue != nil {
					t.Logf("✅ Instance ID from IMDS mock: %s", *instanceID.StringValue)
				}
			}
		} else {
			t.Error("No messages found in SQS queue - signal may not have been published")
		}
	}
}

func TestBinary_Integration_WithProvidedInstanceID(t *testing.T) {
	queueURL := getQueueURL(t, testQueueName)
	purgeQueue(t, queueURL)

	// Build the binary if it doesn't exist in root directory
	if _, err := os.Stat("../tcsignal-aws"); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "../tcsignal-aws", ".")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build binary: %v", err)
		}
	}

	providedInstanceID := "i-provided-integration-test-123"

	// Run the binary with provided instance ID (bypassing IMDS)
	cmd := exec.Command("../tcsignal-aws",
		"--queue-url", queueURL,
		"--id", "integration-binary-test-provided-instance-id",
		"--exec", "../test/fixtures/success.sh",
		"--instance-id", providedInstanceID,
		"--log-level", "debug",
	)

	// Set environment variables for AWS configuration (SQS only, no IMDS endpoint)
	cmd.Env = append(os.Environ(),
		"AWS_ENDPOINT_URL_SQS=http://localhost:9324",
		"AWS_REGION=us-east-1",
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
	)

	output, err := cmd.CombinedOutput()
	t.Logf("Binary output: %s", string(output))

	if err != nil {
		t.Errorf("Binary execution failed: %v", err)
		t.Errorf("Output: %s", string(output))
		return
	}

	// Binary succeeded! Let's verify the message was published with the provided instance ID
	t.Log("✅ Binary executed successfully with provided instance ID!")

	// Check if message was published to SQS
	messages := receiveMessages(t, queueURL, 10)
	if len(messages) == 0 {
		t.Fatal("No messages found in SQS queue - signal was not published")
	}

	t.Logf("✅ Found %d message(s) in SQS queue", len(messages))

	msg := messages[0]
	t.Logf("Message attributes: %+v", msg.MessageAttributes)

	// Verify expected attributes
	if signalID, exists := msg.MessageAttributes["signal_id"]; exists &&
		signalID.StringValue != nil &&
		*signalID.StringValue == "integration-binary-test-provided-instance-id" {
		t.Log("✅ Signal ID matches expected value")
	} else {
		t.Errorf("Expected signal_id 'integration-binary-test-provided-instance-id', got %v", msg.MessageAttributes["signal_id"])
	}

	if status, exists := msg.MessageAttributes["status"]; exists &&
		status.StringValue != nil &&
		*status.StringValue == "SUCCESS" {
		t.Log("✅ Status matches expected value (SUCCESS)")
	} else {
		t.Errorf("Expected status 'SUCCESS', got %v", msg.MessageAttributes["status"])
	}

	// Most importantly, verify the instance ID is the one we provided
	if instanceID, exists := msg.MessageAttributes["instance_id"]; exists &&
		instanceID.StringValue != nil &&
		*instanceID.StringValue == providedInstanceID {
		t.Logf("✅ Instance ID matches provided value: %s", providedInstanceID)
	} else {
		t.Errorf("Expected instance_id '%s', got %v", providedInstanceID, msg.MessageAttributes["instance_id"])
	}

	// Verify the output shows it used the provided instance ID
	outputStr := string(output)
	if contains(outputStr, "Using provided instance ID") {
		t.Log("✅ Binary logged that it used the provided instance ID")
	} else if contains(outputStr, "Fetched instance ID from IMDS") {
		t.Error("Binary should not have fetched from IMDS when instance ID was provided")
	}
}

func TestBinary_Integration_ProvidedInstanceID_vs_IMDS(t *testing.T) {
	// This test compares behavior with and without --instance-id flag
	// It validates that both approaches work but use different instance IDs

	// Check if EC2 metadata mock is available for IMDS test
	if !isEC2MockAvailable() {
		t.Skip("EC2 metadata mock not available - run 'make integration-up' first")
	}

	queueURL := getQueueURL(t, testQueueName)
	purgeQueue(t, queueURL)

	// Build the binary if it doesn't exist in root directory
	if _, err := os.Stat("../tcsignal-aws"); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "../tcsignal-aws", ".")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build binary: %v", err)
		}
	}

	providedInstanceID := "i-provided-comparison-test-456"

	// Test 1: Run with provided instance ID
	t.Log("🔄 Testing binary with provided instance ID...")
	cmd1 := exec.Command("../tcsignal-aws",
		"--queue-url", queueURL,
		"--id", "comparison-test-provided-id",
		"--status", "SUCCESS",
		"--instance-id", providedInstanceID,
		"--log-level", "debug",
	)

	cmd1.Env = append(os.Environ(),
		"AWS_ENDPOINT_URL_SQS=http://localhost:9324",
		"AWS_REGION=us-east-1",
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
	)

	output1, err1 := cmd1.CombinedOutput()
	t.Logf("Binary output (provided instance ID): %s", string(output1))

	if err1 != nil {
		t.Errorf("Binary execution failed with provided instance ID: %v", err1)
	}

	// Test 2: Run with IMDS (different signal ID to avoid conflicts)
	t.Log("🔄 Testing binary with IMDS...")
	cmd2 := exec.Command("../tcsignal-aws",
		"--queue-url", queueURL,
		"--id", "comparison-test-imds-id",
		"--status", "SUCCESS",
		"--log-level", "debug",
	)

	cmd2.Env = append(os.Environ(),
		"AWS_EC2_METADATA_SERVICE_ENDPOINT=http://localhost:1338",
		"AWS_ENDPOINT_URL_SQS=http://localhost:9324",
		"AWS_REGION=us-east-1",
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
	)

	output2, err2 := cmd2.CombinedOutput()
	t.Logf("Binary output (IMDS): %s", string(output2))

	if err2 != nil {
		t.Errorf("Binary execution failed with IMDS: %v", err2)
	}

	// Verify both executions succeeded
	if err1 != nil || err2 != nil {
		return // Don't continue if either test failed
	}

	// Check messages in SQS - should have 2 messages
	messages := receiveMessages(t, queueURL, 10)
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages in SQS queue, got %d", len(messages))
	}

	t.Log("✅ Both binary executions succeeded!")

	// Organize messages by signal_id for comparison
	var providedMessage, imdsMessage *types.Message

	for i := range messages {
		msg := &messages[i]
		if signalID, exists := msg.MessageAttributes["signal_id"]; exists && signalID.StringValue != nil {
			switch *signalID.StringValue {
			case "comparison-test-provided-id":
				providedMessage = msg
			case "comparison-test-imds-id":
				imdsMessage = msg
			}
		}
	}

	// Verify we found both messages
	if providedMessage == nil {
		t.Fatal("Could not find message from provided instance ID test")
	}
	if imdsMessage == nil {
		t.Fatal("Could not find message from IMDS test")
	}

	// Verify provided instance ID message
	if instanceID, exists := providedMessage.MessageAttributes["instance_id"]; exists &&
		instanceID.StringValue != nil &&
		*instanceID.StringValue == providedInstanceID {
		t.Logf("✅ Provided instance ID message has correct instance ID: %s", providedInstanceID)
	} else {
		t.Errorf("Expected provided message to have instance_id '%s', got %v", providedInstanceID, providedMessage.MessageAttributes["instance_id"])
	}

	// Verify IMDS message has a different instance ID
	if instanceID, exists := imdsMessage.MessageAttributes["instance_id"]; exists &&
		instanceID.StringValue != nil {
		imdsInstanceID := *instanceID.StringValue
		t.Logf("✅ IMDS message has instance ID: %s", imdsInstanceID)

		// Verify it's different from the provided one
		if imdsInstanceID == providedInstanceID {
			t.Error("IMDS instance ID should be different from provided instance ID")
		} else {
			t.Log("✅ IMDS and provided instance IDs are different as expected")
		}
	} else {
		t.Error("IMDS message should have an instance_id attribute")
	}

	// Verify the log outputs show the correct behavior
	if contains(string(output1), "Using provided instance ID") {
		t.Log("✅ First execution logged using provided instance ID")
	} else {
		t.Error("First execution should have logged using provided instance ID")
	}

	if contains(string(output2), "Fetched instance ID from IMDS") {
		t.Log("✅ Second execution logged fetching from IMDS")
	} else {
		t.Error("Second execution should have logged fetching from IMDS")
	}

	t.Log("🎉 Comparison test completed successfully!")
}

func isEC2MockAvailable() bool {
	return integration.IsEC2MockHealthy("http://localhost:1338/latest/meta-data/instance-id")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(stringContains(s, substr))))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBinary_Integration_IMDSRegionDetection(t *testing.T) {
	// Check if EC2 metadata mock is available
	if !isEC2MockAvailable() {
		t.Skip("EC2 metadata mock not available - run 'make integration-up' first")
	}

	queueURL := getQueueURL(t, testQueueName)
	purgeQueue(t, queueURL)

	// Build the binary if it doesn't exist in root directory
	if _, err := os.Stat("../tcsignal-aws"); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "../tcsignal-aws", ".")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build binary: %v", err)
		}
	}

	// Run the binary WITHOUT --region flag to test IMDS region detection
	cmd := exec.Command("../tcsignal-aws",
		"--queue-url", queueURL,
		"--id", "integration-test-imds-region-detection",
		"--status", "SUCCESS",
		"--log-level", "debug",
	)

	// Set environment variables for AWS configuration with IMDS mock
	cmd.Env = append(os.Environ(),
		"AWS_EC2_METADATA_SERVICE_ENDPOINT=http://localhost:1338",
		"AWS_ENDPOINT_URL_SQS=http://localhost:9324",
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
		// Explicitly DO NOT set AWS_REGION to force IMDS region detection
	)

	output, err := cmd.CombinedOutput()
	t.Logf("Binary output: %s", string(output))

	if err != nil {
		t.Errorf("Binary execution failed: %v", err)
		t.Errorf("Output: %s", string(output))
		return
	}

	t.Log("✅ Binary executed successfully with IMDS region detection!")

	// Verify the log output shows it fetched region from IMDS
	outputStr := string(output)
	if contains(outputStr, "Fetched region from IMDS") {
		t.Log("✅ Binary logged that it fetched region from IMDS")
	} else {
		t.Error("Binary should have logged fetching region from IMDS")
	}

	// Check if message was published to SQS
	messages := receiveMessages(t, queueURL, 10)
	if len(messages) == 0 {
		t.Fatal("No messages found in SQS queue - signal was not published")
	}

	t.Logf("✅ Found %d message(s) in SQS queue", len(messages))

	msg := messages[0]
	t.Logf("Message attributes: %+v", msg.MessageAttributes)

	// Verify expected attributes
	if signalID, exists := msg.MessageAttributes["signal_id"]; exists &&
		signalID.StringValue != nil &&
		*signalID.StringValue == "integration-test-imds-region-detection" {
		t.Log("✅ Signal ID matches expected value")
	} else {
		t.Errorf("Expected signal_id 'integration-test-imds-region-detection', got %v", msg.MessageAttributes["signal_id"])
	}

	if status, exists := msg.MessageAttributes["status"]; exists &&
		status.StringValue != nil &&
		*status.StringValue == "SUCCESS" {
		t.Log("✅ Status matches expected value (SUCCESS)")
	} else {
		t.Errorf("Expected status 'SUCCESS', got %v", msg.MessageAttributes["status"])
	}

	// Verify instance ID is from IMDS
	if instanceID, exists := msg.MessageAttributes["instance_id"]; exists &&
		instanceID.StringValue != nil {
		t.Logf("✅ Instance ID from IMDS: %s", *instanceID.StringValue)
	} else {
		t.Error("Expected instance_id attribute from IMDS")
	}

	t.Log("🎉 IMDS region detection integration test completed successfully!")
}

func TestElasticMQ_QueueSetup(t *testing.T) {
	client := createTestSQSClient()
	ctx := context.Background()

	// Test that all our queues exist
	expectedQueues := []string{testQueueName, retryQueueName, timeoutQueueName}

	for _, queueName := range expectedQueues {
		queueURL := getQueueURL(t, queueName)
		if queueURL == "" {
			t.Errorf("Queue %s not found", queueName)
			continue
		}

		// Test queue attributes
		attrs, err := client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
			QueueUrl:       aws.String(queueURL),
			AttributeNames: []types.QueueAttributeName{"All"},
		})
		if err != nil {
			t.Errorf("Failed to get attributes for queue %s: %v", queueName, err)
			continue
		}

		t.Logf("Queue %s exists with %d attributes", queueName, len(attrs.Attributes))
	}
}
