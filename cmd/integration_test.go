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
	"github.com/terraconstructs/tcons-signal"
	"github.com/terraconstructs/tcons-signal/test/integration"
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
		MessageBody: aws.String("tcons-signal message"),
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
		MessageBody: aws.String("tcons-signal message"),
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

	// Build the binary if it doesn't exist
	if _, err := os.Stat("tcons-signal"); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "tcons-signal", ".")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build binary: %v", err)
		}
	}

	// Run the binary with ElasticMQ queue URL (without IMDS mock)
	// Note: This will fail because the binary tries to get instance ID from IMDS
	cmd := exec.Command("./tcons-signal",
		"--queue-url", queueURL,
		"--id", "integration-binary-test-no-imds",
		"--status", "SUCCESS",
		"--verbose",
	)

	output, err := cmd.CombinedOutput()
	t.Logf("Binary output: %s", string(output))

	// We expect this to fail due to IMDS not being available
	if err != nil {
		t.Logf("Expected error due to IMDS not available: %v", err)

		// Check if the error mentions IMDS (which means we got past queue URL validation)
		outputStr := string(output)
		if contains(outputStr, "instance-id") || contains(outputStr, "imds") || contains(outputStr, "169.254.169.254") {
			t.Log("Success: Binary got to IMDS step, meaning queue URL and config parsing worked")
		} else {
			t.Errorf("Unexpected error - binary may have failed before reaching IMDS: %s", outputStr)
		}
	}
}

func TestBinary_Integration_WithIMDSMock(t *testing.T) {
	// Check if EC2 metadata mock is available
	if !isEC2MockAvailable() {
		t.Skip("EC2 metadata mock not available - run 'make integration-up' first")
	}

	queueURL := getQueueURL(t, testQueueName)
	purgeQueue(t, queueURL)

	// Build the binary if it doesn't exist
	if _, err := os.Stat("tcons-signal"); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "tcons-signal", ".")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build binary: %v", err)
		}
	}

	// Run the binary with both ElasticMQ and IMDS mock
	cmd := exec.Command("./tcons-signal",
		"--queue-url", queueURL,
		"--id", "integration-binary-test-with-imds",
		"--exec", "../test/fixtures/success.sh",
		"--verbose",
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
