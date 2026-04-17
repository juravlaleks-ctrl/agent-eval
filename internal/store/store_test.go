package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGetSession(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	sess := Session{
		ID:               "sess-001",
		Agent:            "test-agent",
		Model:            "claude-sonnet-4",
		Effort:           "medium",
		Status:           "active",
		StartedAt:        time.Now().UTC().Truncate(time.Second),
		TasksTotal:       5,
		TasksPassed:      0,
		TasksFailed:      0,
		TasksUnsubmitted: 5,
	}

	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := s.GetSession(ctx, "sess-001")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("ID: got %q, want %q", got.ID, sess.ID)
	}
	if got.Agent != sess.Agent {
		t.Errorf("Agent: got %q, want %q", got.Agent, sess.Agent)
	}
	if got.Status != "active" {
		t.Errorf("Status: got %q, want active", got.Status)
	}
	if got.TasksTotal != 5 {
		t.Errorf("TasksTotal: got %d, want 5", got.TasksTotal)
	}
}

func TestGetSessionNotFound(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	_, err := s.GetSession(ctx, "no-such-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdateSessionStatus(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	sess := Session{
		ID: "sess-upd", Agent: "a", Model: "m", Effort: "low",
		Status: "active", StartedAt: time.Now().UTC(), TasksTotal: 1,
	}
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatal(err)
	}

	finished := time.Now().UTC().Truncate(time.Second)
	if err := s.UpdateSessionStatus(ctx, "sess-upd", "completed", finished); err != nil {
		t.Fatalf("UpdateSessionStatus: %v", err)
	}

	got, err := s.GetSession(ctx, "sess-upd")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "completed" {
		t.Errorf("Status: got %q, want completed", got.Status)
	}
	if !got.FinishedAt.Valid {
		t.Error("expected FinishedAt to be set")
	}
}

func TestIncrementWarning(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	sess := Session{
		ID: "sess-warn", Agent: "a", Model: "m", Effort: "low",
		Status: "active", StartedAt: time.Now().UTC(), TasksTotal: 3,
	}
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 3; i++ {
		count, err := s.IncrementWarning(ctx, "sess-warn")
		if err != nil {
			t.Fatalf("IncrementWarning iter %d: %v", i, err)
		}
		if count != i {
			t.Errorf("IncrementWarning iter %d: got %d, want %d", i, count, i)
		}
	}
}

func TestUpsertAnswerAndList(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	sess := Session{
		ID: "sess-ans", Agent: "a", Model: "m", Effort: "low",
		Status: "active", StartedAt: time.Now().UTC(), TasksTotal: 2,
	}
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatal(err)
	}

	a1 := Answer{
		SessionID:  "sess-ans",
		TaskID:     "task-1",
		Category:   "math",
		Difficulty: 1,
	}
	if err := s.UpsertAnswer(ctx, a1); err != nil {
		t.Fatalf("UpsertAnswer: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	a2 := Answer{
		SessionID:   "sess-ans",
		TaskID:      "task-2",
		Category:    "logic",
		Difficulty:  2,
		Answer:      sql.NullString{String: "42", Valid: true},
		Passed:      sql.NullBool{Bool: true, Valid: true},
		SubmittedAt: sql.NullTime{Time: now, Valid: true},
	}
	if err := s.UpsertAnswer(ctx, a2); err != nil {
		t.Fatalf("UpsertAnswer: %v", err)
	}

	answers, err := s.ListAnswers(ctx, "sess-ans")
	if err != nil {
		t.Fatalf("ListAnswers: %v", err)
	}
	if len(answers) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(answers))
	}

	// Verify upsert overwrites correctly
	a1updated := Answer{
		SessionID:   "sess-ans",
		TaskID:      "task-1",
		Category:    "math",
		Difficulty:  1,
		Answer:      sql.NullString{String: "wrong", Valid: true},
		Passed:      sql.NullBool{Bool: false, Valid: true},
		SubmittedAt: sql.NullTime{Time: now, Valid: true},
	}
	if err := s.UpsertAnswer(ctx, a1updated); err != nil {
		t.Fatalf("UpsertAnswer update: %v", err)
	}

	answers, err = s.ListAnswers(ctx, "sess-ans")
	if err != nil {
		t.Fatal(err)
	}
	// task-1 should now have passed=false
	var found bool
	for _, a := range answers {
		if a.TaskID == "task-1" {
			found = true
			if !a.Passed.Valid || a.Passed.Bool {
				t.Errorf("task-1: expected Passed=false after update, got valid=%v bool=%v", a.Passed.Valid, a.Passed.Bool)
			}
		}
	}
	if !found {
		t.Error("task-1 not found after upsert")
	}
}

func TestRecomputeCounts(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	sess := Session{
		ID: "sess-rc", Agent: "a", Model: "m", Effort: "low",
		Status: "active", StartedAt: time.Now().UTC(), TasksTotal: 3,
	}
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	answers := []Answer{
		{SessionID: "sess-rc", TaskID: "t1", Category: "c", Difficulty: 1,
			Answer: sql.NullString{String: "a", Valid: true},
			Passed: sql.NullBool{Bool: true, Valid: true},
			SubmittedAt: sql.NullTime{Time: now, Valid: true}},
		{SessionID: "sess-rc", TaskID: "t2", Category: "c", Difficulty: 1,
			Answer: sql.NullString{String: "b", Valid: true},
			Passed: sql.NullBool{Bool: false, Valid: true},
			SubmittedAt: sql.NullTime{Time: now, Valid: true}},
		{SessionID: "sess-rc", TaskID: "t3", Category: "c", Difficulty: 1},
	}
	for _, a := range answers {
		if err := s.UpsertAnswer(ctx, a); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.RecomputeCounts(ctx, "sess-rc"); err != nil {
		t.Fatalf("RecomputeCounts: %v", err)
	}

	got, err := s.GetSession(ctx, "sess-rc")
	if err != nil {
		t.Fatal(err)
	}
	if got.TasksPassed != 1 {
		t.Errorf("TasksPassed: got %d, want 1", got.TasksPassed)
	}
	if got.TasksFailed != 1 {
		t.Errorf("TasksFailed: got %d, want 1", got.TasksFailed)
	}
	if got.TasksUnsubmitted != 1 {
		t.Errorf("TasksUnsubmitted: got %d, want 1", got.TasksUnsubmitted)
	}
}

func TestListSessions(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	for i, id := range []string{"s1", "s2", "s3"} {
		effort := "low"
		if i == 2 {
			effort = "high"
		}
		sess := Session{
			ID: id, Agent: "agent-x", Model: "claude-sonnet-4", Effort: effort,
			Status: "active", StartedAt: time.Now().UTC(), TasksTotal: 1,
		}
		if err := s.CreateSession(ctx, sess); err != nil {
			t.Fatal(err)
		}
	}

	all, err := s.ListSessions(ctx, SessionFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(all))
	}

	filtered, err := s.ListSessions(ctx, SessionFilter{Effort: "low"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2 low-effort sessions, got %d", len(filtered))
	}

	limited, err := s.ListSessions(ctx, SessionFilter{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 1 {
		t.Errorf("expected 1 result with limit=1, got %d", len(limited))
	}
}
