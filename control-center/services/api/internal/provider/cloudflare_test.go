package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnconfiguredDoesNotCallNetwork(t *testing.T) {
	st, err := Unconfigured{}.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Configured || st.TunnelControlPlane != "not_configured" {
		t.Fatalf("%+v", st)
	}
}

func TestHTTPReadOnlyAndHealthyMapping(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		if r.Method != http.MethodGet {
			t.Errorf("write method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "dns_records"):
			_, _ = io.WriteString(w, `{"success":true,"result":[{"name":"hello.imuii.id","type":"CNAME","content":"tunnel","proxied":true},{"name":"*.imuii.id","type":"A","content":"203.0.113.5","proxied":true}],"result_info":{"total_pages":1}}`)
		default:
			_, _ = io.WriteString(w, `{"success":true,"result":{"status":"healthy","name":"homelab-docker"}}`)
		}
	}))
	defer srv.Close()
	c := NewHTTP("test-token", "acct", "zone", "tun")
	c.client = srv.Client()
	c.client.Transport = rewriteTransport{base: srv.Client().Transport, host: srv.URL}
	st, err := c.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !st.ZoneReadable || st.TunnelControlPlane != "healthy" {
		t.Fatalf("%+v", st)
	}
	if len(st.Routes) != 2 || !st.Routes[0].Protected || st.Routes[0].Ownership != "external" {
		t.Fatalf("%+v", st.Routes)
	}
	if !st.Routes[1].Protected {
		t.Fatal("wildcard must be protected")
	}
	req, _ := http.NewRequest(http.MethodPut, srv.URL, nil)
	if _, err := c.do(req); err == nil {
		t.Fatal("PUT must be blocked")
	}
}

type rewriteTransport struct {
	base http.RoundTripper
	host string
}

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(r.host, "http://")
	base := r.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}
