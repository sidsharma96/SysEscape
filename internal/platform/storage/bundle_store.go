package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go"
)

const defaultBucket, defaultRegion = "ser-bundles", "us-east-1"

type BundleStore interface {
	Upload(ctx context.Context, hash string, data io.Reader, size int64) error
	Download(ctx context.Context, hash string) (io.ReadCloser, error)
	Exists(ctx context.Context, hash string) (bool, error)
}

type StorageConfig struct {
	Endpoint       string
	Bucket         string
	AccessKey      string
	SecretKey      string
	Region         string
	ForcePathStyle bool
}

type S3BundleStore struct {
	client *s3.Client
	bucket string
	region string
}

func NewS3BundleStore(cfg StorageConfig) (*S3BundleStore, error) {
	bucket := cfg.Bucket
	if bucket == "" {
		bucket = defaultBucket
	}
	region := cfg.Region
	if region == "" {
		region = defaultRegion
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.ForcePathStyle
	})

	store := &S3BundleStore{client: client, bucket: bucket, region: region}
	if err := store.createBucketIfNotExists(context.Background()); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *S3BundleStore) Upload(ctx context.Context, hash string, data io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(s.key(hash)),
		Body:          data,
		ContentLength: &size,
	})
	return err
}

func (s *S3BundleStore) Download(ctx context.Context, hash string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(hash)),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return out.Body, nil
}

func (s *S3BundleStore) Exists(ctx context.Context, hash string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(hash)),
	})
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3BundleStore) createBucketIfNotExists(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err == nil {
		return nil
	}
	if !isNotFound(err) {
		return err
	}

	in := &s3.CreateBucketInput{Bucket: aws.String(s.bucket)}
	if s.region != defaultRegion {
		in.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(s.region),
		}
	}
	_, err = s.client.CreateBucket(ctx, in)
	if err == nil {
		return nil
	}

	var owned *types.BucketAlreadyOwnedByYou
	var exists *types.BucketAlreadyExists
	if errors.As(err, &owned) || errors.As(err, &exists) {
		return nil
	}
	_, headErr := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if headErr == nil {
		return nil
	}
	return err
}

func (s *S3BundleStore) key(hash string) string {
	return fmt.Sprintf("bundles/%s.tar", hash)
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch strings.TrimSpace(apiErr.ErrorCode()) {
		case "NotFound", "NoSuchKey", "NoSuchBucket", "404":
			return true
		}
	}
	return false
}
