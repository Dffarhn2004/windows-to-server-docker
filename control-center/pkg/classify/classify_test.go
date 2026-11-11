package classify

import "testing"

func TestManagedAndHostname(t *testing.T) {
	labels := map[string]string{
		"hcc.managed": "true",
		"traefik.http.routers.homelab-hello.rule": "Host(`hello.imuii.id`)",
		"traefik.http.services.homelab-hello.loadbalancer.server.port": "80",
	}
	if !Managed(labels) {
		t.Fatal("expected managed")
	}
	if Hostname(labels) != "hello.imuii.id" {
		t.Fatal(Hostname(labels))
	}
	if ServicePort(labels) != 80 {
		t.Fatal(ServicePort(labels))
	}
	foreign := map[string]string{"traefik.enable": "true"}
	if Managed(foreign) {
		t.Fatal("foreign must stay unmanaged")
	}
	if StateLabel("running") != "Running" || StateLabel("weird") != "Unknown" {
		t.Fatal("state")
	}
	if HealthLabel("") != "Unknown" || HealthLabel("healthy") != "Healthy" {
		t.Fatal("health")
	}
	if Role("traefik:v3.7.17", "homelab-infra-traefik-1") != "proxy" {
		t.Fatal("proxy")
	}
	if Role("cloudflare/cloudflared:2026.10.7", "homelab-infra-cloudflared-1") != "tunnel" {
		t.Fatal("tunnel")
	}
}
