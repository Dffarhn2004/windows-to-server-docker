package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeStaysLoopback(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "infra", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{`"127.0.0.1:8080:8080"`, `"127.0.0.1:3000:3000"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s", want)
		}
	}
	for _, banned := range []string{"docker.sock", "traefik.enable", "homelab_edge", `"8080:8080"`, `"3000:3000"`} {
		if strings.Contains(text, banned) {
			t.Fatalf("compose contains %s", banned)
		}
	}
}

func TestGoSourcesDoNotNameConnectorToken(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && (info.Name() == "node_modules" || info.Name() == ".next") {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "TUNNEL_TOKEN") {
			t.Errorf("connector token name appears in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
