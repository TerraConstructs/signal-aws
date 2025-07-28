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
	PublishTimeout time.Duration
}

type Publisher interface {
	Publish(ctx context.Context, input PublishInput) error
}
