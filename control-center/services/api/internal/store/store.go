package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hcc.local/classify"
	"hcc.local/observe"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Host struct {
	ID             string
	Name           string
	OS             string
	Arch           string
	Status         string
	LastHeartbeat  string
	AgentVersion   string
	DockerVersion  string
	DockerOK       bool
	DockerError    string
	CPUPercent     float64
	MemUsedBytes   int64
	MemTotalBytes  int64
	DiskUsedBytes  int64
	DiskTotalBytes int64
}

type App struct {
	ID          string
	HostID      string
	Slug        string
	Name        string
	Image       string
	State       string
	Health      string
	Managed     bool
	Hostname    string
	ServicePort int
	ContainerID string
	Networks    string
	ObservedAt  string
	ProjectSlug string
}

type Job struct {
	ID        string
	Type      string
	AppID     string
	HostID    string
	State     string
	ErrorCode string
	Summary   string
	Payload   string
	CreatedAt string
	UpdatedAt string
}

type Audit struct {
	ID        string
	Actor     string
	Action    string
	Target    string
	Result    string
	RequestID string
	CreatedAt string
	Redacted  string
}

type migration struct {
	name string
	up   string
	down string
}

const initSQL = `
CREATE TABLE host_nodes (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  os TEXT NOT NULL DEFAULT '',
  arch TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  last_heartbeat_at TEXT,
  agent_version TEXT NOT NULL DEFAULT '',
  docker_version TEXT NOT NULL DEFAULT '',
  docker_ok INTEGER NOT NULL DEFAULT 0,
  docker_error TEXT NOT NULL DEFAULT '',
  cpu_percent REAL NOT NULL DEFAULT 0,
  mem_used_bytes INTEGER NOT NULL DEFAULT 0,
  mem_total_bytes INTEGER NOT NULL DEFAULT 0,
  disk_used_bytes INTEGER NOT NULL DEFAULT 0,
  disk_total_bytes INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE apps (
  id TEXT PRIMARY KEY,
  host_id TEXT NOT NULL,
  slug TEXT NOT NULL,
  name TEXT NOT NULL,
  image TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  health TEXT NOT NULL DEFAULT '',
  managed INTEGER NOT NULL,
  hostname TEXT NOT NULL DEFAULT '',
  service_port INTEGER NOT NULL DEFAULT 0,
  container_id TEXT NOT NULL,
  networks TEXT NOT NULL DEFAULT '',
  observed_at TEXT NOT NULL,
  UNIQUE(host_id, slug)
);
CREATE TABLE jobs (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  app_id TEXT NOT NULL DEFAULT '',
  host_id TEXT NOT NULL DEFAULT '',
  idempotency_key TEXT,
  state TEXT NOT NULL,
  error_code TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  payload TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_jobs_idempotency ON jobs(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE TABLE job_steps (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  name TEXT NOT NULL,
  state TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  started_at TEXT,
  ended_at TEXT
);
CREATE TABLE audit_events (
  id TEXT PRIMARY KEY,
  actor TEXT NOT NULL,
  action TEXT NOT NULL,
  target TEXT NOT NULL DEFAULT '',
  result TEXT NOT NULL,
  request_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  redacted_diff TEXT NOT NULL DEFAULT ''
);
CREATE TABLE agent_nonces (
  nonce TEXT PRIMARY KEY,
  host_id TEXT NOT NULL,
  seen_at TEXT NOT NULL
);
CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE log_snapshots (
  app_id TEXT PRIMARY KEY,
  body TEXT NOT NULL,
  observed_at TEXT NOT NULL
);
`

const dropSQL = `
DROP TABLE IF EXISTS log_snapshots;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS agent_nonces;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS job_steps;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS apps;
DROP TABLE IF EXISTS host_nodes;
`

const appColumns = `id, host_id, slug, name, image, state, health, managed, hostname, service_port, container_id, networks, observed_at, project_slug`

var ErrDuplicate = errors.New("duplicate project")

var steps = []migration{
	{name: "init", up: initSQL, down: dropSQL},
	{name: "job_state_idx", up: `CREATE INDEX idx_jobs_state ON jobs(state)`, down: `DROP INDEX IF EXISTS idx_jobs_state`},
	{name: "app_project_slug", up: `ALTER TABLE apps ADD COLUMN project_slug TEXT NOT NULL DEFAULT ''`, down: `ALTER TABLE apps DROP COLUMN project_slug`},
	{name: "desired_apps", up: `
CREATE TABLE desired_apps (
  id TEXT PRIMARY KEY,
  host_id TEXT NOT NULL,
  slug TEXT NOT NULL,
  display_name TEXT NOT NULL,
  spec_json TEXT NOT NULL,
  spec_hash TEXT NOT NULL,
  image TEXT NOT NULL,
  state TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(host_id, slug)
)`, down: `DROP TABLE IF EXISTS desired_apps`},
}

func Open(path string) (*Store, error) {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	slash := filepath.ToSlash(path)
	dsn := "file:" + slash + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	if strings.Contains(slash, ":") {
		dsn = "file:///" + slash + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping() error { return s.db.Ping() }

func execScript(db execer, script string) error {
	for _, stmt := range strings.Split(script, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("%s: %w", stmt, err)
		}
	}
	return nil
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func (s *Store) Migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return err
	}
	for i, step := range steps {
		ver := i + 1
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, ver).Scan(&n); err != nil {
			return err
		}
		if n == 1 {
			continue
		}
		if err := execScript(s.db, step.up); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_migrations(version, name, applied_at) VALUES(?,?,?)`, ver, step.name, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Down() error {
	var max int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&max); err != nil {
		return err
	}
	for ver := max; ver >= 1; ver-- {
		if err := execScript(s.db, steps[ver-1].down); err != nil {
			return err
		}
		if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = ?`, ver); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ApplyReport(rep observe.Report, now time.Time) error {
	if rep.HostID == "" {
		return errors.New("host id required")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stamp := now.UTC().Format(time.RFC3339Nano)
	dockerOK := 0
	if rep.Docker.Available {
		dockerOK = 1
	}
	_, err = tx.Exec(`INSERT INTO host_nodes(
		id, name, os, arch, status, last_heartbeat_at, agent_version, docker_version, docker_ok, docker_error,
		cpu_percent, mem_used_bytes, mem_total_bytes, disk_used_bytes, disk_total_bytes
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	ON CONFLICT(id) DO UPDATE SET
		name=excluded.name, os=excluded.os, arch=excluded.arch, status=excluded.status,
		last_heartbeat_at=excluded.last_heartbeat_at, agent_version=excluded.agent_version,
		docker_version=excluded.docker_version, docker_ok=excluded.docker_ok, docker_error=excluded.docker_error,
		cpu_percent=excluded.cpu_percent, mem_used_bytes=excluded.mem_used_bytes, mem_total_bytes=excluded.mem_total_bytes,
		disk_used_bytes=excluded.disk_used_bytes, disk_total_bytes=excluded.disk_total_bytes`,
		rep.HostID, fallback(rep.HostName, rep.HostID), rep.OS, rep.Arch, "Online", stamp, rep.AgentVersion,
		rep.Docker.Version, dockerOK, clip(rep.Docker.Error, 300), rep.CPUPercent,
		rep.MemUsedBytes, rep.MemTotalBytes, rep.DiskUsedBytes, rep.DiskTotalBytes)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM apps WHERE host_id = ?`, rep.HostID); err != nil {
		return err
	}
	seen := map[string]int{}
	for _, c := range rep.Containers {
		if c.ID == "" || c.Name == "" {
			continue
		}
		base := slugify(c.Name)
		slug := base
		if seen[base] > 0 {
			slug = fmt.Sprintf("%s-%d", base, seen[base]+1)
		}
		seen[base]++
		managed := 0
		if classify.Managed(c.Labels) {
			managed = 1
		}
		id := "ctr_" + shortID(c.ID)
		_, err = tx.Exec(`INSERT INTO apps(
			id, host_id, slug, name, image, state, health, managed, hostname, service_port, container_id, networks, observed_at, project_slug
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, rep.HostID, slug, strings.TrimPrefix(c.Name, "/"), c.Image, c.State, fallback(c.Health, "none"),
			managed, classify.Hostname(c.Labels), classify.ServicePort(c.Labels), c.ID, strings.Join(c.Networks, ","), stamp, clip(c.Labels["hcc.project-id"], 40))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Hosts() ([]Host, error) {
	rows, err := s.db.Query(`SELECT id, name, os, arch, status, COALESCE(last_heartbeat_at,''), agent_version, docker_version, docker_ok, docker_error, cpu_percent, mem_used_bytes, mem_total_bytes, disk_used_bytes, disk_total_bytes FROM host_nodes ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHosts(rows)
}

func (s *Store) Host(id string) (Host, error) {
	row := s.db.QueryRow(`SELECT id, name, os, arch, status, COALESCE(last_heartbeat_at,''), agent_version, docker_version, docker_ok, docker_error, cpu_percent, mem_used_bytes, mem_total_bytes, disk_used_bytes, disk_total_bytes FROM host_nodes WHERE id = ?`, id)
	var h Host
	var ok int
	err := row.Scan(&h.ID, &h.Name, &h.OS, &h.Arch, &h.Status, &h.LastHeartbeat, &h.AgentVersion, &h.DockerVersion, &ok, &h.DockerError, &h.CPUPercent, &h.MemUsedBytes, &h.MemTotalBytes, &h.DiskUsedBytes, &h.DiskTotalBytes)
	h.DockerOK = ok == 1
	return h, err
}

func scanHosts(rows *sql.Rows) ([]Host, error) {
	var out []Host
	for rows.Next() {
		var h Host
		var ok int
		if err := rows.Scan(&h.ID, &h.Name, &h.OS, &h.Arch, &h.Status, &h.LastHeartbeat, &h.AgentVersion, &h.DockerVersion, &ok, &h.DockerError, &h.CPUPercent, &h.MemUsedBytes, &h.MemTotalBytes, &h.DiskUsedBytes, &h.DiskTotalBytes); err != nil {
			return nil, err
		}
		h.DockerOK = ok == 1
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) Apps(managed *bool) ([]App, error) {
	q := `SELECT ` + appColumns + ` FROM apps`
	args := []any{}
	if managed != nil {
		q += ` WHERE managed = ?`
		v := 0
		if *managed {
			v = 1
		}
		args = append(args, v)
	}
	q += ` ORDER BY name`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []App
	for rows.Next() {
		a, err := scanApp(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) AppsByHost(hostID string) ([]App, error) {
	rows, err := s.db.Query(`SELECT `+appColumns+` FROM apps WHERE host_id = ? ORDER BY name`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []App
	for rows.Next() {
		a, err := scanApp(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) App(id string) (App, error) {
	row := s.db.QueryRow(`SELECT `+appColumns+` FROM apps WHERE id = ?`, id)
	return scanApp(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanApp(sc scanner) (App, error) {
	var a App
	var managed int
	err := sc.Scan(&a.ID, &a.HostID, &a.Slug, &a.Name, &a.Image, &a.State, &a.Health, &managed, &a.Hostname, &a.ServicePort, &a.ContainerID, &a.Networks, &a.ObservedAt, &a.ProjectSlug)
	a.Managed = managed == 1
	return a, err
}

func (s *Store) UseNonce(hostID, nonce string, now time.Time) error {
	cutoff := now.Add(-10 * time.Minute).UTC().Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`DELETE FROM agent_nonces WHERE seen_at < ?`, cutoff); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO agent_nonces(nonce, host_id, seen_at) VALUES(?,?,?)`, nonce, hostID, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("replay")
	}
	return nil
}

func (s *Store) CreateSession(id string, now, exp time.Time) error {
	_, err := s.db.Exec(`INSERT INTO sessions(id, expires_at, created_at) VALUES(?,?,?)`, id, exp.UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) SessionValid(id string, now time.Time) (bool, error) {
	var exp string
	err := s.db.QueryRow(`SELECT expires_at FROM sessions WHERE id = ?`, id).Scan(&exp)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	t, err := time.Parse(time.RFC3339Nano, exp)
	if err != nil {
		return false, nil
	}
	return now.Before(t), nil
}

func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *Store) AddAudit(id, actor, action, target, result, requestID, diff string, now time.Time) error {
	_, err := s.db.Exec(`INSERT INTO audit_events(id, actor, action, target, result, request_id, created_at, redacted_diff) VALUES(?,?,?,?,?,?,?,?)`,
		id, actor, action, target, result, requestID, now.UTC().Format(time.RFC3339Nano), clip(diff, 500))
	return err
}

func (s *Store) Audit(limit int) ([]Audit, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT id, actor, action, target, result, request_id, created_at, redacted_diff FROM audit_events ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Audit
	for rows.Next() {
		var a Audit
		if err := rows.Scan(&a.ID, &a.Actor, &a.Action, &a.Target, &a.Result, &a.RequestID, &a.CreatedAt, &a.Redacted); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) FindIdempotent(key string) (Job, error) {
	var j Job
	err := s.db.QueryRow(`SELECT id, type, app_id, host_id, state, error_code, summary, payload, created_at, updated_at FROM jobs WHERE idempotency_key = ?`, key).Scan(
		&j.ID, &j.Type, &j.AppID, &j.HostID, &j.State, &j.ErrorCode, &j.Summary, &j.Payload, &j.CreatedAt, &j.UpdatedAt)
	return j, err
}

func (s *Store) EnqueueDeploy(jobID, desiredID, hostID, idemKey, slug, displayName, image, specHash, payload string, now time.Time) (string, bool, error) {
	if existing, err := s.FindIdempotent(idemKey); err == nil {
		return existing.ID, false, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	var state string
	err := s.db.QueryRow(`SELECT state FROM desired_apps WHERE host_id = ? AND slug = ?`, hostID, slug).Scan(&state)
	if err == nil && (state == "queued" || state == "active") {
		return "", false, ErrDuplicate
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	tx, err := s.db.Begin()
	if err != nil {
		return "", false, err
	}
	defer func() { _ = tx.Rollback() }()
	if state == "" {
		if _, err := tx.Exec(`INSERT INTO desired_apps(id, host_id, slug, display_name, spec_json, spec_hash, image, state, created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
			desiredID, hostID, slug, displayName, payload, specHash, image, "queued", stamp); err != nil {
			return "", false, err
		}
	} else {
		if _, err := tx.Exec(`UPDATE desired_apps SET display_name=?, spec_json=?, spec_hash=?, image=?, state='queued' WHERE host_id=? AND slug=?`,
			displayName, payload, specHash, image, hostID, slug); err != nil {
			return "", false, err
		}
	}
	if _, err := tx.Exec(`INSERT INTO jobs(id, type, app_id, host_id, idempotency_key, state, payload, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		jobID, "deploy", desiredID, hostID, idemKey, "queued", payload, stamp, stamp); err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(`INSERT INTO job_steps(job_id, sequence, name, state, summary, started_at) VALUES(?,?,?,?,?,?)`,
		jobID, 1, "deploy", "queued", "menunggu agent", stamp); err != nil {
		return "", false, err
	}
	return jobID, true, tx.Commit()
}

func (s *Store) EnqueueAction(jobID, kind, hostID, appID, slug, idemKey string, now time.Time) (string, bool, error) {
	switch kind {
	case "start", "stop", "restart", "delete":
	default:
		return "", false, errors.New("job rejected")
	}
	if existing, err := s.FindIdempotent(idemKey); err == nil {
		return existing.ID, false, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	payload, _ := json.Marshal(map[string]string{"slug": slug})
	stamp := now.UTC().Format(time.RFC3339Nano)
	tx, err := s.db.Begin()
	if err != nil {
		return "", false, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`INSERT INTO jobs(id, type, app_id, host_id, idempotency_key, state, payload, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		jobID, kind, appID, hostID, idemKey, "queued", string(payload), stamp, stamp); err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(`INSERT INTO job_steps(job_id, sequence, name, state, summary, started_at) VALUES(?,?,?,?,?,?)`,
		jobID, 1, kind, "queued", "menunggu agent", stamp); err != nil {
		return "", false, err
	}
	return jobID, true, tx.Commit()
}

func (s *Store) EnqueueLogs(id, hostID, appID, containerID string, tail int, now time.Time) (string, error) {
	var existing string
	err := s.db.QueryRow(`SELECT id FROM jobs WHERE app_id = ? AND type = 'observe_logs' AND state = 'queued' LIMIT 1`, appID).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	payload, _ := json.Marshal(map[string]any{"containerId": containerID, "tail": tail})
	stamp := now.UTC().Format(time.RFC3339Nano)
	_, err = s.db.Exec(`INSERT INTO jobs(id, type, app_id, host_id, state, payload, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		id, "observe_logs", appID, hostID, "queued", string(payload), stamp, stamp)
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(`INSERT INTO job_steps(job_id, sequence, name, state, summary, started_at) VALUES(?,?,?,?,?,?)`,
		id, 1, "observe_logs", "queued", "menunggu agent", stamp)
	return id, err
}

func (s *Store) PendingJobs(hostID string) ([]observe.AssignedJob, error) {
	rows, err := s.db.Query(`SELECT id, type, app_id, payload FROM jobs WHERE host_id = ? AND state = 'queued' AND type IN ('observe_logs','deploy','start','stop','restart','delete') ORDER BY created_at LIMIT 5`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []observe.AssignedJob
	for rows.Next() {
		var id, kind, appID, payload string
		if err := rows.Scan(&id, &kind, &appID, &payload); err != nil {
			return nil, err
		}
		var body struct {
			ContainerID string `json:"containerId"`
			Tail        int    `json:"tail"`
			Slug        string `json:"slug"`
			DisplayName string `json:"displayName"`
			Template    string `json:"template"`
			SpecHash    string `json:"specHash"`
		}
		_ = json.Unmarshal([]byte(payload), &body)
		out = append(out, observe.AssignedJob{
			ID: id, Type: kind, AppID: appID, ContainerID: body.ContainerID, Tail: body.Tail,
			Slug: body.Slug, DisplayName: body.DisplayName, Template: body.Template, SpecHash: body.SpecHash,
		})
	}
	return out, rows.Err()
}

func (s *Store) CompleteLogJob(id, hostID, state, summary, logs, code string, now time.Time) error {
	var typ, jobHost, cur string
	err := s.db.QueryRow(`SELECT type, host_id, state FROM jobs WHERE id = ?`, id).Scan(&typ, &jobHost, &cur)
	if errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err != nil {
		return err
	}
	switch typ {
	case "observe_logs", "deploy", "start", "stop", "restart", "delete":
	default:
		return errors.New("job rejected")
	}
	if jobHost != hostID {
		return errors.New("job rejected")
	}
	if cur == "succeeded" || cur == "failed" {
		return nil
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`UPDATE jobs SET state = ?, summary = ?, error_code = ?, updated_at = ? WHERE id = ?`, state, clip(summary, 300), code, stamp, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE job_steps SET state = ?, summary = ?, ended_at = ? WHERE job_id = ?`, state, clip(summary, 300), stamp, id); err != nil {
		return err
	}
	if state == "succeeded" && typ == "observe_logs" {
		var appID string
		if err := tx.QueryRow(`SELECT app_id FROM jobs WHERE id = ?`, id).Scan(&appID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO log_snapshots(app_id, body, observed_at) VALUES(?,?,?)
			ON CONFLICT(app_id) DO UPDATE SET body = excluded.body, observed_at = excluded.observed_at`, appID, logs, stamp); err != nil {
			return err
		}
	}
	if typ == "deploy" || typ == "delete" {
		var payload string
		if err := tx.QueryRow(`SELECT payload FROM jobs WHERE id = ?`, id).Scan(&payload); err != nil {
			return err
		}
		var body struct {
			Slug string `json:"slug"`
		}
		_ = json.Unmarshal([]byte(payload), &body)
		next := "failed"
		if state == "succeeded" && typ == "deploy" {
			next = "active"
		}
		if state == "succeeded" && typ == "delete" {
			next = "removed"
		}
		if state == "failed" && typ == "deploy" {
			next = "failed"
		}
		if body.Slug != "" && (typ == "deploy" || (typ == "delete" && state == "succeeded")) {
			if _, err := tx.Exec(`UPDATE desired_apps SET state = ? WHERE host_id = ? AND slug = ?`, next, jobHost, body.Slug); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) Jobs(limit int) ([]Job, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.db.Query(`SELECT id, type, app_id, host_id, state, error_code, summary, payload, created_at, updated_at FROM jobs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.Type, &j.AppID, &j.HostID, &j.State, &j.ErrorCode, &j.Summary, &j.Payload, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) Job(id string) (Job, error) {
	var j Job
	err := s.db.QueryRow(`SELECT id, type, app_id, host_id, state, error_code, summary, payload, created_at, updated_at FROM jobs WHERE id = ?`, id).Scan(
		&j.ID, &j.Type, &j.AppID, &j.HostID, &j.State, &j.ErrorCode, &j.Summary, &j.Payload, &j.CreatedAt, &j.UpdatedAt)
	return j, err
}

func (s *Store) LogSnapshot(appID string) (string, string, error) {
	var body, at string
	err := s.db.QueryRow(`SELECT body, observed_at FROM log_snapshots WHERE app_id = ?`, appID).Scan(&body, &at)
	return body, at, err
}

func (s *Store) CountJobs() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM jobs`).Scan(&n)
	return n, err
}

func slugify(name string) string {
	name = strings.ToLower(strings.TrimPrefix(name, "/"))
	var b strings.Builder
	dash := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 63 {
		s = strings.Trim(s[:63], "-")
	}
	if s == "" {
		return "container"
	}
	return s
}

func shortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 20 {
		return id[:20]
	}
	return id
}

func fallback(v, alt string) string {
	if strings.TrimSpace(v) == "" {
		return alt
	}
	return v
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
