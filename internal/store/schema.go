package store

// schema contains the DDL statements run once at database open.
const schema = `
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS sessions (
    id                TEXT PRIMARY KEY,
    agent             TEXT NOT NULL,
    model             TEXT NOT NULL,
    effort            TEXT NOT NULL,
    group_id          TEXT,
    notes             TEXT,
    status            TEXT NOT NULL,
    started_at        TEXT NOT NULL,
    finished_at       TEXT,
    tasks_total       INTEGER NOT NULL,
    tasks_passed      INTEGER NOT NULL DEFAULT 0,
    tasks_failed      INTEGER NOT NULL DEFAULT 0,
    tasks_unsubmitted INTEGER NOT NULL DEFAULT 0,
    warnings          INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_sessions_started_at  ON sessions(started_at);
CREATE INDEX IF NOT EXISTS idx_sessions_model_effort ON sessions(model, effort);
CREATE INDEX IF NOT EXISTS idx_sessions_group_id     ON sessions(group_id);

CREATE TABLE IF NOT EXISTS answers (
    session_id   TEXT NOT NULL REFERENCES sessions(id),
    task_id      TEXT NOT NULL,
    category     TEXT NOT NULL,
    difficulty   INTEGER NOT NULL,
    answer       TEXT,
    passed       INTEGER,
    submitted_at TEXT,
    PRIMARY KEY (session_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_answers_task ON answers(task_id);
`
