// Template: internal/store/store.go — the EVENT-STORE flavor (GOAL.md §3d).
// Apply when the API pushes an EPHEMERAL event stream (WebSocket / RTM / webhook / long-poll) or has
// no durable history/search endpoint for what the CLI sends and sees — then this local DB is the only
// searchable system-of-record. Pair with a `listen`/`webhook listen` capture (record BEFORE display
// filters), `log`/`log search` (see log.go), a `--no-store` flag (and never open the store under
// --dry-run), and EXCLUDE `listen` from the MCP surface (an agent hangs on a blocking stream). For a
// re-GETtable pull-only API use the cache flavor (store.cache.go) instead.
//
// Recording must NEVER break a command: callers on the write path treat a store error as
// "warn once and continue" — wire it through an observer so `internal/api` never imports store:
//
//	type Recorder interface{ Record(ctx context.Context, e Event) error }  // in internal/api
//	client := api.New(..., api.WithRecorder(store))                        // fire-and-forget, warn-and-continue
//
// Driver is modernc.org/sqlite (pure Go, no cgo) — a cgo driver would break GoReleaser's
// one-toolchain cross-compile (keep CGO_ENABLED=0).
//
// ADAPT the Event fields + schema to your API: Profile=account/workspace, Topic=channel/room/chat,
// Actor=user, Kind=event type, TS must be a value that sorts chronologically AS TEXT (zero-padded).
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// DefaultLimit is the default row cap for Query/Search.
const DefaultLimit = 50

// Event is one recorded event. TS is a string that sorts chronologically as text.
type Event struct {
	ID         int64           `json:"id"`
	Profile    string          `json:"profile"`
	Topic      string          `json:"topic"` // channel/room/chat
	TS         string          `json:"ts"`
	Actor      string          `json:"actor,omitempty"` // user/sender
	Kind       string          `json:"kind,omitempty"`  // event type
	Text       string          `json:"text,omitempty"`  // searchable body
	RecordedAt time.Time       `json:"recorded_at"`
	Raw        json.RawMessage `json:"raw,omitempty"`
}

// Filter narrows Query/Search. The zero value means "no constraint" on that field.
type Filter struct {
	Topic string
	Actor string
	Since string // TS lower bound (inclusive)
	Limit int    // <=0 → DefaultLimit
}

// Stats summarizes what a store holds.
type Stats struct {
	Events int64          `json:"events"`
	Topics int64          `json:"topics"`
	Oldest string         `json:"oldest_ts,omitempty"`
	Newest string         `json:"newest_ts,omitempty"`
	FTS    bool           `json:"fts_enabled"`
	ByKind map[string]int `json:"by_kind,omitempty"`
}

// Store is a per-profile SQLite event history.
type Store struct {
	db  *sql.DB
	fts bool // true when the linked SQLite build includes the FTS5 module
}

// Open opens (creating if needed) the SQLite store and initializes its schema idempotently. The
// parent dir is created 0700 and the file chmod'd 0600 — it holds event text. dbPath is typically
// <configDir>/history/<profile>.db; RE-VALIDATE any user-supplied profile in the path (reject '/'
// and '\') before joining it, to prevent a path escape.
func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	db.SetMaxOpenConns(1) // modernc serializes writers; one invocation needs no concurrency
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure store: %w", err)
	}
	if err := os.Chmod(dbPath, 0o600); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("chmod store: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the handle. Plumb through your client's Close() — Windows can't unlink an
// open-handle DB, which breaks temp-dir cleanup in tests.
func (s *Store) Close() error { return s.db.Close() }

// FTSEnabled reports whether Search uses FTS5 MATCH (true) or a LIKE scan fallback (false).
func (s *Store) FTSEnabled() bool { return s.fts }

const schema = `
CREATE TABLE IF NOT EXISTS events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	profile TEXT NOT NULL,
	topic TEXT NOT NULL,
	ts TEXT NOT NULL,
	actor TEXT,
	kind TEXT,
	text TEXT,
	recorded_at TEXT NOT NULL,
	raw TEXT,
	UNIQUE(profile, topic, ts)
);
CREATE INDEX IF NOT EXISTS idx_events_topic_ts ON events(profile, topic, ts);
`

func (s *Store) migrate() error {
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	// FTS5 is optional in some minimal SQLite builds; Search degrades to a LIKE scan when absent.
	_, err := s.db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS events_fts USING fts5(text)`)
	s.fts = err == nil
	return nil
}

const selectEvent = `SELECT events.id, events.profile, events.topic, events.ts, events.actor,
	events.kind, events.text, events.recorded_at, events.raw`

// Record inserts one event, deduping on (profile, topic, ts) — re-capturing the same history never
// creates a duplicate. Every value travels as a bind arg, so message content is never an injection
// surface.
func (s *Store) Record(ctx context.Context, e Event) error {
	if e.Topic == "" || e.TS == "" {
		return nil // not a locatable event; skip silently
	}
	if e.RecordedAt.IsZero() {
		e.RecordedAt = time.Now().UTC()
	}
	var raw any
	if len(e.Raw) > 0 {
		raw = string(e.Raw)
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO events (profile, topic, ts, actor, kind, text, recorded_at, raw)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Profile, e.Topic, e.TS, nullable(e.Actor), nullable(e.Kind), nullable(e.Text),
		e.RecordedAt.UTC().Format(time.RFC3339Nano), raw)
	if err != nil {
		return fmt.Errorf("record event: %w", err)
	}
	// Index in FTS only when a NEW row was inserted (INSERT OR IGNORE affects 0 rows on a dup).
	if s.fts && e.Text != "" {
		if n, _ := res.RowsAffected(); n == 1 {
			if id, idErr := res.LastInsertId(); idErr == nil {
				_, _ = s.db.ExecContext(ctx, `INSERT INTO events_fts (rowid, text) VALUES (?, ?)`, id, e.Text)
			}
		}
	}
	return nil
}

// RecordBatch records many events in one pass (used when persisting a fetched page).
func (s *Store) RecordBatch(ctx context.Context, events []Event) (int, error) {
	n := 0
	for _, e := range events {
		if err := s.Record(ctx, e); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// Query lists events matching f, newest first.
func (s *Store) Query(ctx context.Context, f Filter) ([]Event, error) {
	where, args := f.whereClause()
	parts := []string{selectEvent, "FROM events"}
	if where != "" {
		parts = append(parts, "WHERE", where)
	}
	parts = append(parts, "ORDER BY events.ts DESC, events.id DESC LIMIT ?")
	args = append(args, effectiveLimit(f.Limit))
	rows, err := s.db.QueryContext(ctx, strings.Join(parts, " "), args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanEvents(rows)
}

// Search full-text searches recorded text, newest first — FTS5 MATCH when available (AND/OR/NOT,
// prefix*, "phrases"), else a substring LIKE scan. A query that isn't valid FTS5 syntax (a bare
// "foo-bar" reads `-bar` as an operator) transparently falls back to LIKE, so a plain search never
// errors. Only static SQL fragments are joined; q and every filter value travel as bind args.
func (s *Store) Search(ctx context.Context, q string, f Filter) ([]Event, error) {
	if s.fts {
		if events, err := s.search(ctx, q, f, true); err == nil {
			return events, nil
		}
		// An FTS5 syntax error on a plain query is not a real failure — retry as a substring scan.
	}
	return s.search(ctx, q, f, false)
}

func (s *Store) search(ctx context.Context, q string, f Filter, useFTS bool) ([]Event, error) {
	where, whereArgs := f.whereClause()
	var parts []string
	var args []any
	if useFTS {
		parts = []string{selectEvent, "FROM events",
			"JOIN events_fts ON events_fts.rowid = events.id", "WHERE events_fts MATCH ?"}
		args = []any{q}
	} else {
		parts = []string{selectEvent, "FROM events", "WHERE events.text LIKE ?"}
		args = []any{"%" + q + "%"}
	}
	if where != "" {
		parts = append(parts, "AND", where)
		args = append(args, whereArgs...)
	}
	parts = append(parts, "ORDER BY events.ts DESC, events.id DESC LIMIT ?")
	args = append(args, effectiveLimit(f.Limit))
	rows, err := s.db.QueryContext(ctx, strings.Join(parts, " "), args...)
	if err != nil {
		return nil, fmt.Errorf("search events: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanEvents(rows)
}

// Stats summarizes the store.
func (s *Store) Stats(ctx context.Context) (Stats, error) {
	st := Stats{FTS: s.fts, ByKind: map[string]int{}}
	row := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COUNT(DISTINCT topic), COALESCE(MIN(ts),''), COALESCE(MAX(ts),'') FROM events`)
	if err := row.Scan(&st.Events, &st.Topics, &st.Oldest, &st.Newest); err != nil {
		return Stats{}, fmt.Errorf("stats: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT COALESCE(kind,''), COUNT(*) FROM events GROUP BY kind`)
	if err != nil {
		return Stats{}, fmt.Errorf("stats by kind: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return Stats{}, err
		}
		if k == "" {
			k = "(none)"
		}
		st.ByKind[k] = n
	}
	return st, rows.Err()
}

// Prune deletes events recorded before now-olderThan (and their FTS rows) and returns the count.
func (s *Store) Prune(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("prune: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once Commit succeeds
	if s.fts {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM events_fts WHERE rowid IN (SELECT id FROM events WHERE recorded_at < ?)`, cutoff); err != nil {
			return 0, fmt.Errorf("prune fts index: %w", err)
		}
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM events WHERE recorded_at < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("prune events: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prune: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("prune: %w", err)
	}
	return n, nil
}

// whereClause renders f as a parameterized fragment (no leading WHERE) plus bind args.
func (f Filter) whereClause() (string, []any) {
	var parts []string
	var args []any
	if f.Topic != "" {
		parts = append(parts, "events.topic = ?")
		args = append(args, f.Topic)
	}
	if f.Actor != "" {
		parts = append(parts, "events.actor = ?")
		args = append(args, f.Actor)
	}
	if f.Since != "" {
		parts = append(parts, "events.ts >= ?")
		args = append(args, f.Since)
	}
	return strings.Join(parts, " AND "), args
}

func effectiveLimit(n int) int {
	if n <= 0 {
		return DefaultLimit
	}
	return n
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func scanEvents(rows *sql.Rows) ([]Event, error) {
	out := []Event{}
	for rows.Next() {
		var (
			e        Event
			recorded string
			actor    sql.NullString
			kind     sql.NullString
			text     sql.NullString
			raw      sql.NullString
		)
		if err := rows.Scan(&e.ID, &e.Profile, &e.Topic, &e.TS, &actor, &kind, &text, &recorded, &raw); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339Nano, recorded)
		if err != nil {
			return nil, fmt.Errorf("parse stored recorded_at %q: %w", recorded, err)
		}
		e.RecordedAt = parsed
		e.Actor = actor.String
		e.Kind = kind.String
		e.Text = text.String
		if raw.Valid {
			e.Raw = json.RawMessage(raw.String)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
