package signal

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSPublisher struct {
	Verbose bool
}

func NewSQSPublisher(verbose bool) *SQSPublisher {
	return &SQSPublisher{
		Verbose: verbose,
	}
}

func (p *SQSPublisher) Publish(ctx context.Context, input PublishInput) error {
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	client := sqs.NewFromConfig(awsCfg)

	// Create context with publish timeout
	publishCtx, cancel := context.WithTimeout(ctx, input.PublishTimeout)
	defer cancel()

	sqsInput := &sqs.SendMessageInput{
		QueueUrl:    aws.String(input.QueueURL),
		MessageBody: aws.String(""),
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
		return err
	}

	if p.Verbose {
		log.Printf("SQS message sent successfully, MessageId: %s", *result.MessageId)
	}

	return nil
}
