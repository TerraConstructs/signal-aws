package signal

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

type IMDSClient interface {
	GetInstanceID(ctx context.Context) (string, error)
}

type DefaultIMDSClient struct{}

func NewDefaultIMDSClient() *DefaultIMDSClient {
	return &DefaultIMDSClient{}
}

func (i *DefaultIMDSClient) GetInstanceID(ctx context.Context) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}

	client := imds.NewFromConfig(cfg)

	result, err := client.GetMetadata(ctx, &imds.GetMetadataInput{
		Path: "instance-id",
	})
	if err != nil {
		return "", err
	}
	defer result.Content.Close()

	content, err := io.ReadAll(result.Content)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
