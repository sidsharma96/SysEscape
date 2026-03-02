package resolvers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	authservice "github.com/sidsharma96/SysEscape/internal/auth/service"
	authtransport "github.com/sidsharma96/SysEscape/internal/auth/transport"
	catalogrepo "github.com/sidsharma96/SysEscape/internal/catalog/repo"
	"github.com/sidsharma96/SysEscape/internal/graphql/generated"
	publishsvc "github.com/sidsharma96/SysEscape/internal/platform/publish"
	"github.com/sidsharma96/SysEscape/pkg/models"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type mockRoomRepo struct {
	listFn      func(ctx context.Context, filter catalogrepo.RoomFilter) ([]models.RoomWithLatestVersion, error)
	getBySlugFn func(ctx context.Context, slug string) (*models.RoomWithLatestVersion, error)
}

type mockPublishService struct {
	publishFn func(ctx context.Context, input publishsvc.PublishInput) (*models.RoomVersion, error)
}

func (m *mockPublishService) Publish(ctx context.Context, input publishsvc.PublishInput) (*models.RoomVersion, error) {
	if m.publishFn == nil {
		return nil, errors.New("publishFn not set")
	}
	return m.publishFn(ctx, input)
}

func (m *mockRoomRepo) List(ctx context.Context, filter catalogrepo.RoomFilter) ([]models.RoomWithLatestVersion, error) {
	if m.listFn == nil {
		return nil, nil
	}
	return m.listFn(ctx, filter)
}

func (m *mockRoomRepo) GetBySlug(ctx context.Context, slug string) (*models.RoomWithLatestVersion, error) {
	if m.getBySlugFn == nil {
		return nil, nil
	}
	return m.getBySlugFn(ctx, slug)
}

type mockUserRepo struct {
	getByIDFn func(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

func (m *mockUserRepo) UpsertByGitHubID(ctx context.Context, githubID int64, username, displayName string) (*models.User, error) {
	return nil, nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	if m.getByIDFn == nil {
		return nil, nil
	}
	return m.getByIDFn(ctx, userID)
}

type mockSessionRepo struct {
	getByIDFn func(ctx context.Context, sessionID uuid.UUID) (*models.Session, error)
}

func (m *mockSessionRepo) Create(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (*models.Session, error) {
	return nil, nil
}

func (m *mockSessionRepo) GetByID(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
	if m.getByIDFn == nil {
		return nil, nil
	}
	return m.getByIDFn(ctx, sessionID)
}

func (m *mockSessionRepo) Delete(ctx context.Context, sessionID uuid.UUID) error {
	return nil
}

func TestRoomsResolver_ReturnsAllRooms(t *testing.T) {
	rooms := []models.RoomWithLatestVersion{
		{Room: models.Room{ID: uuid.New(), Slug: "room-1", Title: "Room 1", District: "D1", Engine: "A", Difficulty: "L1", Description: "desc1"}},
		{Room: models.Room{ID: uuid.New(), Slug: "room-2", Title: "Room 2", District: "D2", Engine: "B", Difficulty: "L0", Description: "desc2"}},
		{Room: models.Room{ID: uuid.New(), Slug: "room-3", Title: "Room 3", District: "D3", Engine: "A", Difficulty: "L2", Description: "desc3"}},
	}

	r := &Resolver{CatalogRepo: &mockRoomRepo{listFn: func(ctx context.Context, filter catalogrepo.RoomFilter) ([]models.RoomWithLatestVersion, error) {
		if filter.Engine != nil || filter.Difficulty != nil || filter.District != nil {
			t.Fatalf("expected empty filter, got %+v", filter)
		}
		return rooms, nil
	}}}

	got, err := r.Query().Rooms(context.Background(), nil, nil, nil)
	if err != nil {
		t.Fatalf("Rooms() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(rooms) = %d, want 3", len(got))
	}
}

func TestRoomsResolver_FilterByEngine(t *testing.T) {
	r := &Resolver{CatalogRepo: &mockRoomRepo{listFn: func(ctx context.Context, filter catalogrepo.RoomFilter) ([]models.RoomWithLatestVersion, error) {
		if filter.Engine == nil || *filter.Engine != "A" {
			t.Fatalf("Engine filter = %v, want A", filter.Engine)
		}
		return []models.RoomWithLatestVersion{}, nil
	}}}

	engine := generated.RoomEngineA
	_, err := r.Query().Rooms(context.Background(), &engine, nil, nil)
	if err != nil {
		t.Fatalf("Rooms() error = %v", err)
	}
}

func TestRoomBySlugResolver_Found(t *testing.T) {
	roomID := uuid.New()
	verID := uuid.New()
	publishedAt := time.Now().UTC().Truncate(time.Second)

	room := &models.RoomWithLatestVersion{
		Room: models.Room{
			ID:          roomID,
			Slug:        "cache-thundering-herd",
			Title:       "Cache Thundering Herd",
			District:    "Caching District",
			Engine:      "A",
			Difficulty:  "L1",
			Description: "desc",
		},
		LatestVersion: &models.RoomVersion{
			ID:            verID,
			VersionNumber: 1,
			Status:        "PUBLISHED",
			Changelog:     "Initial",
			PublishedAt:   publishedAt,
		},
	}

	r := &Resolver{CatalogRepo: &mockRoomRepo{getBySlugFn: func(ctx context.Context, slug string) (*models.RoomWithLatestVersion, error) {
		if slug != "cache-thundering-herd" {
			t.Fatalf("slug = %q, want %q", slug, "cache-thundering-herd")
		}
		return room, nil
	}}}

	got, err := r.Query().RoomBySlug(context.Background(), "cache-thundering-herd")
	if err != nil {
		t.Fatalf("RoomBySlug() error = %v", err)
	}
	if got == nil {
		t.Fatal("RoomBySlug() returned nil")
	}
	if got.ID != roomID.String() {
		t.Fatalf("ID = %q, want %q", got.ID, roomID.String())
	}
	if got.Title != "Cache Thundering Herd" {
		t.Fatalf("Title = %q", got.Title)
	}
	if got.Engine != generated.RoomEngineA {
		t.Fatalf("Engine = %q, want %q", got.Engine, generated.RoomEngineA)
	}
	if got.Difficulty != generated.RoomDifficultyL1 {
		t.Fatalf("Difficulty = %q, want %q", got.Difficulty, generated.RoomDifficultyL1)
	}
	if got.LatestVersion == nil {
		t.Fatal("LatestVersion = nil")
	}
	if got.LatestVersion.Status != generated.RoomVersionStatusPublished {
		t.Fatalf("LatestVersion.Status = %q, want %q", got.LatestVersion.Status, generated.RoomVersionStatusPublished)
	}
}

func TestRoomBySlugResolver_NotFound(t *testing.T) {
	r := &Resolver{CatalogRepo: &mockRoomRepo{getBySlugFn: func(ctx context.Context, slug string) (*models.RoomWithLatestVersion, error) {
		return nil, nil
	}}}

	got, err := r.Query().RoomBySlug(context.Background(), "missing")
	if err != nil {
		t.Fatalf("RoomBySlug() error = %v", err)
	}
	if got != nil {
		t.Fatalf("RoomBySlug() = %+v, want nil", got)
	}
}

func TestViewerResolver_Authenticated(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()

	authSvc := authservice.NewAuthService(
		&mockUserRepo{getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: id, Role: "USER", GitHubUsername: "octocat"}, nil
		}},
		&mockSessionRepo{getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Session, error) {
			return &models.Session{ID: id, UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil
		}},
		&authservice.MockGitHubClient{},
		authservice.OAuthConfig{},
	)

	var ctx context.Context
	h := authtransport.SessionMiddleware(authSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	r := &Resolver{}
	got, err := r.Query().Viewer(ctx)
	if err != nil {
		t.Fatalf("Viewer() error = %v", err)
	}
	if got == nil {
		t.Fatal("Viewer() returned nil")
	}
	if got.UserID != userID.String() {
		t.Fatalf("UserID = %q, want %q", got.UserID, userID.String())
	}
	if got.Role != "USER" {
		t.Fatalf("Role = %q, want USER", got.Role)
	}
	if got.GithubUsername == nil || *got.GithubUsername != "octocat" {
		t.Fatalf("GithubUsername = %v, want octocat", got.GithubUsername)
	}
}

func TestViewerResolver_Unauthenticated(t *testing.T) {
	r := &Resolver{}
	got, err := r.Query().Viewer(context.Background())
	if err != nil {
		t.Fatalf("Viewer() error = %v", err)
	}
	if got != nil {
		t.Fatalf("Viewer() = %+v, want nil", got)
	}
}

func TestPublishResolver_RejectsNonAdmin(t *testing.T) {
	r := &Resolver{PublishService: &mockPublishService{}}
	ctx := authtransport.ContextWithUser(context.Background(), &models.User{ID: uuid.New(), Role: "USER"})
	_, err := r.Mutation().PublishRoomVersion(ctx, generated.PublishRoomVersionInput{
		ClientRequestID:  "req",
		RoomSlug:         "cache-stampede",
		Version:          1,
		Changelog:        "c",
		BundleHashSha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Activate:         false,
	})
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	var gqlErr *gqlerror.Error
	if !errors.As(err, &gqlErr) {
		t.Fatalf("error type = %T, want *gqlerror.Error", err)
	}
	if got, _ := gqlErr.Extensions["code"].(string); got != "FORBIDDEN" {
		t.Fatalf("error code = %q, want FORBIDDEN", got)
	}
}
