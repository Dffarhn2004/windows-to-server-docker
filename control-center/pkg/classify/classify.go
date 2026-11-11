package classify

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var hostRule = regexp.MustCompile("Host\\([`\"]([^`\"]+)[`\"]\\)")

func Managed(labels map[string]string) bool {
	return strings.EqualFold(labels["hcc.managed"], "true")
}

func Hostname(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if strings.Contains(k, "traefik.http.routers.") && strings.HasSuffix(k, ".rule") {
			if m := hostRule.FindStringSubmatch(labels[k]); len(m) == 2 {
				return m[1]
			}
		}
	}
	return ""
}

func ServicePort(labels map[string]string) int {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if strings.Contains(k, "loadbalancer.server.port") {
			n, _ := strconv.Atoi(strings.TrimSpace(labels[k]))
			if n > 0 && n <= 65535 {
				return n
			}
		}
	}
	return 0
}

func StateLabel(dockerState string) string {
	switch dockerState {
	case "running":
		return "Running"
	case "exited", "dead":
		return "Exited"
	case "created", "restarting":
		return "Pending"
	case "paused":
		return "Degraded"
	default:
		return "Unknown"
	}
}

func HealthLabel(health string) string {
	switch health {
	case "healthy":
		return "Healthy"
	case "unhealthy":
		return "Failed"
	case "starting":
		return "Pending"
	default:
		return "Unknown"
	}
}

func Role(image, name string) string {
	s := strings.ToLower(image + " " + name)
	switch {
	case strings.Contains(s, "cloudflared"):
		return "tunnel"
	case strings.Contains(s, "socket-proxy"):
		return "socket-proxy"
	case strings.Contains(s, "traefik"):
		return "proxy"
	default:
		return ""
	}
}
