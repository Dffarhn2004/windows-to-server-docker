package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"hcc.local/redact"
)

type DockerProvider interface {
	ReadOnly() bool
}

type ObservedDocker struct{}

func (ObservedDocker) ReadOnly() bool { return true }

type Route struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Proxied   bool   `json:"proxied"`
	Ownership string `json:"ownership"`
	Protected bool   `json:"protected"`
}

type Status struct {
	Configured         bool    `json:"configured"`
	ZoneReadable       bool    `json:"zoneReadable"`
	TunnelControlPlane string  `json:"tunnelControlPlane"`
	LastCheckedAt      string  `json:"lastCheckedAt"`
	Error              string  `json:"error,omitempty"`
	Remediation        string  `json:"remediation,omitempty"`
	Routes             []Route `json:"routes"`
	Truncated          bool    `json:"truncated"`
}

type CloudflareProvider interface {
	Status(ctx context.Context) (Status, error)
}

type Unconfigured struct{}

func (Unconfigured) Status(context.Context) (Status, error) {
	return Status{
		Configured:         false,
		TunnelControlPlane: "not_configured",
		LastCheckedAt:      time.Now().UTC().Format(time.RFC3339),
		Routes:             []Route{},
		Remediation:        "Token manajemen Cloudflare belum diisi. Dashboard tetap baca-saja dan tidak menulis DNS.",
	}, nil
}

type StaticCloudflare struct{ Value Status }

func (s StaticCloudflare) Status(context.Context) (Status, error) { return s.Value, nil }

type HTTPCloudflare struct {
	token   string
	account string
	zone    string
	tunnel  string
	client  *http.Client
}

func NewHTTP(token, account, zone, tunnel string) *HTTPCloudflare {
	return &HTTPCloudflare{
		token: token, account: account, zone: zone, tunnel: tunnel,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *HTTPCloudflare) Status(ctx context.Context) (Status, error) {
	st := Status{
		Configured:         true,
		TunnelControlPlane: "unknown",
		LastCheckedAt:      time.Now().UTC().Format(time.RFC3339),
		Routes:             []Route{},
	}
	body, pages, err := c.get(ctx, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?per_page=100", url.PathEscape(c.zone)))
	if err != nil {
		st.Error = err.Error()
		st.Remediation = "Periksa izin baca DNS pada token manajemen. Tidak ada record yang diubah."
		return st, nil
	}
	st.Truncated = pages > 1
	var recs []struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Content string `json:"content"`
		Proxied bool   `json:"proxied"`
	}
	if err := decodeResult(body, &recs); err != nil {
		st.Error = "respons DNS tidak terbaca"
		return st, nil
	}
	st.ZoneReadable = true
	for _, rec := range recs {
		st.Routes = append(st.Routes, Route{
			Name: rec.Name, Type: rec.Type, Content: rec.Content, Proxied: rec.Proxied,
			Ownership: "external", Protected: ProtectedName(rec.Name),
		})
	}
	if c.tunnel == "" {
		st.Remediation = "HCC_CF_TUNNEL_ID kosong, jadi kesehatan control plane tetap Unknown."
		return st, nil
	}
	body, _, err = c.get(ctx, fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/cfd_tunnel/%s", url.PathEscape(c.account), url.PathEscape(c.tunnel)))
	if err != nil {
		st.Error = err.Error()
		st.TunnelControlPlane = "unknown"
		return st, nil
	}
	var tun struct {
		Status string `json:"status"`
	}
	if err := decodeResult(body, &tun); err != nil {
		st.TunnelControlPlane = "unknown"
		return st, nil
	}
	switch strings.ToLower(tun.Status) {
	case "healthy":
		st.TunnelControlPlane = "healthy"
	case "degraded":
		st.TunnelControlPlane = "degraded"
	case "down", "inactive":
		st.TunnelControlPlane = strings.ToLower(tun.Status)
	default:
		st.TunnelControlPlane = "unknown"
	}
	return st, nil
}

func (c *HTTPCloudflare) get(ctx context.Context, rawURL string) (json.RawMessage, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	res, err := c.do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, 0, err
	}
	if res.StatusCode >= 400 {
		return nil, 0, fmt.Errorf("cloudflare HTTP %d", res.StatusCode)
	}
	var env struct {
		Success bool            `json:"success"`
		Result  json.RawMessage `json:"result"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
		ResultInfo struct {
			TotalPages int `json:"total_pages"`
		} `json:"result_info"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		return nil, 0, fmt.Errorf("cloudflare JSON")
	}
	if !env.Success {
		msg := "cloudflare menolak permintaan baca"
		if len(env.Errors) > 0 {
			msg = redact.Text(env.Errors[0].Message)
		}
		return nil, 0, fmt.Errorf("%s", msg)
	}
	return env.Result, env.ResultInfo.TotalPages, nil
}

func (c *HTTPCloudflare) do(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return nil, fmt.Errorf("cloudflare write blocked")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	return c.client.Do(req)
}

func decodeResult(raw json.RawMessage, dest any) error {
	return json.Unmarshal(raw, dest)
}

func ProtectedName(name string) bool {
	n := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
	switch n {
	case "*", "@", "imuii.id", "hello.imuii.id", "waha-v2.imuii.id", "api.imuii.id":
		return true
	}
	return strings.HasPrefix(n, "*")
}
