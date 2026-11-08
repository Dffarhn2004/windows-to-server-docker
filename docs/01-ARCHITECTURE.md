# 01 — Desain arsitektur

## Komponen
1. **Windows 11 Home**: mesin host fisik. Docker Desktop memakai WSL2; bukan full VM Ubuntu yang dikelola pengguna.
2. **Cloudflare DNS**: wildcard CNAME untuk subdomain, proxied melalui Cloudflare.
3. **Cloudflare Tunnel** (`cloudflared` dalam Docker): koneksi outbound dari rumah/kampus ke Cloudflare.
4. **Traefik**: reverse proxy HTTP yang membaca Docker labels dan merutekan berdasarkan `Host`.
5. **Docker-socket-proxy**: API Docker read-only yang dibatasi untuk discovery Traefik; hanya terhubung dengan Traefik pada network khusus.
6. **Edge network (`homelab_edge`)**: Traefik, cloudflared, dan web apps yang sengaja dipublish.
7. **Services network (`homelab_services`)**: WAHA, 9Router, backend, DB/Redis (tidak memiliki port publik). Ini bukan network `internal:true` karena WAHA/AI butuh egress provider.

Lihat `diagrams/architecture.mmd` dan `diagrams/request-sequence.mmd` di Mermaid viewer.

## Contoh HTTP request
`https://hello.example.com` → Cloudflare Edge → tunnel connector → `http://traefik:80` (di Docker edge network) → Docker label `Host(hello.example.com)` → Nginx container port 80.

## Kenapa Traefik?
Cloudflare mampu memetakan beberapa host langsung ke port berbeda. **Traefik bukan keharusan**, tetapi memberi routing otomatis berbasis Docker metadata, mengurangi perubahan dashboard saat nambah project.

## Konsep zone/domain
Wildcard `*.example.com` lazim dipakai untuk satu level (`hello.example.com`). Apex `example.com` perlu routing tersendiri. DNS record eksplisit bisa override wildcard, dan host tanpa label router Traefik tidak akan otomatis mendapat service.

## Deliberate non-goals
- Belum ada Kubernetes, Docker Swarm, HA multi-node, live storage replication, zero downtime.
- Belum ada SSO untuk developer/deployment API.
- Belum ada port SSH publik; akses admin lokal melalui PowerShell / Docker CLI, opsi remote access khusus direncanakan terpisah.
- Belum membuat worker/DB produk yang tidak diketahui.
