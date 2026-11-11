package manifest

import (
	"strings"
	"testing"
)

func TestRenderInternalNginx(t *testing.T) {
	file, err := Render(Spec{Slug: "hcc-smoke-internal", DisplayName: "Smoke", Template: TemplateNginx})
	if err != nil {
		t.Fatal(err)
	}
	if file.Project != "hcc-hcc-smoke-internal" {
		t.Fatal(file.Project)
	}
	again, err := Render(Spec{Slug: "hcc-smoke-internal", DisplayName: "Smoke", Template: TemplateNginx})
	if err != nil || again.YAML != file.YAML || again.Hash != file.Hash {
		t.Fatal("render tidak deterministik")
	}
	for _, absent := range []string{"ports:", "traefik", "homelab_edge", "privileged", "volumes:", "docker.sock"} {
		if strings.Contains(file.YAML, absent) {
			t.Fatal(absent)
		}
	}
	if !strings.Contains(file.YAML, `image: nginx:alpine`) || !strings.Contains(file.YAML, `hcc.managed: "true"`) {
		t.Fatal(file.YAML)
	}
}

func TestRejectDangerous(t *testing.T) {
	base, err := Render(Spec{Slug: "demo-app", DisplayName: "Demo", Template: TemplateNginx})
	if err != nil {
		t.Fatal(err)
	}
	for _, extra := range []string{
		"\n    privileged: true\n",
		"\n    ports:\n      - 80:80\n",
		"\n    volumes:\n      - /var/run/docker.sock:/var/run/docker.sock\n",
		"\n    network_mode: host\n",
	} {
		if err := Reject(base.YAML + extra); err == nil {
			t.Fatalf("should reject %q", extra)
		}
	}
	if _, err := Render(Spec{Slug: "../etc", DisplayName: "X", Template: TemplateNginx}); err == nil {
		t.Fatal("slug path")
	}
	if _, err := Render(Spec{Slug: "demo-app", DisplayName: "X", Template: "custom"}); err == nil {
		t.Fatal("template")
	}
}
