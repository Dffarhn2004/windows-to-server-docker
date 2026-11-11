package compose

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hcc.local/manifest"
	"hcc.local/observe"
)

func Dir(root, slug string) (string, error) {
	if err := manifest.ValidateSlug(slug); err != nil {
		return "", err
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	dir := filepath.Clean(filepath.Join(root, slug))
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || strings.Contains(rel, string(filepath.Separator)) {
		return "", errors.New("path proyek di luar sandbox")
	}
	return dir, nil
}

func Command(action, project, file string) ([]string, error) {
	if strings.Contains(action, "-") || strings.ContainsAny(project, " \t") || strings.ContainsAny(file, "\n") {
		return nil, errors.New("argumen ditolak")
	}
	base := []string{"compose", "-p", project, "-f", file}
	switch action {
	case "up":
		return append(base, "up", "-d"), nil
	case "start", "stop", "restart":
		return append(base, action), nil
	case "down":
		return append(base, "down", "--remove-orphans"), nil
	default:
		return nil, errors.New("perintah tidak ada dalam daftar izin")
	}
}

func Execute(ctx context.Context, root string, job observe.AssignedJob, run func(context.Context, []string) error) observe.JobResult {
	dir, err := Dir(root, job.Slug)
	if err != nil {
		return deny(err.Error())
	}
	file := filepath.Join(dir, "compose.yaml")
	switch job.Type {
	case "deploy":
		rendered, err := manifest.Render(manifest.Spec{Slug: job.Slug, DisplayName: job.DisplayName, Template: job.Template})
		if err != nil {
			return deny(err.Error())
		}
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return deny("folder proyek tidak bisa dibuat")
		}
		if err := os.WriteFile(file, []byte(rendered.YAML), 0o640); err != nil {
			return deny("compose tidak bisa ditulis")
		}
		if err := runCommand(ctx, "up", rendered.Project, file, run); err != nil {
			return observe.JobResult{State: "failed", Code: "DEPLOY_FAILED", Summary: err.Error()}
		}
		return observe.JobResult{State: "succeeded", Summary: "stack internal berjalan"}
	case "start", "stop", "restart", "delete":
		raw, err := os.ReadFile(file)
		if err != nil {
			return deny("compose milik HCC tidak ditemukan")
		}
		if err := manifest.Reject(string(raw)); err != nil {
			return deny(err.Error())
		}
		project := manifest.ProjectName(job.Slug)
		action := job.Type
		if job.Type == "delete" {
			action = "down"
		}
		if err := runCommand(ctx, action, project, file, run); err != nil {
			return observe.JobResult{State: "failed", Code: "DEPLOY_FAILED", Summary: err.Error()}
		}
		if job.Type == "delete" {
			if err := os.RemoveAll(dir); err != nil {
				return observe.JobResult{State: "failed", Code: "ROLLBACK_INCOMPLETE", Summary: "stack berhenti, folder spec masih ada"}
			}
			return observe.JobResult{State: "succeeded", Summary: "stack dihapus, volume tidak dihapus"}
		}
		return observe.JobResult{State: "succeeded", Summary: "aksi " + job.Type + " selesai"}
	default:
		return deny("tipe job tidak diizinkan")
	}
}

func runCommand(ctx context.Context, action, project, file string, run func(context.Context, []string) error) error {
	args, err := Command(action, project, file)
	if err != nil {
		return err
	}
	for _, arg := range args {
		if arg == "-v" || arg == "--volumes" || strings.Contains(arg, "docker.sock") {
			return errors.New("argumen terlarang")
		}
	}
	if err := run(ctx, args); err != nil {
		return fmt.Errorf("docker compose gagal")
	}
	return nil
}

func deny(summary string) observe.JobResult {
	return observe.JobResult{State: "failed", Code: "DOCKER_POLICY_DENIED", Summary: summary}
}
