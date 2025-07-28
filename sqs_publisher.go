package signal

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.uber.org/zap"
)

type SQSPublisher struct {
	Logger Logger
}

func NewSQSPublisher(logger Logger) *SQSPublisher {
	return &SQSPublisher{
		Logger: logger,
	}
}

func (p *SQSPublisher) Publish(ctx context.Context, input PublishInput) error {
	// Configure AWS SDK with custom retry settings
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(
				retry.NewStandard(),
				input.Retries+1, // +1 because AWS counts attempts, not retries
			)
		}),
	)
	if err != nil {
		return err
	}

	client := sqs.NewFromConfig(awsCfg)

	// Create context with publish timeout
	publishCtx, cancel := context.WithTimeout(ctx, input.PublishTimeout)
	defer cancel()

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

	result, err := client.SendMessage(publishCtx, sqsInput)
	if err != nil {
		p.Logger.Error("Failed to send SQS message",
			zap.Int("retries", input.Retries),
			zap.String("signal_id", input.SignalID),
			zap.String("instance_id", input.InstanceID),
			zap.Error(err))
		return err
	}

	p.Logger.Info("SQS message sent successfully",
		zap.String("message_id", *result.MessageId),
		zap.String("signal_id", input.SignalID),
		zap.String("instance_id", input.InstanceID),
		zap.String("status", input.Status))

	return nil
}
