package redact

import (
	"strings"
	"testing"
)

func TestText(t *testing.T) {
	in := "TUNNEL_TOKEN=supersecretvalue123 Bearer abcdefghijklmnop eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	out := Text(in)
	for _, secret := range []string{"supersecretvalue123", "abcdefghijklmnop", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"} {
		if strings.Contains(out, secret) {
			t.Fatalf("secret leaked: %s in %s", secret, out)
		}
	}
	if !strings.Contains(out, "[redacted]") {
		t.Fatal(out)
	}
}
