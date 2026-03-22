package store

const currentSchemaVersion = 2

const schemaSQL = `
CREATE TABLE IF NOT EXISTS schema_version (
    version    INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000)
);

CREATE TABLE IF NOT EXISTS sessions (
    id               TEXT PRIMARY KEY,
    cwd              TEXT NOT NULL,
    project          TEXT,
    branch           TEXT,
    model            TEXT,
    started_at       INTEGER NOT NULL,
    ended_at         INTEGER,
    end_reason       TEXT,
    total_tool_calls INTEGER DEFAULT 0,
    total_prompts    INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_sessions_started_at ON sessions(started_at);
CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project);

CREATE TABLE IF NOT EXISTS prompts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL REFERENCES sessions(id),
    prompt_text TEXT NOT NULL,
    timestamp   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_prompts_session ON prompts(session_id);

CREATE TABLE IF NOT EXISTS tool_calls (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id       TEXT NOT NULL REFERENCES sessions(id),
    tool_use_id      TEXT,
    tool_name        TEXT NOT NULL,
    tool_input_json  TEXT,
    tool_output_json TEXT,
    succeeded        INTEGER DEFAULT 1,
    error_message    TEXT,
    started_at       INTEGER NOT NULL,
    completed_at     INTEGER,
    duration_ms      INTEGER
);
CREATE INDEX IF NOT EXISTS idx_tool_calls_session ON tool_calls(session_id);
CREATE INDEX IF NOT EXISTS idx_tool_calls_name ON tool_calls(tool_name);
CREATE INDEX IF NOT EXISTS idx_tool_calls_tool_use_id ON tool_calls(tool_use_id);

CREATE TABLE IF NOT EXISTS stop_events (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id    TEXT NOT NULL REFERENCES sessions(id),
    event_type    TEXT NOT NULL,
    error_type    TEXT,
    error_details TEXT,
    timestamp     INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_stop_events_session ON stop_events(session_id);
`

const migrationV2SQL = `
ALTER TABLE sessions ADD COLUMN total_input_tokens INTEGER DEFAULT 0;
ALTER TABLE sessions ADD COLUMN total_output_tokens INTEGER DEFAULT 0;
ALTER TABLE sessions ADD COLUMN total_cache_read_tokens INTEGER DEFAULT 0;
ALTER TABLE sessions ADD COLUMN total_cache_creation_tokens INTEGER DEFAULT 0;
`
