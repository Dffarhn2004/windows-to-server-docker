# 15 — Roadmap implementasi

## Phase 0 — Audit
Docker Desktop/WSL2, domain, disk/RAM, DNS Cloudflare, kebijakan keamanan.

## Phase 1 — Proof of concept
Tunnel + Traefik + Nginx hello/status. Dua public subdomain, unknown host 404, no port forwarding. Wajib lulus sebelum menambah service.

## Phase 2 — Services
Aktifkan WAHA dan 9Router secara private, backup, key validation, backend integrasi. Lanjut PostgreSQL/Redis jika memang diperlukan oleh aplikasi.

## Phase 3 — Operational maturity
Monitoring uptime, log rotation, scheduled backup, secrets manager, health checks aplikasi, staged updates, incident runbook.

## Phase 4 — Developer deployment
CI/CD image registry, project-scoped authorization, approval, audit. Hindari SSH socket Docker bebas.

## Phase 5 — Portability
Dokumen migrasi + test restore pada laptop cadangan dan/atau VPS Ubuntu.
