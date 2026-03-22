# 数据模型

## sessions

```sql
CREATE TABLE sessions (
    id              TEXT PRIMARY KEY,
    cwd             TEXT NOT NULL,
    project         TEXT,
    branch          TEXT,
    model           TEXT,
    started_at      INTEGER NOT NULL,
    ended_at        INTEGER,
    end_reason      TEXT,
    total_tool_calls INTEGER DEFAULT 0,
    total_prompts   INTEGER DEFAULT 0
);
CREATE INDEX idx_sessions_started_at ON sessions(started_at);
CREATE INDEX idx_sessions_project ON sessions(project);
```

## prompts

```sql
CREATE TABLE prompts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL REFERENCES sessions(id),
    prompt_text TEXT NOT NULL,
    timestamp   INTEGER NOT NULL
);
CREATE INDEX idx_prompts_session ON prompts(session_id);
```

## tool_calls

```sql
CREATE TABLE tool_calls (
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
CREATE INDEX idx_tool_calls_session ON tool_calls(session_id);
CREATE INDEX idx_tool_calls_name ON tool_calls(tool_name);
CREATE INDEX idx_tool_calls_tool_use_id ON tool_calls(tool_use_id);
```

## stop_events

```sql
CREATE TABLE stop_events (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id    TEXT NOT NULL REFERENCES sessions(id),
    event_type    TEXT NOT NULL,
    error_type    TEXT,
    error_details TEXT,
    timestamp     INTEGER NOT NULL
);
CREATE INDEX idx_stop_events_session ON stop_events(session_id);
```

## schema_version

```sql
CREATE TABLE schema_version (
    version    INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000)
);
```

## 关键 PRAGMA

```sql
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
```

所有时间戳为 unix milliseconds (int64)。
