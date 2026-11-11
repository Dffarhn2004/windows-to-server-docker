package redact

import "regexp"

var jwt = regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)

var bearer = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9\-._~+/]+=*`)

var assigned = regexp.MustCompile(`(?i)((?:tunnel_token|token|password|passwd|secret|api[_-]?key|authorization|credential)["']?\s*[:=]\s*["']?)([^\s"',;&]{4,})`)

func Text(s string) string {
	out := assigned.ReplaceAllString(s, "${1}[redacted]")
	out = bearer.ReplaceAllString(out, "${1}[redacted]")
	out = jwt.ReplaceAllString(out, "[redacted]")
	return out
}
