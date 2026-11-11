package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"hcc.local/observe"
	"hcc.local/redact"
)

const inspectFormat = `{{.Id}}	{{.Name}}	{{.Config.Image}}	{{json .State}}	{{json .Config.Labels}}	{{json .NetworkSettings.Networks}}`

func Discover(ctx context.Context) observe.DockerInfo {
	out, err := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}").Output()
	if err != nil {
		return observe.DockerInfo{Available: false, Error: "Docker CLI tidak tersedia atau daemon tidak menjawab"}
	}
	return observe.DockerInfo{Available: true, Version: strings.TrimSpace(string(out))}
}

func Containers(ctx context.Context) ([]observe.Container, error) {
	idsRaw, err := exec.CommandContext(ctx, "docker", "ps", "-aq").Output()
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, line := range strings.Split(string(idsRaw), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			ids = append(ids, line)
		}
	}
	if len(ids) == 0 {
		return []observe.Container{}, nil
	}
	if len(ids) > 200 {
		ids = ids[:200]
	}
	args := append([]string{"inspect", "-f", inspectFormat}, ids...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var containers []observe.Container
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 6)
		if len(parts) != 6 {
			continue
		}
		var state struct {
			Status string `json:"Status"`
			Health *struct {
				Status string `json:"Status"`
			} `json:"Health"`
		}
		_ = json.Unmarshal([]byte(parts[3]), &state)
		labels := map[string]string{}
		if parts[4] != "" && parts[4] != "null" {
			_ = json.Unmarshal([]byte(parts[4]), &labels)
		}
		labels = safeLabels(labels)
		networks := networkNames(parts[5])
		health := "none"
		if state.Health != nil && state.Health.Status != "" {
			health = state.Health.Status
		}
		containers = append(containers, observe.Container{
			ID: parts[0], Name: strings.TrimPrefix(parts[1], "/"), Image: parts[2],
			State: state.Status, Health: health, Labels: labels, Networks: networks,
		})
	}
	return containers, nil
}

func Logs(ctx context.Context, containerID string, tail int) (string, error) {
	if !safeContainer(containerID) {
		return "", fmt.Errorf("container id ditolak")
	}
	if tail < 1 {
		tail = 1
	}
	if tail > 500 {
		tail = 500
	}
	cmd := exec.CommandContext(ctx, "docker", "logs", "--tail", fmt.Sprintf("%d", tail), containerID)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	_ = cmd.Run()
	text := buf.String()
	if len(text) > 64*1024 {
		text = text[len(text)-64*1024:]
	}
	return redact.Text(text), nil
}

func safeContainer(id string) bool {
	if id == "" || len(id) > 128 || strings.Contains(id, "..") || strings.ContainsAny(id, " /\\'\"$`|&;") {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' || r == ':' {
			continue
		}
		return false
	}
	return true
}

func safeLabels(in map[string]string) map[string]string {
	out := map[string]string{}
	n := 0
	for k, v := range in {
		if n >= 40 {
			break
		}
		lk := strings.ToLower(k)
		if strings.Contains(lk, "token") || strings.Contains(lk, "password") || strings.Contains(lk, "secret") {
			continue
		}
		if len(v) > 300 {
			v = v[:300]
		}
		out[k] = v
		n++
	}
	return out
}

func networkNames(raw string) []string {
	if raw == "" || raw == "null" {
		return []string{}
	}
	var nets map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &nets); err != nil {
		return []string{}
	}
	names := make([]string, 0, len(nets))
	for name := range nets {
		names = append(names, name)
	}
	return names
}

func WithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 20*time.Second)
}
