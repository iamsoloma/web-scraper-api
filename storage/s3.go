package storage

import (
	"bytes"
	"context"
	"io"
	"web-scraper-api/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Storage struct {
	Config config.Storage
	client *s3.Client
}

func NewStorage(config config.Storage) *Storage {
	cfg := aws.NewConfig()
	cfg.BaseEndpoint = &config.Endpoint
	cfg.Region = config.Region
	cfg.Credentials = aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
		return aws.Credentials{
			AccessKeyID:     config.AccessKeyID,
			SecretAccessKey: config.SecretAccessKey,
		}, nil
	})

	client := s3.NewFromConfig(*cfg)
	return &Storage{Config: config, client: client}
}

func (s *Storage) PutObject(key string, data []byte) error {
	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: &s.Config.Bucket,
		Key:    &key,
		Body:   bytes.NewReader(data),
	})
	return err
}

func (s *Storage) GetObject(key string) ([]byte, error) {
	output, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &s.Config.Bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	defer output.Body.Close()

	return io.ReadAll(output.Body)
}
