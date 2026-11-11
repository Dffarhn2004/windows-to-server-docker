package sign

import (
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	body := []byte(`{"hostId":"h1"}`)
	ts := "1000"
	sig := Signature("secret", ts, "n1", "h1", body)
	now := time.Unix(1000, 0)
	if err := Verify("secret", ts, "n1", "h1", sig, body, now); err != nil {
		t.Fatal(err)
	}
	if err := Verify("secret", ts, "n1", "h1", sig, []byte(`{"hostId":"h2"}`), now); err == nil {
		t.Fatal("expected body mismatch")
	}
	if err := Verify("secret", "1", "n1", "h1", sig, body, now); err == nil {
		t.Fatal("expected expiry")
	}
}
