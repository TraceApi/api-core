/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-21
 * Change License: AGPL-3.0
 */

package s3

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type BlobStore struct {
	client *s3.Client
}

type Config struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
}

func NewBlobStore(ctx context.Context, cfg Config) (*BlobStore, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if cfg.Endpoint != "" {
					return aws.Endpoint{
						PartitionID:   "aws",
						URL:           cfg.Endpoint,
						SigningRegion: cfg.Region,
					}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			},
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &BlobStore{client: client}, nil
}

func (b *BlobStore) UploadJSON(ctx context.Context, bucket string, key string, data []byte) (string, error) {
	retentionDate := time.Now().AddDate(10, 0, 0)

	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:                    aws.String(bucket),
		Key:                       aws.String(key),
		Body:                      bytes.NewReader(data),
		ContentType:               aws.String("application/json"),
		ObjectLockMode:            types.ObjectLockModeGovernance,
		ObjectLockRetainUntilDate: &retentionDate,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	return fmt.Sprintf("s3://%s/%s", bucket, key), nil
}
