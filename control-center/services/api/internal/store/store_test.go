package store

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"hcc.local/observe"
)

func TestMigrateDownAndConcurrency(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "hcc.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := st.Migrate(); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "aud_a"
			if n == 1 {
				id = "aud_b"
			}
			errCh <- st.AddAudit(id, "test", "write", "db", "ok", "req", "", now)
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := st.Down(); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Apps(nil); err == nil {
		t.Fatal("expected tables to be gone")
	}
	if err := st.Migrate(); err != nil {
		t.Fatal(err)
	}
	rep := observe.Report{HostID: "h1", HostName: "laptop", Docker: observe.DockerInfo{Available: true, Version: "29"}, Containers: []observe.Container{{
		ID: "abc123abc123abc123abc123", Name: "homelab-hello-hello-1", Image: "nginx:alpine", State: "running", Health: "none",
		Labels: map[string]string{"traefik.http.routers.homelab-hello.rule": "Host(`hello.imuii.id`)"},
	}, {
		ID: "def456def456def456def456", Name: "demo", Image: "nginx:alpine", State: "running", Health: "healthy",
		Labels: map[string]string{"hcc.managed": "true"},
	}}}
	if err := st.ApplyReport(rep, now); err != nil {
		t.Fatal(err)
	}
	all, err := st.Apps(nil)
	if err != nil || len(all) != 2 {
		t.Fatalf("%v %v", len(all), err)
	}
	yes := true
	managed, err := st.Apps(&yes)
	if err != nil || len(managed) != 1 || managed[0].Name != "demo" {
		t.Fatalf("%+v %v", managed, err)
	}
}
