package roomctl

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sidsharma96/SysEscape/internal/platform/storage"
)

func TestPublishCmd_ValidatesBeforeBuilding(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	roomDir := filepath.Join("rooms", "cache-stampede")
	mustMkdir(t, filepath.Join(roomDir, "engineA"))

	_, err := PublishRoom(context.Background(), PublishOptions{
		RoomDir:          roomDir,
		Version:          1,
		BFFURL:           "http://localhost:8080/graphql",
		S3Endpoint:       "http://localhost:9000",
		S3Bucket:         "ser-bundles",
		S3AccessKey:      "minioadmin",
		S3SecretKey:      "minioadmin",
		S3Region:         "us-east-1",
		S3ForcePathStyle: true,
		AdminAPIKey:      "test-admin-key",
	})
	if err == nil {
		t.Fatal("PublishRoom() error = nil, want validation failure")
	}
	if !strings.Contains(err.Error(), "metadata.yaml") {
		t.Fatalf("expected metadata validation error, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(".build", "cache-stampede")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no build output, stat err = %v", statErr)
	}
}

func TestPublishCmd_FullPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full pipeline test in short mode")
	}

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	adminKey := strings.TrimSpace(os.Getenv("SER_ADMIN_API_KEY"))
	if adminKey == "" {
		adminKey = strings.TrimSpace(os.Getenv("ADMIN_API_KEY"))
	}
	if adminKey == "" {
		t.Skip("SER_ADMIN_API_KEY or ADMIN_API_KEY not set")
	}
	if strings.TrimSpace(os.Getenv("S3_ENDPOINT")) == "" {
		t.Skip("S3_ENDPOINT not set")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repoRoot: %v", err)
	}
	roomDir := filepath.Join(repoRoot, "rooms", "cache-stampede")

	_, err = pool.Exec(context.Background(), `
		INSERT INTO rooms (slug, title, district, engine, difficulty, description)
		VALUES ('cache-stampede', 'Cache Stampede', 'distributed-systems', 'A', 'L1', 'smoke publish room')
		ON CONFLICT (slug) DO NOTHING`)
	if err != nil {
		t.Fatalf("seed room: %v", err)
	}

	_, err = pool.Exec(context.Background(), `
		UPDATE rooms SET active_room_version_id = NULL WHERE slug = 'cache-stampede';
		DELETE FROM room_versions WHERE room_id = (SELECT id FROM rooms WHERE slug = 'cache-stampede')`)
	if err != nil {
		t.Fatalf("cleanup versions: %v", err)
	}

	res, err := PublishRoom(context.Background(), PublishOptions{
		RoomDir:          roomDir,
		Version:          1,
		Activate:         true,
		BFFURL:           strings.TrimSpace(os.Getenv("SER_BFF_URL")),
		S3Endpoint:       strings.TrimSpace(os.Getenv("S3_ENDPOINT")),
		S3Bucket:         strings.TrimSpace(os.Getenv("S3_BUCKET")),
		S3AccessKey:      strings.TrimSpace(os.Getenv("S3_ACCESS_KEY")),
		S3SecretKey:      strings.TrimSpace(os.Getenv("S3_SECRET_KEY")),
		S3Region:         strings.TrimSpace(os.Getenv("S3_REGION")),
		S3ForcePathStyle: true,
		AdminAPIKey:      adminKey,
	})
	if err != nil {
		t.Fatalf("PublishRoom() error = %v", err)
	}

	var versionID uuid.UUID
	var versionNumber int
	var status string
	var bundleHash string
	err = pool.QueryRow(context.Background(), `
		SELECT rv.id, rv.version_number, rv.status, COALESCE(rv.bundle_hash, '')
		FROM room_versions rv
		JOIN rooms r ON r.id = rv.room_id
		WHERE r.slug = 'cache-stampede' AND rv.version_number = 1`).
		Scan(&versionID, &versionNumber, &status, &bundleHash)
	if err != nil {
		t.Fatalf("query room_version: %v", err)
	}
	if versionNumber != 1 || status != "PUBLISHED" || bundleHash == "" {
		t.Fatalf("unexpected room_version row: version=%d status=%s bundle_hash=%q", versionNumber, status, bundleHash)
	}

	var activeID *uuid.UUID
	err = pool.QueryRow(context.Background(), `SELECT active_room_version_id FROM rooms WHERE slug = 'cache-stampede'`).Scan(&activeID)
	if err != nil {
		t.Fatalf("query active pointer: %v", err)
	}
	if activeID == nil || *activeID != versionID {
		t.Fatalf("active_room_version_id = %v, want %s", activeID, versionID)
	}

	store, err := storage.NewS3BundleStore(storage.StorageConfig{
		Endpoint:       strings.TrimSpace(os.Getenv("S3_ENDPOINT")),
		Bucket:         strings.TrimSpace(os.Getenv("S3_BUCKET")),
		AccessKey:      strings.TrimSpace(os.Getenv("S3_ACCESS_KEY")),
		SecretKey:      strings.TrimSpace(os.Getenv("S3_SECRET_KEY")),
		Region:         strings.TrimSpace(os.Getenv("S3_REGION")),
		ForcePathStyle: true,
	})
	if err != nil {
		t.Fatalf("NewS3BundleStore: %v", err)
	}
	ok, err := store.Exists(context.Background(), bundleHash)
	if err != nil {
		t.Fatalf("store.Exists: %v", err)
	}
	if !ok {
		t.Fatalf("expected bundle hash %s in object store", bundleHash)
	}
	if res.RoomVersionID != versionID.String() {
		t.Fatalf("PublishResult.RoomVersionID = %s, want %s", res.RoomVersionID, versionID)
	}
}
