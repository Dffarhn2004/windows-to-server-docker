package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	TemplateNginx = "nginx"
	ImageNginx    = "nginx:alpine"
	NetworkName   = "homelab_services"
	ServiceName   = "app"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?$`)

type Spec struct {
	Slug        string
	DisplayName string
	Template    string
}

type File struct {
	YAML    string
	Hash    string
	Project string
	Image   string
	Network string
}

func Normalize(spec Spec) (Spec, error) {
	spec.Slug = strings.TrimSpace(spec.Slug)
	spec.DisplayName = strings.TrimSpace(spec.DisplayName)
	spec.Template = strings.TrimSpace(spec.Template)
	if err := ValidateSlug(spec.Slug); err != nil {
		return Spec{}, err
	}
	if spec.DisplayName == "" || len(spec.DisplayName) > 80 || strings.ContainsAny(spec.DisplayName, "\r\n") {
		return Spec{}, errors.New("display name tidak valid")
	}
	if spec.Template != TemplateNginx {
		return Spec{}, errors.New("template tidak ada dalam daftar izin")
	}
	return spec, nil
}

func ValidateSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return errors.New("slug harus huruf kecil, angka, atau tanda hubung, panjang 2 sampai 40")
	}
	return nil
}

func ProjectName(slug string) string {
	return "hcc-" + slug
}

func Render(spec Spec) (File, error) {
	spec, err := Normalize(spec)
	if err != nil {
		return File{}, err
	}
	sum := sha256.Sum256([]byte(spec.Slug + "\n" + spec.DisplayName + "\n" + spec.Template + "\n" + ImageNginx + "\n80\n" + NetworkName))
	hash := hex.EncodeToString(sum[:])
	project := ProjectName(spec.Slug)
	yaml := fmt.Sprintf(`name: %s
services:
  %s:
    image: %s
    restart: unless-stopped
    networks:
      - services
    labels:
      hcc.managed: "true"
      hcc.project-id: %q
      hcc.spec-hash: %q
networks:
  services:
    external: true
    name: %s
`, project, ServiceName, ImageNginx, spec.Slug, hash, NetworkName)
	if err := Reject(yaml); err != nil {
		return File{}, err
	}
	return File{YAML: yaml, Hash: hash, Project: project, Image: ImageNginx, Network: NetworkName}, nil
}

func Reject(yaml string) error {
	lower := strings.ToLower(yaml)
	banned := []string{
		"privileged",
		"network_mode",
		"pid:",
		"devices:",
		"cap_add",
		"docker.sock",
		"ports:",
		"traefik",
		"homelab_edge",
		"volumes:",
		"/var/run",
	}
	for _, word := range banned {
		if strings.Contains(lower, word) {
			return fmt.Errorf("compose ditolak: %s", word)
		}
	}
	if strings.Contains(yaml, "..") {
		return errors.New("compose ditolak: path relatif")
	}
	if !strings.Contains(yaml, "image: "+ImageNginx) {
		return errors.New("image di luar daftar izin")
	}
	if !strings.Contains(yaml, "name: "+NetworkName) {
		return errors.New("network di luar profil internal")
	}
	return nil
}
