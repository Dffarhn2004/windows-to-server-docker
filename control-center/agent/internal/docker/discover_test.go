package docker

import "testing"

func TestSafeContainer(t *testing.T) {
	if !safeContainer("abc123def456") {
		t.Fatal("hex id")
	}
	for _, bad := range []string{"", "abc;rm", "a b", "../sock", "a|b", "$(id)"} {
		if safeContainer(bad) {
			t.Fatal(bad)
		}
	}
}
