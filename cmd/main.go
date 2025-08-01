package main

import (
	"context"
	"fmt"
	"os"

	"github.com/terraconstructs/signal-aws"
	"go.uber.org/zap"
)

func main() {
	cfg, err := signal.ParseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	// Create logger based on config
	logger, err := signal.NewLogger(cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(2)
	}
	defer logger.Sync()

	// Set up overall timeout context
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Create component instances
	executor := signal.NewDefaultExecutor(logger)
	publisher := signal.NewSQSPublisher(logger)
	imdsClient := signal.NewDefaultIMDSClient()

	result, err := run(ctx, *cfg, executor, publisher, imdsClient, logger)
	if err != nil {
		logger.Error("Application error", zap.Error(err))
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

func run(ctx context.Context, cfg signal.Config, executor signal.Executor, publisher signal.Publisher, imdsClient signal.IMDSClient, logger signal.Logger) (*RunResult, error) {
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
			logger.Error("Command execution failed",
				zap.String("command", cfg.Exec),
				zap.Error(err),
				zap.String("signal_id", cfg.ID))
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

	// Get instance ID - use provided value or fetch from IMDS
	var instanceID string
	if cfg.InstanceID != "" {
		instanceID = cfg.InstanceID
		logger.Debug("Using provided instance ID", zap.String("instance_id", instanceID))
	} else {
		var err error
		instanceID, err = imdsClient.GetInstanceID(ctx)
		if err != nil {
			return result, fmt.Errorf("failed to get instance ID: %w", err)
		}
		logger.Debug("Fetched instance ID from IMDS", zap.String("instance_id", instanceID))
	}

	// Resolve region - use provided value, fallback to IMDS, then AWS config
	var region string
	if cfg.Region != "" {
		region = cfg.Region
		logger.Debug("Using provided region", zap.String("region", region))
	} else {
		// Try to get region from IMDS first
		var err error
		region, err = imdsClient.GetRegion(ctx)
		if err != nil {
			logger.Debug("Failed to get region from IMDS, falling back to AWS config", zap.Error(err))
			// Region will be empty, let AWS SDK handle default resolution
		} else {
			logger.Debug("Fetched region from IMDS", zap.String("region", region))
		}
	}

	// Publish signal
	publishInput := signal.PublishInput{
		QueueURL:       cfg.QueueURL,
		SignalID:       cfg.ID,
		InstanceID:     instanceID,
		Status:         status,
		Region:         region,
		PublishTimeout: cfg.PublishTimeout,
		Retries:        cfg.Retries,
	}

	if err := publisher.Publish(ctx, publishInput); err != nil {
		return result, fmt.Errorf("failed to publish signal: %w", err)
	}

	logger.Info("Successfully published signal",
		zap.String("status", status),
		zap.String("signal_id", cfg.ID),
		zap.String("instance_id", instanceID))

	return result, nil
}
