package compose

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hcc.local/observe"
)

func TestDownDoesNotDeleteVolumes(t *testing.T) {
	args, err := Command("down", "hcc-smoke-internal", filepath.Join("managed-projects", "smoke-internal", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "-v") || strings.Contains(joined, "--volumes") {
		t.Fatal(joined)
	}
	if args[0] != "compose" || args[len(args)-1] != "--remove-orphans" {
		t.Fatal(args)
	}
}

func TestExecuteRejectsEscapeAndWritesInternalFile(t *testing.T) {
	root := t.TempDir()
	var got []string
	result := Execute(context.Background(), root, observe.AssignedJob{
		Type: "deploy", Slug: "smoke-internal", DisplayName: "Smoke", Template: "nginx",
	}, func(_ context.Context, args []string) error {
		got = args
		return nil
	})
	if result.State != "succeeded" {
		t.Fatal(result.Summary)
	}
	if strings.Join(got, " ") != strings.Join([]string{"compose", "-p", "hcc-smoke-internal", "-f", filepath.Join(root, "smoke-internal", "compose.yaml"), "up", "-d"}, " ") {
		t.Fatal(got)
	}
	raw, err := os.ReadFile(filepath.Join(root, "smoke-internal", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "traefik") || strings.Contains(string(raw), "ports:") {
		t.Fatal(string(raw))
	}
	escaped := Execute(context.Background(), root, observe.AssignedJob{Type: "deploy", Slug: "../evil", DisplayName: "X", Template: "nginx"}, func(context.Context, []string) error {
		t.Fatal("docker harus tidak dipanggil")
		return nil
	})
	if escaped.Code != "DOCKER_POLICY_DENIED" {
		t.Fatal(escaped)
	}
}
