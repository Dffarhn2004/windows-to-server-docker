package jobs

import "testing"

func TestAllowTypedComposeOnly(t *testing.T) {
	for _, kind := range []string{"observe_logs", "deploy", "start", "stop", "restart", "delete"} {
		if !Allow(kind) {
			t.Fatal(kind)
		}
	}
	for _, kind := range []string{"compose", "shell", "down-v", "privileged", ""} {
		if Allow(kind) {
			t.Fatal(kind)
		}
	}
}
