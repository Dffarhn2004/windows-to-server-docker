package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"hcc.local/api/internal/provider"
	"hcc.local/api/internal/store"
	"hcc.local/sign"
)

func TestReadOnlyContract(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "hcc.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(); err != nil {
		t.Fatal(err)
	}
	cur := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	srv := &Server{
		Store:    st,
		Password: "test-pass",
		Secret:   "agent-secret",
		Docker:   provider.ObservedDocker{},
		Now:      func() time.Time { return cur },
	}
	if !srv.Docker.ReadOnly() {
		t.Fatal("docker provider must stay read-only")
	}
	h := srv.Handler()

	rec := call(h, http.MethodGet, "/api/v1/health", nil, nil)
	if rec.Code != 200 || strings.Contains(rec.Body.String(), "agent-secret") || strings.Contains(rec.Body.String(), "test-pass") {
		t.Fatalf("health %d %s", rec.Code, rec.Body.String())
	}
	if call(h, http.MethodGet, "/api/v1/overview", nil, nil).Code != 401 {
		t.Fatal("anonymous overview")
	}
	if call(h, http.MethodPost, "/api/v1/apps", []byte(`{}`), nil).Code != 401 {
		t.Fatal("anonymous deploy")
	}

	login := call(h, http.MethodPost, "/api/v1/session", []byte(`{"password":"test-pass"}`), nil)
	if login.Code != 200 {
		t.Fatal(login.Body.String())
	}
	cookies := login.Result().Cookies()
	csrf := cookieValue(cookies, "hcc_csrf")
	auth := map[string]string{"X-CSRF-Token": csrf, "Idempotency-Key": "idem-test-01"}

	ov := call(h, http.MethodGet, "/api/v1/overview", nil, cookies)
	if ov.Code != 200 || !strings.Contains(ov.Body.String(), `"status":"Unknown"`) {
		t.Fatalf("empty overview %s", ov.Body.String())
	}
	before, _ := st.CountJobs()
	denied := call(h, http.MethodPost, "/api/v1/apps", []byte(`{"slug":"x"}`), cookies, auth)
	if denied.Code != 400 {
		t.Fatalf("invalid deploy %d %s", denied.Code, denied.Body.String())
	}
	after, _ := st.CountJobs()
	if before != after {
		t.Fatal("invalid spec created a job")
	}

	report := []byte(`{"schemaVersion":1,"hostId":"local-windows","hostName":"laptop","agentVersion":"0.1.0","os":"windows","arch":"amd64","docker":{"available":true,"version":"29.8.2"},"containers":[{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","name":"homelab-hello-hello-1","image":"nginx:alpine","state":"running","health":"none","labels":{"traefik.http.routers.homelab-hello.rule":"Host(` + "`hello.imuii.id`" + `)"},"networks":["homelab_edge"]},{"id":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","name":"hcc-demo","image":"nginx:alpine","state":"running","health":"healthy","labels":{"hcc.managed":"true","hcc.project-id":"demo-app"},"networks":["homelab_services"]}]}`)
	poll := agentCall(h, http.MethodPost, "/agent/v1/poll", report, "local-windows", "agent-secret", "nonce-1", cur)
	if poll.Code != 200 {
		t.Fatal(poll.Body.String())
	}
	replay := agentCall(h, http.MethodPost, "/agent/v1/poll", report, "local-windows", "agent-secret", "nonce-1", cur)
	if replay.Code != 401 || !strings.Contains(replay.Body.String(), "REPLAY") {
		t.Fatalf("replay %d %s", replay.Code, replay.Body.String())
	}
	bad := agentCall(h, http.MethodPost, "/agent/v1/poll", report, "local-windows", "other-secret", "nonce-2", cur)
	if bad.Code != 401 {
		t.Fatal("bad signature should be denied")
	}

	apps := call(h, http.MethodGet, "/api/v1/apps", nil, cookies)
	if !strings.Contains(apps.Body.String(), `"ownership":"Unmanaged"`) || !strings.Contains(apps.Body.String(), `"ownership":"Managed"`) {
		t.Fatal(apps.Body.String())
	}
	if !strings.Contains(apps.Body.String(), `"healthLabel":"Unknown"`) || !strings.Contains(apps.Body.String(), `"healthLabel":"Healthy"`) {
		t.Fatal(apps.Body.String())
	}
	var listed struct {
		Apps []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Ownership string `json:"ownership"`
		} `json:"apps"`
	}
	_ = json.Unmarshal(apps.Body.Bytes(), &listed)
	var helloID, demoID string
	for _, a := range listed.Apps {
		if a.Name == "homelab-hello-hello-1" {
			helloID = a.ID
		}
		if a.Name == "hcc-demo" {
			demoID = a.ID
		}
	}
	unmanaged := call(h, http.MethodPost, "/api/v1/apps/"+helloID+"/actions", []byte(`{"action":"stop"}`), cookies, auth)
	if unmanaged.Code != 409 || !strings.Contains(unmanaged.Body.String(), "UNMANAGED_APP") {
		t.Fatalf("unmanaged %d %s", unmanaged.Code, unmanaged.Body.String())
	}
	managed := call(h, http.MethodPost, "/api/v1/apps/"+demoID+"/actions", []byte(`{"action":"stop"}`), cookies, map[string]string{"X-CSRF-Token": csrf, "Idempotency-Key": "idem-stop-01"})
	if managed.Code != 202 || !strings.Contains(managed.Body.String(), `"action":"stop"`) {
		t.Fatalf("managed %d %s", managed.Code, managed.Body.String())
	}
	deploy := call(h, http.MethodPost, "/api/v1/apps", []byte(`{"slug":"smoke-internal","displayName":"Smoke","template":"nginx"}`), cookies, map[string]string{"X-CSRF-Token": csrf, "Idempotency-Key": "idem-deploy-01"})
	if deploy.Code != 202 || !strings.Contains(deploy.Body.String(), `"state":"queued"`) {
		t.Fatalf("deploy %d %s", deploy.Code, deploy.Body.String())
	}
	var created struct {
		JobID string `json:"jobId"`
	}
	_ = json.Unmarshal(deploy.Body.Bytes(), &created)
	again := call(h, http.MethodPost, "/api/v1/apps", []byte(`{"slug":"smoke-internal","displayName":"Smoke","template":"nginx"}`), cookies, map[string]string{"X-CSRF-Token": csrf, "Idempotency-Key": "idem-deploy-01"})
	var replayed struct {
		JobID string `json:"jobId"`
	}
	_ = json.Unmarshal(again.Body.Bytes(), &replayed)
	if again.Code != 202 || replayed.JobID == "" || replayed.JobID != created.JobID {
		t.Fatalf("idempotency %d %s", again.Code, again.Body.String())
	}
	preview := call(h, http.MethodPost, "/api/v1/apps/validate", []byte(`{"slug":"other-demo","displayName":"Lain","template":"nginx"}`), cookies, map[string]string{"X-CSRF-Token": csrf, "Idempotency-Key": "idem-validate-01"})
	if preview.Code != 200 || strings.Contains(preview.Body.String(), "traefik") || strings.Contains(preview.Body.String(), "ports:") {
		t.Fatalf("preview %d %s", preview.Code, preview.Body.String())
	}

	logs := call(h, http.MethodGet, "/api/v1/apps/"+helloID+"/logs?tail=20", nil, cookies)
	if logs.Code != 200 || !strings.Contains(logs.Body.String(), `"pending":true`) {
		t.Fatal(logs.Body.String())
	}
	var logBody struct {
		JobID string `json:"jobId"`
	}
	_ = json.Unmarshal(logs.Body.Bytes(), &logBody)
	result := []byte(`{"state":"succeeded","summary":"ok","logs":"TUNNEL_TOKEN=supersecretvalue123\nhello"}`)
	done := agentCall(h, http.MethodPost, "/agent/v1/jobs/"+logBody.JobID+"/result", result, "local-windows", "agent-secret", "nonce-3", cur)
	if done.Code != 200 {
		t.Fatal(done.Body.String())
	}
	logsAgain := call(h, http.MethodGet, "/api/v1/apps/"+helloID+"/logs?tail=20", nil, cookies)
	if strings.Contains(logsAgain.Body.String(), "supersecretvalue123") || !strings.Contains(logsAgain.Body.String(), "[redacted]") {
		t.Fatal(logsAgain.Body.String())
	}

	cur = cur.Add(2 * time.Minute)
	stale := call(h, http.MethodGet, "/api/v1/overview", nil, cookies)
	if !strings.Contains(stale.Body.String(), `"status":"Unknown"`) || strings.Contains(stale.Body.String(), `"status":"Healthy"`) {
		t.Fatalf("stale host must not look healthy: %s", stale.Body.String())
	}
	audit := call(h, http.MethodGet, "/api/v1/audit", nil, cookies)
	if audit.Code != 200 || !strings.Contains(audit.Body.String(), "login") {
		t.Fatal(audit.Body.String())
	}
}

func call(h http.Handler, method, path string, body []byte, cookies []*http.Cookie, headers ...map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(string(body)))
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if len(headers) > 0 {
		for k, v := range headers[0] {
			req.Header.Set(k, v)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func agentCall(h http.Handler, method, path string, body []byte, host, secret, nonce string, now time.Time) *httptest.ResponseRecorder {
	ts := strconv.FormatInt(now.Unix(), 10)
	req := httptest.NewRequest(method, path, strings.NewReader(string(body)))
	req.Header.Set("X-HCC-Host", host)
	req.Header.Set("X-HCC-Timestamp", ts)
	req.Header.Set("X-HCC-Nonce", nonce)
	req.Header.Set("X-HCC-Signature", sign.Signature(secret, ts, nonce, host, body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func cookieValue(cookies []*http.Cookie, name string) string {
	for _, c := range cookies {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}
