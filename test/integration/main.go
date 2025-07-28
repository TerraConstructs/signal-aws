// Package integration provides shared helper functions for integration testing
package integration

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// StartEnvironment starts ElasticMQ and EC2 metadata mock services
func StartEnvironment() error {
	fmt.Println("Starting integration test environment...")

	// Start docker compose
	fmt.Println("Starting ElasticMQ and EC2 metadata mock...")
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start docker compose: %v", err)
	}

	// Wait for ElasticMQ to be ready
	fmt.Println("Waiting for ElasticMQ to be ready...")
	if !WaitForService("http://localhost:9324/", 30*time.Second, IsElasticMQHealthy) {
		return fmt.Errorf("ElasticMQ failed to start within 30 seconds")
	}
	fmt.Println("✅ ElasticMQ is ready!")

	// Wait for EC2 metadata mock to be ready
	fmt.Println("Waiting for EC2 metadata mock to be ready...")
	if !WaitForService("http://localhost:1338/latest/meta-data/instance-id", 30*time.Second, IsEC2MockHealthy) {
		return fmt.Errorf("EC2 metadata mock failed to start within 30 seconds")
	}
	fmt.Println("✅ EC2 metadata mock is ready!")

	fmt.Println("🎉 Integration test environment is ready!")
	return nil
}

// StopEnvironment stops the integration test services
func StopEnvironment() error {
	fmt.Println("Stopping integration test environment...")

	// Stop docker compose
	cmd := exec.Command("docker", "compose", "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop docker compose: %v", err)
	}

	fmt.Println("✅ Integration test environment stopped!")
	return nil
}

// RunFullTest runs the complete integration test suite
func RunFullTest() error {
	fmt.Println("Running full integration test suite...")

	// Build the binary first
	fmt.Println("Building tcsignal-aws binary...")
	if err := RunCommand("go", "build", "-o", "tcsignal-aws", "./cmd"); err != nil {
		return fmt.Errorf("failed to build binary: %v", err)
	}
	fmt.Println("✅ Binary built successfully!")

	// Start the integration environment
	fmt.Println("Starting integration environment...")
	if err := StartEnvironment(); err != nil {
		return fmt.Errorf("failed to start integration environment: %v", err)
	}

	// Set environment variables for AWS configuration
	os.Setenv("AWS_EC2_METADATA_SERVICE_ENDPOINT", "http://localhost:1338")
	os.Setenv("AWS_ENDPOINT_URL_SQS", "http://localhost:9324")
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test")

	// Run integration tests
	fmt.Println("Running integration tests...")
	testErr := RunCommand("go", "test", "-v", "./cmd", "-tags=integration")

	// Always try to clean up, even if tests failed
	fmt.Println("Stopping integration environment...")
	cleanupErr := StopEnvironment()

	// Report final results
	if testErr != nil {
		return fmt.Errorf("integration tests failed: %v", testErr)
	}

	if cleanupErr != nil {
		log.Printf("Warning: Failed to cleanup integration environment: %v", cleanupErr)
	}

	fmt.Println("🎉 All integration tests passed!")
	return nil
}

// WaitForService waits for a service to be healthy with timeout
func WaitForService(url string, timeout time.Duration, healthCheck func(string) bool) bool {
	deadline := time.Now().Add(timeout)
	attempt := 1

	for time.Now().Before(deadline) {
		if healthCheck(url) {
			return true
		}

		fmt.Printf("   Waiting... (attempt %d)\n", attempt)
		attempt++
		time.Sleep(1 * time.Second)
	}

	return false
}

// IsElasticMQHealthy checks if ElasticMQ service is healthy
func IsElasticMQHealthy(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// ElasticMQ returns 400 for root path, which is expected
	return resp.StatusCode == 400
}

// IsEC2MockHealthy checks if EC2 metadata mock service is healthy
func IsEC2MockHealthy(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// EC2 metadata mock should return 200 for instance-id endpoint
	return resp.StatusCode == 200
}

// RunCommand executes a command with output displayed
func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
