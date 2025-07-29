package signal

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

type IMDSClient interface {
	GetInstanceID(ctx context.Context) (string, error)
	GetRegion(ctx context.Context) (string, error)
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

	result, err := client.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return "", err
	}

	return result.InstanceIdentityDocument.InstanceID, nil
}

func (i *DefaultIMDSClient) GetRegion(ctx context.Context) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}

	client := imds.NewFromConfig(cfg)

	result, err := client.GetRegion(ctx, &imds.GetRegionInput{})
	if err != nil {
		return "", err
	}

	return result.Region, nil
}
