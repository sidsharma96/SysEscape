package repo_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sidsharma96/SysEscape/internal/catalog/repo"
	"github.com/sidsharma96/SysEscape/internal/testutil"
)

func TestPostgresRoomRepo_List_NoFilter(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresRoomRepo(tx)

	rooms, err := r.List(context.Background(), repo.RoomFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// Seed data has 3 rooms.
	if len(rooms) < 3 {
		t.Errorf("List() returned %d rooms, want at least 3", len(rooms))
	}

	for _, room := range rooms {
		if room.Slug == "" {
			t.Error("expected non-empty slug")
		}
		if room.LatestVersion == nil {
			t.Errorf("room %q: expected LatestVersion, got nil", room.Slug)
		}
	}
}

func TestPostgresRoomRepo_List_FilterByEngine(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresRoomRepo(tx)

	engineA := "A"
	rooms, err := r.List(context.Background(), repo.RoomFilter{Engine: &engineA})
	if err != nil {
		t.Fatalf("List(engine=A) error = %v", err)
	}

	if len(rooms) == 0 {
		t.Fatal("expected at least one Engine A room")
	}

	for _, room := range rooms {
		if room.Engine != "A" {
			t.Errorf("room %q: Engine = %q, want %q", room.Slug, room.Engine, "A")
		}
	}
}

func TestPostgresRoomRepo_List_FilterByDifficulty(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresRoomRepo(tx)

	diff := "L1"
	rooms, err := r.List(context.Background(), repo.RoomFilter{Difficulty: &diff})
	if err != nil {
		t.Fatalf("List(difficulty=L1) error = %v", err)
	}

	if len(rooms) == 0 {
		t.Fatal("expected at least one L1 room")
	}

	for _, room := range rooms {
		if room.Difficulty != "L1" {
			t.Errorf("room %q: Difficulty = %q, want %q", room.Slug, room.Difficulty, "L1")
		}
	}
}

func TestPostgresRoomRepo_List_FilterByDistrict(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresRoomRepo(tx)

	district := "Networking District"
	rooms, err := r.List(context.Background(), repo.RoomFilter{District: &district})
	if err != nil {
		t.Fatalf("List(district=Networking District) error = %v", err)
	}

	if len(rooms) == 0 {
		t.Fatal("expected at least one Networking District room")
	}

	for _, room := range rooms {
		if room.District != "Networking District" {
			t.Errorf("room %q: District = %q, want %q", room.Slug, room.District, "Networking District")
		}
	}
}

func TestPostgresRoomRepo_List_MultipleFilters(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresRoomRepo(tx)

	engineA := "A"
	diff := "L1"
	rooms, err := r.List(context.Background(), repo.RoomFilter{Engine: &engineA, Difficulty: &diff})
	if err != nil {
		t.Fatalf("List(engine=A, difficulty=L1) error = %v", err)
	}

	if len(rooms) == 0 {
		t.Fatal("expected at least one room matching engine=A AND difficulty=L1")
	}

	for _, room := range rooms {
		if room.Engine != "A" {
			t.Errorf("room %q: Engine = %q, want %q", room.Slug, room.Engine, "A")
		}
		if room.Difficulty != "L1" {
			t.Errorf("room %q: Difficulty = %q, want %q", room.Slug, room.Difficulty, "L1")
		}
	}
}

func TestPostgresRoomRepo_GetBySlug_Found(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresRoomRepo(tx)

	room, err := r.GetBySlug(context.Background(), "cache-thundering-herd")
	if err != nil {
		t.Fatalf("GetBySlug() error = %v", err)
	}
	if room == nil {
		t.Fatal("expected room, got nil")
	}
	if room.Title != "Cache Thundering Herd" {
		t.Errorf("Title = %q, want %q", room.Title, "Cache Thundering Herd")
	}
	if room.Engine != "A" {
		t.Errorf("Engine = %q, want %q", room.Engine, "A")
	}
	if room.Difficulty != "L1" {
		t.Errorf("Difficulty = %q, want %q", room.Difficulty, "L1")
	}
	if room.LatestVersion == nil {
		t.Fatal("expected LatestVersion, got nil")
	}
	if room.LatestVersion.VersionNumber != 1 {
		t.Errorf("LatestVersion.VersionNumber = %d, want 1", room.LatestVersion.VersionNumber)
	}
}

func TestPostgresRoomRepo_GetBySlug_NoPublishedVersion(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresRoomRepo(tx)

	ctx := context.Background()

	// Insert a room with no versions at all.
	_, err := tx.Exec(ctx, `
		INSERT INTO rooms (slug, title, district, engine, difficulty, description)
		VALUES ('no-version-room', 'No Version Room', 'Test District', 'A', 'L0', 'Room without any version')`)
	if err != nil {
		t.Fatalf("inserting room: %v", err)
	}

	room, err := r.GetBySlug(ctx, "no-version-room")
	if err != nil {
		t.Fatalf("GetBySlug() error = %v", err)
	}
	if room == nil {
		t.Fatal("expected room, got nil")
	}
	if room.Title != "No Version Room" {
		t.Errorf("Title = %q, want %q", room.Title, "No Version Room")
	}
	if room.LatestVersion != nil {
		t.Errorf("expected nil LatestVersion, got %+v", room.LatestVersion)
	}
}

func TestPostgresRoomRepo_GetBySlug_NotFound(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresRoomRepo(tx)

	room, err := r.GetBySlug(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetBySlug() error = %v", err)
	}
	if room != nil {
		t.Errorf("expected nil for nonexistent slug, got %+v", room)
	}
}

func TestRoomRepo_ScanIncludesActiveVersionID(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresRoomRepo(tx)

	ctx := context.Background()

	var roomID uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO rooms (slug, title, district, engine, difficulty, description)
		VALUES ('active-version-room', 'Active Version Room', 'Test District', 'A', 'L1', 'Room with active version')
		RETURNING id`,
	).Scan(&roomID)
	if err != nil {
		t.Fatalf("inserting room: %v", err)
	}

	var versionID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO room_versions (room_id, version_number, status, changelog)
		VALUES ($1, 1, 'PUBLISHED', 'Initial release')
		RETURNING id`,
		roomID,
	).Scan(&versionID)
	if err != nil {
		t.Fatalf("inserting room version: %v", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE rooms
		SET active_room_version_id = $1
		WHERE id = $2`,
		versionID, roomID,
	)
	if err != nil {
		t.Fatalf("updating active_room_version_id: %v", err)
	}

	room, err := r.GetBySlug(ctx, "active-version-room")
	if err != nil {
		t.Fatalf("GetBySlug() error = %v", err)
	}
	if room == nil {
		t.Fatal("expected room, got nil")
	}
	if room.ActiveRoomVersionID == nil {
		t.Fatal("expected ActiveRoomVersionID, got nil")
	}
	if *room.ActiveRoomVersionID != versionID {
		t.Fatalf("ActiveRoomVersionID = %v, want %v", *room.ActiveRoomVersionID, versionID)
	}
}

func TestRoomRepo_NullActiveVersionID(t *testing.T) {
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	r := repo.NewPostgresRoomRepo(tx)

	ctx := context.Background()

	_, err := tx.Exec(ctx, `
		INSERT INTO rooms (slug, title, district, engine, difficulty, description)
		VALUES ('null-active-version-room', 'Null Active Version Room', 'Test District', 'A', 'L0', 'Room without active version')`,
	)
	if err != nil {
		t.Fatalf("inserting room: %v", err)
	}

	room, err := r.GetBySlug(ctx, "null-active-version-room")
	if err != nil {
		t.Fatalf("GetBySlug() error = %v", err)
	}
	if room == nil {
		t.Fatal("expected room, got nil")
	}
	if room.ActiveRoomVersionID != nil {
		t.Fatalf("expected nil ActiveRoomVersionID, got %v", *room.ActiveRoomVersionID)
	}
}
