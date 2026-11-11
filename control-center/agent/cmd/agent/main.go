package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"

	"hcc.local/agent/internal/compose"
	"hcc.local/agent/internal/docker"
	"hcc.local/agent/internal/jobs"
	"hcc.local/observe"
	"hcc.local/sign"
)

func main() {
	api := flag.String("api", env("HCC_API_URL", "http://127.0.0.1:8080"), "control API base URL")
	hostID := flag.String("host-id", env("HCC_HOST_ID", "local-windows"), "host id")
	hostName := flag.String("host-name", env("HCC_HOST_NAME", "windows-host"), "display name")
	interval := flag.Duration("interval", 10*time.Second, "poll interval")
	once := flag.Bool("once", false, "send one observation and exit")
	projects := flag.String("projects", env("HCC_PROJECTS_DIR", filepath.Join("..", "..", "managed-projects")), "managed compose directory")
	flag.Parse()
	secret := loadSecret()
	if secret == "" {
		log.Fatal(`{"msg":"agent secret missing"}`)
	}
	if !strings.HasPrefix(*api, "http://127.0.0.1") && !strings.HasPrefix(*api, "http://localhost") && os.Getenv("HCC_AGENT_ALLOW_REMOTE") != "1" {
		log.Fatal(`{"msg":"refusing non-loopback API URL"}`)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	run := func() error {
		report := collect(*hostID, *hostName)
		body, jobsOut, err := post(client, *api, secret, *hostID, "/agent/v1/poll", report)
		if err != nil {
			return err
		}
		_ = body
		for _, job := range jobsOut {
			result := runJob(job, *projects)
			if _, _, err := post(client, *api, secret, *hostID, "/agent/v1/jobs/"+job.ID+"/result", result); err != nil {
				log.Printf(`{"msg":"result failed","job":%q,"err":%q}`, job.ID, err.Error())
			}
		}
		return nil
	}
	if err := run(); err != nil {
		log.Printf(`{"msg":"poll failed","err":%q}`, err.Error())
		if *once {
			os.Exit(1)
		}
	}
	if *once {
		return
	}
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for range ticker.C {
		if err := run(); err != nil {
			log.Printf(`{"msg":"poll failed","err":%q}`, err.Error())
		}
	}
}

func collect(hostID, hostName string) observe.Report {
	ctx, cancel := docker.WithTimeout()
	defer cancel()
	info := docker.Discover(ctx)
	var containers []observe.Container
	if info.Available {
		found, err := docker.Containers(ctx)
		if err != nil {
			info.Available = false
			info.Error = "inventaris container gagal"
			containers = []observe.Container{}
		} else {
			containers = found
		}
	}
	rep := observe.Report{
		SchemaVersion: 1, HostID: hostID, HostName: hostName, AgentVersion: "0.1.0",
		OS: runtime.GOOS, Arch: runtime.GOARCH, Docker: info, Containers: containers,
	}
	if pct, err := cpu.Percent(200*time.Millisecond, false); err == nil && len(pct) > 0 {
		rep.CPUPercent = pct[0]
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		rep.MemUsedBytes = int64(vm.Used)
		rep.MemTotalBytes = int64(vm.Total)
	}
	path := "/"
	if runtime.GOOS == "windows" {
		path = `C:\`
	}
	if du, err := disk.Usage(path); err == nil {
		rep.DiskUsedBytes = int64(du.Used)
		rep.DiskTotalBytes = int64(du.Total)
	}
	return rep
}

func runJob(job observe.AssignedJob, projects string) observe.JobResult {
	if !jobs.Allow(job.Type) {
		return observe.JobResult{State: "failed", Code: "DOCKER_POLICY_DENIED", Summary: "tipe job tidak diizinkan"}
	}
	if job.Type == "observe_logs" {
		ctx, cancel := docker.WithTimeout()
		defer cancel()
		text, err := docker.Logs(ctx, job.ContainerID, job.Tail)
		if err != nil {
			return observe.JobResult{State: "failed", Code: "DOCKER_POLICY_DENIED", Summary: err.Error()}
		}
		return observe.JobResult{State: "succeeded", Summary: "log terbaca", Logs: text}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	return compose.Execute(ctx, projects, job, func(ctx context.Context, args []string) error {
		cmd := exec.CommandContext(ctx, "docker", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf(`{"msg":"compose failed","bytes":%d}`, len(out))
			return err
		}
		return nil
	})
}

func post(client *http.Client, api, secret, hostID, path string, payload any) ([]byte, []observe.AssignedJob, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	ts := fmt.Sprintf("%d", time.Now().Unix())
	nonce := fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(api, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-HCC-Host", hostID)
	req.Header.Set("X-HCC-Timestamp", ts)
	req.Header.Set("X-HCC-Nonce", nonce)
	req.Header.Set("X-HCC-Signature", sign.Signature(secret, ts, nonce, hostID, body))
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return nil, nil, err
	}
	if res.StatusCode >= 300 {
		return raw, nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	var parsed observe.PollResponse
	_ = json.Unmarshal(raw, &parsed)
	if parsed.Jobs == nil {
		parsed.Jobs = []observe.AssignedJob{}
	}
	return raw, parsed.Jobs, nil
}

func loadSecret() string {
	if path := os.Getenv("HCC_AGENT_SECRET_FILE"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			log.Printf(`{"msg":"secret file unreadable"}`)
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	return os.Getenv("HCC_AGENT_SECRET")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
