package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/terraconstructs/tcons-signal"
)

func main() {
	cfg, err := signal.ParseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	// Set up overall timeout context
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Create component instances
	executor := signal.NewDefaultExecutor(cfg.Verbose)
	publisher := signal.NewSQSPublisher(cfg.Verbose)
	imdsClient := signal.NewDefaultIMDSClient()

	result, err := run(ctx, *cfg, executor, publisher, imdsClient)
	if err != nil {
		log.Printf("Error: %v", err)
		os.Exit(2)
	}

	// Handle exit based on result
	if result.ShouldExit {
		os.Exit(result.ExitCode)
	}
}

type RunResult struct {
	Status     string
	ShouldExit bool
	ExitCode   int
}

func run(ctx context.Context, cfg signal.Config, executor signal.Executor, publisher signal.Publisher, imdsClient signal.IMDSClient) (*RunResult, error) {
	result := &RunResult{
		ShouldExit: false,
		ExitCode:   0,
	}

	// Determine status
	status := cfg.Status
	if status == "" {
		// Execute command and determine status from exit code
		exitCode, err := executor.Run(cfg.Exec)
		if err != nil {
			if cfg.Verbose {
				log.Printf("Command execution failed: %v", err)
			}
			status = "FAILURE"
		} else if exitCode == 0 {
			status = "SUCCESS"
		} else {
			status = "FAILURE"
		}

		// Mark that we should exit with code 1 for failures
		if status == "FAILURE" {
			result.ShouldExit = true
			result.ExitCode = 1
		}
	}

	result.Status = status

	// Get instance ID from IMDS
	instanceID, err := imdsClient.GetInstanceID(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to get instance ID: %w", err)
	}

	// Publish signal
	publishInput := signal.PublishInput{
		QueueURL:       cfg.QueueURL,
		SignalID:       cfg.ID,
		InstanceID:     instanceID,
		Status:         status,
		PublishTimeout: cfg.PublishTimeout,
	}

	if err := publisher.Publish(ctx, publishInput); err != nil {
		return result, fmt.Errorf("failed to publish signal: %w", err)
	}

	if cfg.Verbose {
		log.Printf("Successfully published signal: status=%s, signal_id=%s, instance_id=%s", status, cfg.ID, instanceID)
	}

	return result, nil
}
