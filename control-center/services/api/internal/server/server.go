package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"hcc.local/api/internal/provider"
	"hcc.local/api/internal/store"
	"hcc.local/classify"
	"hcc.local/manifest"
	"hcc.local/observe"
	"hcc.local/redact"
	"hcc.local/sign"
)

const (
	staleAfter  = 45 * time.Second
	sessionTTL  = 8 * time.Hour
	logFreshFor = 20 * time.Second
	maxLogTail  = 500
	cfCacheTTL  = 60 * time.Second
)

type Server struct {
	Store    *store.Store
	Password string
	Secret   string
	Docker   provider.DockerProvider
	CF       provider.CloudflareProvider
	Now      func() time.Time

	mu       sync.Mutex
	fails    map[string][]time.Time
	cfMu     sync.Mutex
	cfAt     time.Time
	cfStatus provider.Status
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Server) Handler() http.Handler {
	if s.Docker == nil {
		s.Docker = provider.ObservedDocker{}
	}
	if s.CF == nil {
		s.CF = provider.Unconfigured{}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.health)
	mux.HandleFunc("POST /api/v1/session", s.login)
	mux.HandleFunc("POST /api/v1/logout", s.withSession(s.logout))
	mux.HandleFunc("GET /api/v1/overview", s.withSession(s.overview))
	mux.HandleFunc("GET /api/v1/hosts", s.withSession(s.hosts))
	mux.HandleFunc("GET /api/v1/hosts/{id}", s.withSession(s.host))
	mux.HandleFunc("GET /api/v1/apps", s.withSession(s.apps))
	mux.HandleFunc("GET /api/v1/apps/{id}", s.withSession(s.app))
	mux.HandleFunc("GET /api/v1/apps/{id}/logs", s.withSession(s.logs))
	mux.HandleFunc("POST /api/v1/apps/validate", s.withSession(s.validateApp))
	mux.HandleFunc("POST /api/v1/apps", s.withSession(s.createApp))
	mux.HandleFunc("POST /api/v1/apps/{id}/actions", s.withSession(s.action))
	mux.HandleFunc("DELETE /api/v1/apps/{id}", s.withSession(s.deleteApp))
	mux.HandleFunc("GET /api/v1/jobs", s.withSession(s.jobs))
	mux.HandleFunc("GET /api/v1/jobs/{id}", s.withSession(s.job))
	mux.HandleFunc("POST /api/v1/jobs/{id}/retry", s.withSession(s.disabled))
	mux.HandleFunc("GET /api/v1/audit", s.withSession(s.audit))
	mux.HandleFunc("GET /api/v1/routes", s.withSession(s.routes))
	mux.HandleFunc("GET /api/v1/integrations/cloudflare/status", s.withSession(s.routes))
	mux.HandleFunc("POST /agent/v1/poll", s.agentPoll)
	mux.HandleFunc("POST /agent/v1/jobs/{id}/result", s.agentResult)
	return s.wrap(mux)
}

func (s *Server) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Request-Id", requestID(r))
		if origin := r.Header.Get("Origin"); origin != "" && !loopbackOrigin(origin) {
			writeProblem(w, http.StatusForbidden, "ORIGIN_DENIED", "Origin tidak diizinkan.", requestID(r), false, "Gunakan dashboard di 127.0.0.1.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loopbackOrigin(origin string) bool {
	switch origin {
	case "http://127.0.0.1:3000", "http://localhost:3000", "http://127.0.0.1:8080", "http://localhost:8080":
		return true
	default:
		return false
	}
}

func requestID(r *http.Request) string {
	if v := r.Header.Get("X-Request-Id"); v != "" && len(v) < 80 {
		return v
	}
	return "req_" + randomHex(8)
}

func (s *Server) withSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("hcc_session")
		if err != nil {
			writeProblem(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Sesi tidak ada.", requestID(r), false, "Masuk sebagai admin.")
			return
		}
		ok, err := s.Store.SessionValid(c.Value, s.now())
		if err != nil || !ok {
			writeProblem(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Sesi tidak berlaku.", requestID(r), false, "Masuk lagi.")
			return
		}
		next(w, r)
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ready := s.Store.Ping() == nil
	status := http.StatusOK
	body := map[string]any{"status": "ok", "ready": ready, "version": "0.1.0"}
	if !ready {
		status = http.StatusServiceUnavailable
		body["status"] = "unavailable"
	}
	writeJSON(w, status, body)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if s.limited(r.RemoteAddr) {
		writeProblem(w, http.StatusTooManyRequests, "RATE_LIMITED", "Terlalu banyak percobaan masuk.", requestID(r), true, "Tunggu beberapa menit.")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		writeProblem(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid.", requestID(r), false, "Kirim JSON password.")
		return
	}
	if !matchPassword(body.Password, s.Password) {
		s.noteFail(r.RemoteAddr)
		_ = s.Store.AddAudit(newID("aud_"), "anonymous", "login", "session", "denied", requestID(r), "", s.now())
		writeProblem(w, http.StatusUnauthorized, "AUTH_DENIED", "Kata sandi ditolak.", requestID(r), false, "Periksa HCC_ADMIN_PASSWORD di environment proses API.")
		return
	}
	id := newID("ses_")
	csrf := randomHex(16)
	now := s.now()
	if err := s.Store.CreateSession(id, now, now.Add(sessionTTL)); err != nil {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Sesi tidak tersimpan.", requestID(r), true, "Periksa file SQLite.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "hcc_session", Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: int(sessionTTL.Seconds())})
	http.SetCookie(w, &http.Cookie{Name: "hcc_csrf", Value: csrf, Path: "/", HttpOnly: false, SameSite: http.SameSiteStrictMode, MaxAge: int(sessionTTL.Seconds())})
	_ = s.Store.AddAudit(newID("aud_"), "admin", "login", "session", "ok", requestID(r), "", now)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !s.csrfOK(r) {
		writeProblem(w, http.StatusForbidden, "CSRF_REQUIRED", "Token CSRF tidak cocok.", requestID(r), false, "Muat ulang halaman.")
		return
	}
	if c, err := r.Cookie("hcc_session"); err == nil {
		_ = s.Store.DeleteSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "hcc_session", Value: "", Path: "/", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: "hcc_csrf", Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) overview(w http.ResponseWriter, r *http.Request) {
	hosts, err := s.Store.Hosts()
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Host tidak terbaca.", requestID(r), true, "Periksa SQLite.")
		return
	}
	cf := s.cloudflare()
	if len(hosts) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"host":          nil,
			"docker":        map[string]any{"status": "Unknown", "version": "", "lastCheckedAt": nil},
			"traefik":       map[string]any{"status": "Unknown", "lastCheckedAt": nil},
			"cloudflared":   map[string]any{"container": "Unknown", "controlPlane": cf.TunnelControlPlane},
			"socketProxy":   map[string]any{"status": "Unknown"},
			"counts":        map[string]int{"managed": 0, "unmanaged": 0},
			"integration":   integrationView(cf),
			"lastCheckedAt": nil,
		})
		return
	}
	h := freshest(hosts)
	fresh := freshHost(h, s.now())
	apps, err := s.Store.AppsByHost(h.ID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Aplikasi tidak terbaca.", requestID(r), true, "Periksa SQLite.")
		return
	}
	managed, unmanaged := 0, 0
	for _, a := range apps {
		if a.Managed {
			managed++
		} else {
			unmanaged++
		}
	}
	dockerStatus := "Unknown"
	if fresh {
		if h.DockerOK {
			dockerStatus = "Running"
		} else {
			dockerStatus = "Offline"
		}
	}
	hostStatus := "Offline"
	if fresh {
		hostStatus = "Online"
	} else if h.LastHeartbeat == "" {
		hostStatus = "Unknown"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"host":          hostView(h, hostStatus),
		"docker":        map[string]any{"status": dockerStatus, "version": h.DockerVersion, "error": h.DockerError, "lastCheckedAt": h.LastHeartbeat},
		"traefik":       map[string]any{"status": componentStatus(apps, "proxy", fresh), "lastCheckedAt": h.LastHeartbeat},
		"cloudflared":   map[string]any{"container": componentStatus(apps, "tunnel", fresh), "controlPlane": cf.TunnelControlPlane, "lastCheckedAt": h.LastHeartbeat},
		"socketProxy":   map[string]any{"status": componentStatus(apps, "socket-proxy", fresh)},
		"counts":        map[string]int{"managed": managed, "unmanaged": unmanaged},
		"integration":   integrationView(cf),
		"lastCheckedAt": h.LastHeartbeat,
	})
}

func (s *Server) hosts(w http.ResponseWriter, r *http.Request) {
	hosts, err := s.Store.Hosts()
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Host tidak terbaca.", requestID(r), true, "Periksa SQLite.")
		return
	}
	out := make([]map[string]any, 0, len(hosts))
	for _, h := range hosts {
		status := "Offline"
		if freshHost(h, s.now()) {
			status = "Online"
		} else if h.LastHeartbeat == "" {
			status = "Unknown"
		}
		out = append(out, hostView(h, status))
	}
	writeJSON(w, http.StatusOK, map[string]any{"hosts": out})
}

func (s *Server) host(w http.ResponseWriter, r *http.Request) {
	h, err := s.Store.Host(r.PathValue("id"))
	if err != nil {
		writeProblem(w, http.StatusNotFound, "NOT_FOUND", "Host tidak ditemukan.", requestID(r), false, "Periksa daftar host.")
		return
	}
	status := "Offline"
	if freshHost(h, s.now()) {
		status = "Online"
	}
	writeJSON(w, http.StatusOK, hostView(h, status))
}

func (s *Server) apps(w http.ResponseWriter, r *http.Request) {
	var managed *bool
	switch r.URL.Query().Get("managed") {
	case "true":
		v := true
		managed = &v
	case "false":
		v := false
		managed = &v
	}
	apps, err := s.Store.Apps(managed)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Aplikasi tidak terbaca.", requestID(r), true, "Periksa SQLite.")
		return
	}
	out := make([]map[string]any, 0, len(apps))
	for _, a := range apps {
		out = append(out, appView(a))
	}
	writeJSON(w, http.StatusOK, map[string]any{"apps": out})
}

func (s *Server) app(w http.ResponseWriter, r *http.Request) {
	a, err := s.Store.App(r.PathValue("id"))
	if err != nil {
		writeProblem(w, http.StatusNotFound, "NOT_FOUND", "Aplikasi tidak ditemukan.", requestID(r), false, "Muat ulang daftar.")
		return
	}
	view := appView(a)
	view["networks"] = splitCSV(a.Networks)
	view["desiredSpec"] = nil
	view["publicUrlReady"] = false
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) logs(w http.ResponseWriter, r *http.Request) {
	a, err := s.Store.App(r.PathValue("id"))
	if err != nil {
		writeProblem(w, http.StatusNotFound, "NOT_FOUND", "Aplikasi tidak ditemukan.", requestID(r), false, "Muat ulang daftar.")
		return
	}
	tail := 200
	if raw := r.URL.Query().Get("tail"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxLogTail {
			writeProblem(w, http.StatusBadRequest, "BAD_REQUEST", "Parameter tail di luar batas.", requestID(r), false, "Gunakan tail antara 1 dan 500.")
			return
		}
		tail = n
	}
	body, at, snapErr := s.Store.LogSnapshot(a.ID)
	pending := false
	jobID := ""
	stale := snapErr != nil || logAge(at, s.now()) > logFreshFor
	if stale {
		jobID, err = s.Store.EnqueueLogs(newID("job_"), a.HostID, a.ID, a.ContainerID, tail, s.now())
		if err != nil {
			writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Antrean log gagal.", requestID(r), true, "Periksa agent dan SQLite.")
			return
		}
		pending = true
		_ = s.Store.AddAudit(newID("aud_"), "admin", "read_logs", a.ID, "queued", requestID(r), "", s.now())
	}
	if snapErr != nil {
		body = ""
		at = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"appId": a.ID, "tail": tail, "text": body, "redacted": true,
		"observedAt": emptyNil(at), "pending": pending, "jobId": emptyNil(jobID),
	})
}

func (s *Server) validateApp(w http.ResponseWriter, r *http.Request) {
	if !s.mutationHeaders(w, r) {
		return
	}
	spec, err := readSpec(r)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), requestID(r), false, "Periksa slug, nama, dan template nginx.")
		return
	}
	file, err := manifest.Render(spec)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "DOCKER_POLICY_DENIED", err.Error(), requestID(r), false, "Hanya template nginx internal yang diizinkan.")
		return
	}
	before, _ := s.Store.CountJobs()
	after, _ := s.Store.CountJobs()
	if before != after {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Pratinjau tidak boleh membuat job.", requestID(r), false, "Hentikan API dan laporkan.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "compose": file.YAML, "specHash": file.Hash, "project": file.Project,
		"image": file.Image, "network": file.Network, "publicUrlReady": false, "violations": []string{},
	})
}

func (s *Server) createApp(w http.ResponseWriter, r *http.Request) {
	if !s.mutationHeaders(w, r) {
		return
	}
	spec, err := readSpec(r)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), requestID(r), false, "Periksa slug, nama, dan template nginx.")
		return
	}
	file, err := manifest.Render(spec)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "DOCKER_POLICY_DENIED", err.Error(), requestID(r), false, "Hanya template nginx internal yang diizinkan.")
		return
	}
	host, ok := s.onlineHost()
	if !ok {
		writeProblem(w, http.StatusConflict, "HOST_OFFLINE", "Host atau Docker tidak siap.", requestID(r), true, "Nyalakan agent dan Docker, lalu coba lagi.")
		return
	}
	payload, _ := json.Marshal(map[string]string{
		"slug": spec.Slug, "displayName": spec.DisplayName, "template": spec.Template, "specHash": file.Hash,
	})
	jobID, _, err := s.Store.EnqueueDeploy(newID("job_"), newID("des_"), host.ID, r.Header.Get("Idempotency-Key"), spec.Slug, spec.DisplayName, file.Image, file.Hash, string(payload), s.now())
	if errors.Is(err, store.ErrDuplicate) {
		writeProblem(w, http.StatusConflict, "DUPLICATE_PROJECT", "Slug ini sudah punya deploy yang aktif.", requestID(r), false, "Gunakan slug lain atau hapus proyek yang ada.")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Job deploy tidak tersimpan.", requestID(r), true, "Periksa SQLite.")
		return
	}
	_ = s.Store.AddAudit(newID("aud_"), "admin", "deploy", spec.Slug, "queued", requestID(r), "", s.now())
	writeJSON(w, http.StatusAccepted, map[string]any{"jobId": jobID, "state": "queued", "specHash": file.Hash, "publicUrlReady": false})
}

func (s *Server) action(w http.ResponseWriter, r *http.Request) {
	if !s.mutationHeaders(w, r) {
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		writeProblem(w, http.StatusBadRequest, "BAD_REQUEST", "Body aksi tidak valid.", requestID(r), false, "Kirim action start, stop, atau restart.")
		return
	}
	s.queueAction(w, r, body.Action, false)
}

func (s *Server) deleteApp(w http.ResponseWriter, r *http.Request) {
	if !s.mutationHeaders(w, r) {
		return
	}
	var body struct {
		Confirm bool `json:"confirm"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil || !body.Confirm {
		writeProblem(w, http.StatusBadRequest, "CONFIRMATION_REQUIRED", "Penghapusan butuh konfirmasi.", requestID(r), false, "Kirim confirm true. Volume tidak dihapus.")
		return
	}
	s.queueAction(w, r, "delete", true)
}

func (s *Server) queueAction(w http.ResponseWriter, r *http.Request, kind string, delete bool) {
	switch kind {
	case "start", "stop", "restart", "delete":
	default:
		writeProblem(w, http.StatusBadRequest, "BAD_REQUEST", "Aksi tidak dikenal.", requestID(r), false, "Gunakan start, stop, atau restart.")
		return
	}
	a, err := s.Store.App(r.PathValue("id"))
	if err != nil {
		writeProblem(w, http.StatusNotFound, "NOT_FOUND", "Aplikasi tidak ditemukan.", requestID(r), false, "Muat ulang daftar.")
		return
	}
	if !a.Managed {
		_ = s.Store.AddAudit(newID("aud_"), "admin", "mutate_unmanaged", a.ID, "denied", requestID(r), "", s.now())
		writeProblem(w, http.StatusConflict, "UNMANAGED_APP", "Container ini tidak dimiliki HCC.", requestID(r), false, "Unmanaged hanya bisa dilihat.")
		return
	}
	if err := manifest.ValidateSlug(a.ProjectSlug); err != nil {
		writeProblem(w, http.StatusConflict, "OWNERSHIP_UNKNOWN", "Proyek HCC tidak dikenali.", requestID(r), false, "Hanya stack yang dibuat wizard yang bisa diubah.")
		return
	}
	if _, ok := s.onlineHost(); !ok {
		writeProblem(w, http.StatusConflict, "HOST_OFFLINE", "Host atau Docker tidak siap.", requestID(r), true, "Nyalakan agent dan Docker, lalu coba lagi.")
		return
	}
	jobID, _, err := s.Store.EnqueueAction(newID("job_"), kind, a.HostID, a.ID, a.ProjectSlug, r.Header.Get("Idempotency-Key"), s.now())
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Aksi tidak mengantri.", requestID(r), true, "Periksa SQLite.")
		return
	}
	_ = delete
	_ = s.Store.AddAudit(newID("aud_"), "admin", kind, a.ProjectSlug, "queued", requestID(r), "", s.now())
	writeJSON(w, http.StatusAccepted, map[string]any{"jobId": jobID, "state": "queued", "action": kind, "preserveVolumes": true})
}

func readSpec(r *http.Request) (manifest.Spec, error) {
	var body struct {
		Slug        string `json:"slug"`
		DisplayName string `json:"displayName"`
		Template    string `json:"template"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&body); err != nil {
		return manifest.Spec{}, errors.New("body tidak valid")
	}
	return manifest.Normalize(manifest.Spec{Slug: body.Slug, DisplayName: body.DisplayName, Template: body.Template})
}

func (s *Server) onlineHost() (store.Host, bool) {
	hosts, err := s.Store.Hosts()
	if err != nil || len(hosts) == 0 {
		return store.Host{}, false
	}
	h := freshest(hosts)
	if !freshHost(h, s.now()) || !h.DockerOK {
		return h, false
	}
	return h, true
}

func (s *Server) disabled(w http.ResponseWriter, r *http.Request) {
	if !s.mutationHeaders(w, r) {
		return
	}
	before, _ := s.Store.CountJobs()
	_ = s.Store.AddAudit(newID("aud_"), "admin", "deploy_disabled", r.URL.Path, "denied", requestID(r), "", s.now())
	after, _ := s.Store.CountJobs()
	if after != before {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Penolakan tidak boleh membuat job.", requestID(r), false, "Hentikan API dan laporkan.")
		return
	}
	writeProblem(w, http.StatusForbidden, "FEATURE_DISABLED", "Deploy belum dibuka pada gate ini.", requestID(r), false, "Tidak ada perubahan Docker atau DNS.")
}

func (s *Server) jobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.Store.Jobs(20)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Job tidak terbaca.", requestID(r), true, "Periksa SQLite.")
		return
	}
	out := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, jobView(j))
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": out})
}

func (s *Server) job(w http.ResponseWriter, r *http.Request) {
	j, err := s.Store.Job(r.PathValue("id"))
	if err != nil {
		writeProblem(w, http.StatusNotFound, "NOT_FOUND", "Job tidak ditemukan.", requestID(r), false, "Muat ulang daftar job.")
		return
	}
	writeJSON(w, http.StatusOK, jobView(j))
}

func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	events, err := s.Store.Audit(50)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Audit tidak terbaca.", requestID(r), true, "Periksa SQLite.")
		return
	}
	out := make([]map[string]any, 0, len(events))
	for _, e := range events {
		out = append(out, map[string]any{
			"id": e.ID, "actor": e.Actor, "action": e.Action, "target": e.Target,
			"result": e.Result, "requestId": e.RequestID, "createdAt": e.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": out})
}

func (s *Server) routes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.cloudflare())
}

func (s *Server) agentPoll(w http.ResponseWriter, r *http.Request) {
	body, rep, ok := s.readAgent(w, r)
	if !ok {
		return
	}
	_ = body
	if rep.HostID != r.Header.Get("X-HCC-Host") {
		writeProblem(w, http.StatusForbidden, "HOST_MISMATCH", "Host pada body dan header berbeda.", requestID(r), false, "Samakan host id agent.")
		return
	}
	if err := s.Store.ApplyReport(rep, s.now()); err != nil {
		writeProblem(w, http.StatusBadRequest, "BAD_REQUEST", "Laporan observasi ditolak.", requestID(r), false, "Periksa schema laporan agent.")
		return
	}
	jobs, err := s.Store.PendingJobs(rep.HostID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "STORE_ERROR", "Antrean tidak terbaca.", requestID(r), true, "Periksa SQLite.")
		return
	}
	if jobs == nil {
		jobs = []observe.AssignedJob{}
	}
	writeJSON(w, http.StatusOK, observe.PollResponse{Jobs: jobs})
}

func (s *Server) agentResult(w http.ResponseWriter, r *http.Request) {
	body, ok := s.authAgent(w, r)
	if !ok {
		return
	}
	var result observe.JobResult
	if err := json.Unmarshal(body, &result); err != nil {
		writeProblem(w, http.StatusBadRequest, "BAD_REQUEST", "Hasil job tidak valid.", requestID(r), false, "Kirim JSON hasil.")
		return
	}
	state := result.State
	if state != "succeeded" && state != "failed" {
		state = "failed"
	}
	logs := redact.Text(result.Logs)
	if err := s.Store.CompleteLogJob(r.PathValue("id"), r.Header.Get("X-HCC-Host"), state, redact.Text(result.Summary), logs, result.Code, s.now()); err != nil {
		status := http.StatusConflict
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeProblem(w, status, "JOB_REJECTED", "Hasil job tidak disimpan.", requestID(r), false, "Job harus bertipe observe_logs untuk host ini.")
		return
	}
	_ = s.Store.AddAudit(newID("aud_"), "agent:"+r.Header.Get("X-HCC-Host"), "observe_logs", r.PathValue("id"), state, requestID(r), "", s.now())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) readAgent(w http.ResponseWriter, r *http.Request) ([]byte, observe.Report, bool) {
	body, ok := s.authAgent(w, r)
	if !ok {
		return nil, observe.Report{}, false
	}
	var rep observe.Report
	if err := json.Unmarshal(body, &rep); err != nil {
		writeProblem(w, http.StatusBadRequest, "BAD_REQUEST", "JSON observasi tidak valid.", requestID(r), false, "Kirim laporan schemaVersion 1.")
		return nil, observe.Report{}, false
	}
	return body, rep, true
}

func (s *Server) authAgent(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	if s.Secret == "" {
		writeProblem(w, http.StatusUnauthorized, "AGENT_NOT_CONFIGURED", "Rahasia agent belum diatur.", requestID(r), false, "Isi HCC_AGENT_SECRET di API dan agent.")
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak terbaca.", requestID(r), false, "Kirim ulang.")
		return nil, false
	}
	hostID := r.Header.Get("X-HCC-Host")
	ts := r.Header.Get("X-HCC-Timestamp")
	nonce := r.Header.Get("X-HCC-Nonce")
	sig := r.Header.Get("X-HCC-Signature")
	if err := (hmacTransport{}).Verify(s.Secret, ts, nonce, hostID, sig, body, s.now()); err != nil {
		_ = s.Store.AddAudit(newID("aud_"), "agent", "auth", hostID, "denied", requestID(r), "", s.now())
		writeProblem(w, http.StatusUnauthorized, "AGENT_DENIED", "Tanda tangan agent ditolak.", requestID(r), false, "Periksa jam, nonce, dan rahasia host.")
		return nil, false
	}
	if err := s.Store.UseNonce(hostID, nonce, s.now()); err != nil {
		writeProblem(w, http.StatusUnauthorized, "REPLAY", "Nonce sudah dipakai.", requestID(r), false, "Buat nonce baru untuk tiap permintaan.")
		return nil, false
	}
	return body, true
}

type hmacTransport struct{}

func (hmacTransport) Verify(secret, ts, nonce, hostID, sig string, body []byte, now time.Time) error {
	return sign.Verify(secret, ts, nonce, hostID, sig, body, now)
}

func (s *Server) mutationHeaders(w http.ResponseWriter, r *http.Request) bool {
	if !s.csrfOK(r) {
		writeProblem(w, http.StatusForbidden, "CSRF_REQUIRED", "Token CSRF tidak cocok.", requestID(r), false, "Kirim header X-CSRF-Token.")
		return false
	}
	key := r.Header.Get("Idempotency-Key")
	if len(key) < 8 || len(key) > 128 || strings.ContainsAny(key, " \t\r\n") {
		writeProblem(w, http.StatusBadRequest, "IDEMPOTENCY_REQUIRED", "Idempotency-Key wajib.", requestID(r), false, "Kirim kunci unik pada setiap percobaan tulis.")
		return false
	}
	return true
}

func (s *Server) csrfOK(r *http.Request) bool {
	c, err := r.Cookie("hcc_csrf")
	if err != nil || c.Value == "" {
		return false
	}
	got := r.Header.Get("X-CSRF-Token")
	if len(got) != len(c.Value) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(c.Value)) == 1
}

func (s *Server) cloudflare() provider.Status {
	s.cfMu.Lock()
	defer s.cfMu.Unlock()
	if !s.cfAt.IsZero() && s.now().Sub(s.cfAt) < cfCacheTTL {
		return s.cfStatus
	}
	st, err := s.CF.Status(context.Background())
	if err != nil {
		st = provider.Status{Configured: true, TunnelControlPlane: "unknown", Error: "pemeriksaan Cloudflare gagal", Remediation: "Tidak ada perubahan DNS.", Routes: []provider.Route{}}
	}
	if st.Routes == nil {
		st.Routes = []provider.Route{}
	}
	s.cfAt = s.now()
	s.cfStatus = st
	return st
}

func (s *Server) limited(addr string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fails == nil {
		s.fails = map[string][]time.Time{}
	}
	cutoff := s.now().Add(-10 * time.Minute)
	var keep []time.Time
	for _, t := range s.fails[addr] {
		if t.After(cutoff) {
			keep = append(keep, t)
		}
	}
	s.fails[addr] = keep
	return len(keep) >= 8
}

func (s *Server) noteFail(addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fails == nil {
		s.fails = map[string][]time.Time{}
	}
	s.fails[addr] = append(s.fails[addr], s.now())
}

func matchPassword(got, want string) bool {
	if got == "" || want == "" {
		return false
	}
	g := sha256.Sum256([]byte(got))
	w := sha256.Sum256([]byte(want))
	return subtle.ConstantTimeCompare(g[:], w[:]) == 1
}

func freshHost(h store.Host, now time.Time) bool {
	if h.LastHeartbeat == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339Nano, h.LastHeartbeat)
	if err != nil {
		t, err = time.Parse(time.RFC3339, h.LastHeartbeat)
		if err != nil {
			return false
		}
	}
	return now.Sub(t) <= staleAfter
}

func freshest(hosts []store.Host) store.Host {
	best := hosts[0]
	for _, h := range hosts[1:] {
		if h.LastHeartbeat > best.LastHeartbeat {
			best = h
		}
	}
	return best
}

func componentStatus(apps []store.App, role string, fresh bool) string {
	if !fresh {
		return "Unknown"
	}
	found, running := false, false
	for _, a := range apps {
		if classify.Role(a.Image, a.Name) != role {
			continue
		}
		found = true
		if a.State == "running" {
			running = true
		}
	}
	if !found {
		return "Unknown"
	}
	if running {
		return "Running"
	}
	return "Exited"
}

func hostView(h store.Host, status string) map[string]any {
	return map[string]any{
		"id": h.ID, "name": h.Name, "os": h.OS, "arch": h.Arch, "status": status,
		"lastCheckedAt": emptyNil(h.LastHeartbeat), "agentVersion": h.AgentVersion,
		"dockerVersion": h.DockerVersion, "dockerOk": h.DockerOK, "dockerError": h.DockerError,
		"cpuPercent": h.CPUPercent, "memUsedBytes": h.MemUsedBytes, "memTotalBytes": h.MemTotalBytes,
		"diskUsedBytes": h.DiskUsedBytes, "diskTotalBytes": h.DiskTotalBytes,
	}
}

func appView(a store.App) map[string]any {
	ownership := "Unmanaged"
	if a.Managed {
		ownership = "Managed"
	}
	return map[string]any{
		"id": a.ID, "slug": a.Slug, "name": a.Name, "hostId": a.HostID, "image": a.Image,
		"state": a.State, "stateLabel": classify.StateLabel(a.State),
		"health": a.Health, "healthLabel": classify.HealthLabel(a.Health),
		"managed": a.Managed, "ownership": ownership, "hostname": a.Hostname,
		"servicePort": a.ServicePort, "observedAt": a.ObservedAt,
		"publicUrlReady": false, "actions": actionNames(a), "projectSlug": a.ProjectSlug,
	}
}

func actionNames(a store.App) []string {
	if !a.Managed || manifest.ValidateSlug(a.ProjectSlug) != nil {
		return []string{}
	}
	return []string{"start", "stop", "restart", "delete"}
}

func jobView(j store.Job) map[string]any {
	return map[string]any{
		"id": j.ID, "type": j.Type, "appId": j.AppID, "hostId": j.HostID, "state": j.State,
		"errorCode": j.ErrorCode, "summary": j.Summary, "createdAt": j.CreatedAt, "updatedAt": j.UpdatedAt,
	}
}

func integrationView(st provider.Status) map[string]any {
	return map[string]any{
		"configured": st.Configured, "zoneReadable": st.ZoneReadable,
		"tunnelControlPlane": st.TunnelControlPlane, "error": st.Error, "remediation": st.Remediation,
	}
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

func logAge(stamp string, now time.Time) time.Duration {
	t, err := time.Parse(time.RFC3339Nano, stamp)
	if err != nil {
		return logFreshFor + time.Second
	}
	return now.Sub(t)
}

func emptyNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeProblem(w http.ResponseWriter, status int, code, message, requestID string, retryable bool, remediation string) {
	writeJSON(w, status, map[string]any{
		"code": code, "message": message, "requestId": requestID, "retryable": retryable,
		"details": map[string]any{}, "remediation": remediation,
	})
}

func newID(prefix string) string { return prefix + randomHex(12) }

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
