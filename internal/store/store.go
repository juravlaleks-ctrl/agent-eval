package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database and provides typed access to sessions and answers.
type Store struct {
	db *sql.DB
}

// Session represents a row in the sessions table.
type Session struct {
	ID               string
	Agent            string
	Model            string
	Effort           string
	GroupID          sql.NullString
	Notes            sql.NullString
	Status           string
	StartedAt        time.Time
	FinishedAt       sql.NullTime
	TasksTotal       int
	TasksPassed      int
	TasksFailed      int
	TasksUnsubmitted int
	Warnings         int
}

// Answer represents a row in the answers table.
type Answer struct {
	SessionID   string
	TaskID      string
	Category    string
	Difficulty  int
	Answer      sql.NullString
	Passed      sql.NullBool
	SubmittedAt sql.NullTime
}

// SessionFilter restricts which sessions are returned by ListSessions.
type SessionFilter struct {
	Agent  string
	Model  string
	Effort string
	Limit  int
}

// Open opens (or creates) the SQLite database at path, applies the schema, and
// returns a Store ready for use. Use ":memory:" for an in-memory database.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open %q: %w", path, err)
	}

	// SQLite is not safe for concurrent writes; one connection avoids locking.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: apply schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// CreateSession inserts a new session row.
func (s *Store) CreateSession(ctx context.Context, sess Session) error {
	const q = `
		INSERT INTO sessions
		    (id, agent, model, effort, group_id, notes, status,
		     started_at, finished_at, tasks_total, tasks_passed,
		     tasks_failed, tasks_unsubmitted, warnings)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var finishedAt interface{}
	if sess.FinishedAt.Valid {
		finishedAt = sess.FinishedAt.Time.UTC().Format(time.RFC3339)
	}

	_, err := s.db.ExecContext(ctx, q,
		sess.ID,
		sess.Agent,
		sess.Model,
		sess.Effort,
		sess.GroupID,
		sess.Notes,
		sess.Status,
		sess.StartedAt.UTC().Format(time.RFC3339),
		finishedAt,
		sess.TasksTotal,
		sess.TasksPassed,
		sess.TasksFailed,
		sess.TasksUnsubmitted,
		sess.Warnings,
	)
	if err != nil {
		return fmt.Errorf("store: create session %q: %w", sess.ID, err)
	}
	return nil
}

// GetSession retrieves a session by its ID. Returns sql.ErrNoRows if not found.
func (s *Store) GetSession(ctx context.Context, id string) (*Session, error) {
	const q = `
		SELECT id, agent, model, effort, group_id, notes, status,
		       started_at, finished_at, tasks_total, tasks_passed,
		       tasks_failed, tasks_unsubmitted, warnings
		FROM sessions WHERE id = ?`

	row := s.db.QueryRowContext(ctx, q, id)

	var sess Session
	var startedAt string
	var finishedAt sql.NullString
	err := row.Scan(
		&sess.ID, &sess.Agent, &sess.Model, &sess.Effort,
		&sess.GroupID, &sess.Notes, &sess.Status,
		&startedAt, &finishedAt,
		&sess.TasksTotal, &sess.TasksPassed, &sess.TasksFailed,
		&sess.TasksUnsubmitted, &sess.Warnings,
	)
	if err != nil {
		return nil, err
	}

	sess.StartedAt, err = time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return nil, fmt.Errorf("store: parse started_at %q: %w", startedAt, err)
	}
	if finishedAt.Valid {
		t, err := time.Parse(time.RFC3339, finishedAt.String)
		if err != nil {
			return nil, fmt.Errorf("store: parse finished_at %q: %w", finishedAt.String, err)
		}
		sess.FinishedAt = sql.NullTime{Time: t, Valid: true}
	}
	return &sess, nil
}

// UpdateSessionStatus sets status and finished_at on a session.
func (s *Store) UpdateSessionStatus(ctx context.Context, id, status string, finished time.Time) error {
	const q = `UPDATE sessions SET status = ?, finished_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, q, status, finished.UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("store: update session status %q: %w", id, err)
	}
	return nil
}

// IncrementWarning increments the warnings counter for a session and returns
// the new count.
func (s *Store) IncrementWarning(ctx context.Context, id string) (int, error) {
	const q = `UPDATE sessions SET warnings = warnings + 1 WHERE id = ? RETURNING warnings`
	var count int
	err := s.db.QueryRowContext(ctx, q, id).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: increment warning for session %q: %w", id, err)
	}
	return count, nil
}

// UpsertAnswer inserts or replaces an answer row.
func (s *Store) UpsertAnswer(ctx context.Context, a Answer) error {
	const q = `
		INSERT INTO answers
		    (session_id, task_id, category, difficulty, answer, passed, submitted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id, task_id) DO UPDATE SET
		    answer       = excluded.answer,
		    passed       = excluded.passed,
		    submitted_at = excluded.submitted_at`

	var passedVal interface{}
	if a.Passed.Valid {
		if a.Passed.Bool {
			passedVal = 1
		} else {
			passedVal = 0
		}
	}

	var submittedAt interface{}
	if a.SubmittedAt.Valid {
		submittedAt = a.SubmittedAt.Time.UTC().Format(time.RFC3339)
	}

	_, err := s.db.ExecContext(ctx, q,
		a.SessionID,
		a.TaskID,
		a.Category,
		a.Difficulty,
		a.Answer,
		passedVal,
		submittedAt,
	)
	if err != nil {
		return fmt.Errorf("store: upsert answer session=%q task=%q: %w", a.SessionID, a.TaskID, err)
	}
	return nil
}

// ListAnswers returns all answers for a session, ordered by task_id.
func (s *Store) ListAnswers(ctx context.Context, sessionID string) ([]Answer, error) {
	const q = `
		SELECT session_id, task_id, category, difficulty, answer, passed, submitted_at
		FROM answers WHERE session_id = ?
		ORDER BY task_id`

	rows, err := s.db.QueryContext(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("store: list answers for session %q: %w", sessionID, err)
	}
	defer rows.Close()

	var answers []Answer
	for rows.Next() {
		var a Answer
		var passedInt sql.NullInt64
		var submittedAt sql.NullString
		if err := rows.Scan(
			&a.SessionID, &a.TaskID, &a.Category, &a.Difficulty,
			&a.Answer, &passedInt, &submittedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan answer: %w", err)
		}
		if passedInt.Valid {
			a.Passed = sql.NullBool{Bool: passedInt.Int64 != 0, Valid: true}
		}
		if submittedAt.Valid {
			t, err := time.Parse(time.RFC3339, submittedAt.String)
			if err != nil {
				return nil, fmt.Errorf("store: parse submitted_at %q: %w", submittedAt.String, err)
			}
			a.SubmittedAt = sql.NullTime{Time: t, Valid: true}
		}
		answers = append(answers, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate answers: %w", err)
	}
	return answers, nil
}

// RecomputeCounts recalculates tasks_passed, tasks_failed, and
// tasks_unsubmitted for a session by aggregating over the answers table.
func (s *Store) RecomputeCounts(ctx context.Context, sessionID string) error {
	const q = `
		UPDATE sessions SET
		    tasks_passed      = (SELECT COUNT(*) FROM answers WHERE session_id = ? AND passed = 1),
		    tasks_failed      = (SELECT COUNT(*) FROM answers WHERE session_id = ? AND passed = 0),
		    tasks_unsubmitted = (SELECT COUNT(*) FROM answers WHERE session_id = ? AND passed IS NULL)
		WHERE id = ?`

	_, err := s.db.ExecContext(ctx, q, sessionID, sessionID, sessionID, sessionID)
	if err != nil {
		return fmt.Errorf("store: recompute counts for session %q: %w", sessionID, err)
	}
	return nil
}

// ListSessions returns sessions matching the filter. All filter fields are
// optional; zero values are ignored.
func (s *Store) ListSessions(ctx context.Context, filter SessionFilter) ([]Session, error) {
	query := `
		SELECT id, agent, model, effort, group_id, notes, status,
		       started_at, finished_at, tasks_total, tasks_passed,
		       tasks_failed, tasks_unsubmitted, warnings
		FROM sessions WHERE 1=1`
	var args []interface{}

	if filter.Agent != "" {
		query += " AND agent = ?"
		args = append(args, filter.Agent)
	}
	if filter.Model != "" {
		query += " AND model = ?"
		args = append(args, filter.Model)
	}
	if filter.Effort != "" {
		query += " AND effort = ?"
		args = append(args, filter.Effort)
	}
	query += " ORDER BY started_at DESC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var startedAt string
		var finishedAt sql.NullString
		if err := rows.Scan(
			&sess.ID, &sess.Agent, &sess.Model, &sess.Effort,
			&sess.GroupID, &sess.Notes, &sess.Status,
			&startedAt, &finishedAt,
			&sess.TasksTotal, &sess.TasksPassed, &sess.TasksFailed,
			&sess.TasksUnsubmitted, &sess.Warnings,
		); err != nil {
			return nil, fmt.Errorf("store: scan session: %w", err)
		}
		sess.StartedAt, err = time.Parse(time.RFC3339, startedAt)
		if err != nil {
			return nil, fmt.Errorf("store: parse started_at %q: %w", startedAt, err)
		}
		if finishedAt.Valid {
			t, err := time.Parse(time.RFC3339, finishedAt.String)
			if err != nil {
				return nil, fmt.Errorf("store: parse finished_at %q: %w", finishedAt.String, err)
			}
			sess.FinishedAt = sql.NullTime{Time: t, Valid: true}
		}
		sessions = append(sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate sessions: %w", err)
	}
	return sessions, nil
}
