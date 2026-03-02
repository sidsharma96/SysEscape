package roomctl

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/sidsharma96/SysEscape/internal/platform/storage"
)

const defaultPublishChangelog = "Published by roomctl"

type PublishOptions struct {
	RoomDir          string
	Version          int
	Activate         bool
	BFFURL           string
	S3Endpoint       string
	S3Bucket         string
	S3AccessKey      string
	S3SecretKey      string
	S3Region         string
	S3ForcePathStyle bool
	AdminAPIKey      string
	Changelog        string
}

type PublishResult struct {
	Slug          string
	Version       int
	Hash          string
	Activated     bool
	RoomVersionID string
}

func PublishRoom(ctx context.Context, opts PublishOptions) (*PublishResult, error) {
	opts = withPublishDefaults(opts)
	if err := validatePublishOptions(opts); err != nil {
		return nil, err
	}

	if err := ValidateRoomDir(opts.RoomDir); err != nil {
		return nil, err
	}

	buildRes, err := BuildRoom(opts.RoomDir, opts.Version)
	if err != nil {
		return nil, err
	}
	bundle, err := os.ReadFile(buildRes.BundlePath)
	if err != nil {
		return nil, err
	}

	store, err := storage.NewS3BundleStore(storage.StorageConfig{
		Endpoint:       opts.S3Endpoint,
		Bucket:         opts.S3Bucket,
		AccessKey:      opts.S3AccessKey,
		SecretKey:      opts.S3SecretKey,
		Region:         opts.S3Region,
		ForcePathStyle: opts.S3ForcePathStyle,
	})
	if err != nil {
		return nil, err
	}

	hash := buildRes.Manifest.BundleHashSha256
	if err := store.Upload(ctx, hash, bytes.NewReader(bundle), int64(len(bundle))); err != nil {
		return nil, err
	}

	client := NewGraphQLClient(opts.BFFURL, opts.AdminAPIKey)
	published, err := client.PublishRoomVersion(ctx, PublishMutationInput{
		ClientRequestID:  uuid.NewString(),
		RoomSlug:         buildRes.Manifest.Slug,
		Version:          opts.Version,
		Changelog:        opts.Changelog,
		BundleHashSha256: hash,
		Activate:         opts.Activate,
	})
	if err != nil {
		return nil, err
	}

	return &PublishResult{
		Slug:          buildRes.Manifest.Slug,
		Version:       opts.Version,
		Hash:          hash,
		Activated:     opts.Activate,
		RoomVersionID: published.ID,
	}, nil
}

func withPublishDefaults(opts PublishOptions) PublishOptions {
	if strings.TrimSpace(opts.BFFURL) == "" {
		opts.BFFURL = envOrDefault("SER_BFF_URL", "http://localhost:8080/graphql")
	}
	if strings.TrimSpace(opts.S3Endpoint) == "" {
		opts.S3Endpoint = envOrDefault("S3_ENDPOINT", "http://localhost:9000")
	}
	if strings.TrimSpace(opts.S3Bucket) == "" {
		opts.S3Bucket = envOrDefault("S3_BUCKET", "ser-bundles")
	}
	if strings.TrimSpace(opts.S3AccessKey) == "" {
		opts.S3AccessKey = envOrDefault("S3_ACCESS_KEY", "minioadmin")
	}
	if strings.TrimSpace(opts.S3SecretKey) == "" {
		opts.S3SecretKey = envOrDefault("S3_SECRET_KEY", "minioadmin")
	}
	if strings.TrimSpace(opts.S3Region) == "" {
		opts.S3Region = envOrDefault("S3_REGION", "us-east-1")
	}
	if !opts.S3ForcePathStyle {
		opts.S3ForcePathStyle = boolEnvOrDefault("S3_FORCE_PATH_STYLE", true)
	}
	if strings.TrimSpace(opts.AdminAPIKey) == "" {
		opts.AdminAPIKey = strings.TrimSpace(os.Getenv("SER_ADMIN_API_KEY"))
		if opts.AdminAPIKey == "" {
			opts.AdminAPIKey = strings.TrimSpace(os.Getenv("ADMIN_API_KEY"))
		}
	}
	if strings.TrimSpace(opts.Changelog) == "" {
		opts.Changelog = defaultPublishChangelog
	}
	return opts
}

func validatePublishOptions(opts PublishOptions) error {
	switch {
	case strings.TrimSpace(opts.RoomDir) == "":
		return fmt.Errorf("room directory is required")
	case opts.Version < 1:
		return fmt.Errorf("version must be a positive integer")
	case strings.TrimSpace(opts.BFFURL) == "":
		return fmt.Errorf("bff url is required")
	case strings.TrimSpace(opts.AdminAPIKey) == "":
		return fmt.Errorf("admin api key is required")
	case strings.TrimSpace(opts.S3Endpoint) == "":
		return fmt.Errorf("s3 endpoint is required")
	case strings.TrimSpace(opts.S3Bucket) == "":
		return fmt.Errorf("s3 bucket is required")
	case strings.TrimSpace(opts.S3AccessKey) == "":
		return fmt.Errorf("s3 access key is required")
	case strings.TrimSpace(opts.S3SecretKey) == "":
		return fmt.Errorf("s3 secret key is required")
	case strings.TrimSpace(opts.S3Region) == "":
		return fmt.Errorf("s3 region is required")
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func boolEnvOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
