package repo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	enginerepo "github.com/sidsharma96/SysEscape/internal/engine/a/repo"
	"github.com/sidsharma96/SysEscape/internal/testutil"
	"github.com/sidsharma96/SysEscape/pkg/models"
)

func TestPostgresRunRepo_CreateRun_AndGetByID(t *testing.T) {
	ctx, tx, r := setupRepo(t)
	run := mustCreateRun(t, ctx, r, seedUser(t, ctx, tx), seedRoomVersion(t, ctx, tx), 42)
	if run.Status != models.RunStatusActive || run.CompletedAt != nil {
		t.Fatalf("unexpected run fields: %+v", run)
	}
	got, err := r.GetRunByID(ctx, run.ID)
	if err != nil || got == nil || got.ID != run.ID {
		t.Fatalf("GetRunByID() got=%+v err=%v", got, err)
	}
}

func TestPostgresRunRepo_GetByID_NotFound(t *testing.T) {
	ctx, _, r := setupRepo(t)
	got, err := r.GetRunByID(ctx, uuid.New())
	if err != nil || got != nil {
		t.Fatalf("GetRunByID() got=%+v err=%v", got, err)
	}
}

func TestPostgresRunRepo_AppendAction_AssignsMonotonicSeq(t *testing.T) {
	ctx, tx, r := setupRepo(t)
	runID := mustCreateRun(t, ctx, r, seedUser(t, ctx, tx), seedRoomVersion(t, ctx, tx), 7).ID
	k1, k2, id1, id2 := "enable_singleflight", "add_jitter_to_ttl", uuid.NewString(), uuid.NewString()
	a1 := mustAppend(t, ctx, r, enginerepo.AppendActionInput{RunID: runID, ActionType: models.RunActionTypePlayer, ActionKey: &k1, ClientRequestID: &id1})
	a2 := mustAppend(t, ctx, r, enginerepo.AppendActionInput{RunID: runID, ActionType: models.RunActionTypePlayer, ActionKey: &k2, ClientRequestID: &id2})
	if a1.Seq != 1 || a2.Seq != 2 {
		t.Fatalf("seqs=(%d,%d), want (1,2)", a1.Seq, a2.Seq)
	}
}

func TestPostgresRunRepo_AppendAction_DedupSameClientRequestIDReturnsExisting(t *testing.T) {
	ctx, tx, r := setupRepo(t)
	runID := mustCreateRun(t, ctx, r, seedUser(t, ctx, tx), seedRoomVersion(t, ctx, tx), 8).ID
	key, req := "enable_singleflight", uuid.NewString()
	in := enginerepo.AppendActionInput{RunID: runID, ActionType: models.RunActionTypePlayer, ActionKey: &key, ClientRequestID: &req}
	first := mustAppend(t, ctx, r, in)
	second := mustAppend(t, ctx, r, in)
	if first.ID != second.ID || first.Seq != second.Seq {
		t.Fatalf("expected idempotent action, first=%+v second=%+v", first, second)
	}
}

func TestPostgresRunRepo_AppendAction_DedupDifferentActionKeyReturnsConflict(t *testing.T) {
	ctx, tx, r := setupRepo(t)
	runID := mustCreateRun(t, ctx, r, seedUser(t, ctx, tx), seedRoomVersion(t, ctx, tx), 9).ID
	req, k1, k2 := uuid.NewString(), "enable_singleflight", "increase_cache_ttl"
	mustAppend(t, ctx, r, enginerepo.AppendActionInput{RunID: runID, ActionType: models.RunActionTypePlayer, ActionKey: &k1, ClientRequestID: &req})
	_, err := r.AppendAction(ctx, enginerepo.AppendActionInput{RunID: runID, ActionType: models.RunActionTypePlayer, ActionKey: &k2, ClientRequestID: &req})
	if !errors.Is(err, enginerepo.ErrActionConflict) {
		t.Fatalf("AppendAction() err=%v, want ErrActionConflict", err)
	}
}

func TestPostgresRunRepo_AppendAction_TickAllowsNilClientRequestIDAndNilActionKey(t *testing.T) {
	ctx, tx, r := setupRepo(t)
	runID := mustCreateRun(t, ctx, r, seedUser(t, ctx, tx), seedRoomVersion(t, ctx, tx), 10).ID
	a1 := mustAppend(t, ctx, r, enginerepo.AppendActionInput{RunID: runID, ActionType: models.RunActionTypeTick})
	a2 := mustAppend(t, ctx, r, enginerepo.AppendActionInput{RunID: runID, ActionType: models.RunActionTypeTick})
	if a1.Seq != 1 || a2.Seq != 2 || a1.ActionKey != nil || a1.ClientRequestID != nil {
		t.Fatalf("unexpected tick actions: a1=%+v a2=%+v", a1, a2)
	}
}

func TestPostgresRunRepo_ListActions_OrderedBySeqAsc(t *testing.T) {
	ctx, tx, r := setupRepo(t)
	runID := mustCreateRun(t, ctx, r, seedUser(t, ctx, tx), seedRoomVersion(t, ctx, tx), 11).ID
	k1, k2, id1, id2 := "enable_singleflight", "add_jitter_to_ttl", uuid.NewString(), uuid.NewString()
	mustAppend(t, ctx, r, enginerepo.AppendActionInput{RunID: runID, ActionType: models.RunActionTypePlayer, ActionKey: &k1, ClientRequestID: &id1})
	mustAppend(t, ctx, r, enginerepo.AppendActionInput{RunID: runID, ActionType: models.RunActionTypeTick})
	mustAppend(t, ctx, r, enginerepo.AppendActionInput{RunID: runID, ActionType: models.RunActionTypePlayer, ActionKey: &k2, ClientRequestID: &id2})
	actions, err := r.ListActions(ctx, runID)
	if err != nil || len(actions) != 3 || actions[0].Seq != 1 || actions[1].Seq != 2 || actions[2].Seq != 3 {
		t.Fatalf("ListActions() actions=%+v err=%v", actions, err)
	}
}

func setupRepo(t *testing.T) (context.Context, pgx.Tx, *enginerepo.PostgresRunRepo) {
	t.Helper()
	pool := testutil.TestPool(t)
	tx := testutil.TxForTest(t, pool)
	return context.Background(), tx, enginerepo.NewPostgresRunRepo(tx)
}

func mustCreateRun(t *testing.T, ctx context.Context, r *enginerepo.PostgresRunRepo, userID, roomVersionID uuid.UUID, seed int64) *models.Run {
	t.Helper()
	run, err := r.CreateRun(ctx, userID, roomVersionID, seed)
	if err != nil {
		t.Fatalf("CreateRun() err=%v", err)
	}
	return run
}

func mustAppend(t *testing.T, ctx context.Context, r *enginerepo.PostgresRunRepo, in enginerepo.AppendActionInput) *models.RunAction {
	t.Helper()
	action, err := r.AppendAction(ctx, in)
	if err != nil {
		t.Fatalf("AppendAction() err=%v", err)
	}
	return action
}

func seedUser(t *testing.T, ctx context.Context, tx pgx.Tx) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := tx.Exec(ctx, `INSERT INTO users (id, github_id, github_username, display_name, role) VALUES ($1, $2, $3, $4, 'USER')`, id, int64(1000000), "run-user", "Run User")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

func seedRoomVersion(t *testing.T, ctx context.Context, tx pgx.Tx) uuid.UUID {
	t.Helper()
	var roomID, versionID uuid.UUID
	err := tx.QueryRow(ctx, `INSERT INTO rooms (slug, title, district, engine, difficulty, description) VALUES ($1, 'Run Room', 'distributed-systems', 'A', 'L1', 'desc') RETURNING id`, "run-room-"+uuid.NewString()).Scan(&roomID)
	if err != nil {
		t.Fatalf("seed room: %v", err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO room_versions (room_id, version_number, status, changelog) VALUES ($1, 1, 'PUBLISHED', 'initial') RETURNING id`, roomID).Scan(&versionID)
	if err != nil {
		t.Fatalf("seed room_version: %v", err)
	}
	return versionID
}
