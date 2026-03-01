//go:build integration

package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestBundleStore_UploadAndDownload(t *testing.T) {
	s := newTestStore(t)
	h := hashOf('1')
	want := []byte("bundle-bytes")
	mustNoErr(t, s.Upload(context.Background(), h, bytes.NewReader(want), int64(len(want))))
	rc, err := s.Download(context.Background(), h)
	mustNoErr(t, err)
	defer rc.Close()
	got, err := io.ReadAll(rc)
	mustNoErr(t, err)
	if string(got) != string(want) {
		t.Fatalf("download mismatch: got %q want %q", got, want)
	}
}

func TestBundleStore_Exists_True(t *testing.T) {
	s := newTestStore(t)
	h := hashOf('2')
	mustNoErr(t, s.Upload(context.Background(), h, bytes.NewReader([]byte("x")), 1))
	exists, err := s.Exists(context.Background(), h)
	mustNoErr(t, err)
	if !exists {
		t.Fatal("Exists() = false, want true")
	}
}

func TestBundleStore_Exists_False(t *testing.T) {
	s := newTestStore(t)
	exists, err := s.Exists(context.Background(), hashOf('3'))
	mustNoErr(t, err)
	if exists {
		t.Fatal("Exists() = true, want false")
	}
}

func TestBundleStore_UploadIdempotent(t *testing.T) {
	s := newTestStore(t)
	h, data := hashOf('4'), []byte("same-content")
	mustNoErr(t, s.Upload(context.Background(), h, bytes.NewReader(data), int64(len(data))))
	mustNoErr(t, s.Upload(context.Background(), h, bytes.NewReader(data), int64(len(data))))
}

func TestBundleStore_DownloadNotFound(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Download(context.Background(), hashOf('5')); err == nil {
		t.Fatal("Download() error = nil, want not found")
	}
}

func TestBundleStore_BucketAutoCreated(t *testing.T) {
	requireStorageEnv(t)
	cfg := testStorageConfig(uniqueBucket())
	_, err := NewS3BundleStore(cfg)
	mustNoErr(t, err)
	client, err := newRawS3Client(cfg)
	mustNoErr(t, err)
	_, err = client.HeadBucket(context.Background(), &s3.HeadBucketInput{Bucket: aws.String(cfg.Bucket)})
	mustNoErr(t, err)
}

func newTestStore(t *testing.T) *S3BundleStore {
	t.Helper()
	requireStorageEnv(t)
	s, err := NewS3BundleStore(testStorageConfig(uniqueBucket()))
	mustNoErr(t, err)
	return s
}

func requireStorageEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("S3_ENDPOINT") == "" {
		t.Skip("skipping integration test: S3_ENDPOINT not set")
	}
}

func testStorageConfig(bucket string) StorageConfig {
	region := os.Getenv("S3_REGION")
	if region == "" {
		region = "us-east-1"
	}
	fps, _ := strconv.ParseBool(envOrDefault("S3_FORCE_PATH_STYLE", "true"))
	return StorageConfig{
		Endpoint:       os.Getenv("S3_ENDPOINT"),
		Bucket:         bucket,
		AccessKey:      envOrDefault("S3_ACCESS_KEY", "minioadmin"),
		SecretKey:      envOrDefault("S3_SECRET_KEY", "minioadmin"),
		Region:         region,
		ForcePathStyle: fps,
	}
}

func envOrDefault(k, d string) string {
	v := os.Getenv(k)
	if v != "" {
		return v
	}
	return d
}
func hashOf(ch byte) string { return string(bytes.Repeat([]byte{ch}, 64)) }
func uniqueBucket() string  { return "ser-bundles-it-" + strconv.FormatInt(time.Now().UnixNano(), 36) }

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func newRawS3Client(cfg StorageConfig) (*s3.Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = cfg.ForcePathStyle
	}), nil
}
