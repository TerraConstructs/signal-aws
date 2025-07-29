package signal

import (
	"context"
	"time"
)

type PublishInput struct {
	QueueURL       string
	SignalID       string
	InstanceID     string
	Status         string
	Region         string
	PublishTimeout time.Duration
	Retries        int
}

type Publisher interface {
	Publish(ctx context.Context, input PublishInput) error
}
