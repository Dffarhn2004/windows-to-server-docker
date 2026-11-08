# 12 — Test plan (acceptance tests)

## A. Sebelum deploy
- [ ] `wsl --status` dan `docker version` sukses.
- [ ] `.env` ada dan hanya lokal; `TUNNEL_TOKEN` bukan placeholder.
- [ ] `docker compose --env-file .env -f infra/compose.yaml config --quiet` sukses.
- [ ] `docker compose --env-file .env -f projects/hello/compose.yaml config --quiet` sukses.
- [ ] `docker compose --env-file .env -f projects/status/compose.yaml config --quiet` sukses.

## B. Internal Docker network
- [ ] `scripts/start-infra.ps1 -WithExamples` sukses.
- [ ] `scripts/test-routing.ps1` menghasilkan status HTTP **200** dua hostname berbeda.
- [ ] Request memakai Host random tidak sengaja di-route ke service apa pun (expected 404).
- [ ] `docker ps --format '{{.Names}} {{.Ports}}'`: tidak ada public binding WAHA/9Router/DB/Traefik.
- [ ] Container Docker socket proxy hanya di network khusus, tanpa published ports.

## C. Cloudflare
- [ ] Wildcard DNS menuju tunnel benar.
- [ ] Tunnel Healthy/Connected.
- [ ] Browser di koneksi berbeda membuka `https://hello.<domain>` dan `https://status.<domain>`.
- [ ] Terdapat HTTPS, tanpa perubahan manual per app setelah wildcard diset.

## D. Setelah WAHA/9Router opsional
- [ ] Credential nyata bukan default/placeholder.
- [ ] Masing-masing service hanya private network.
- [ ] Backend internal bisa memanggil service dengan API key.
- [ ] Backup volumes dan test restore dilakukan.

## E. Portabilitas
- [ ] Restore di host lain berhasil bila migrasi diminta, lalu old connector dimatikan.

Catat hasil nyata di `docs/00-HANDOFF.md`, bukan hanya menandai asumsi.
