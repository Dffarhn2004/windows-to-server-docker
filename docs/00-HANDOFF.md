# 00 — Handoff dan status

**Status pada 2026-11-11:** Mode uji Quick Tunnel sudah dijalankan lalu dimatikan. Container homelab berstatus Exited; volume tidak dihapus. Routing Traefik internal lolos (200/200/404). URL publik sementara hanya sempat membuka `hello`. Tunnel bernama belum dikonfigurasi. Langkah laptop baru ada di `docs/18-QUICKSTART-LAPTOP-BARU.md`.

## Status terbaru — pembaruan 11 November 2026 (berdasarkan terminal & screenshot pengguna)

**Catatan:** Bagian "Status pada 2026-11-11" di bawah adalah snapshot PoC **sebelumnya**, bukan status runtime terbaru. Agent harus memeriksa lagi di mesin.

- Nameserver domain `imuii.id` pindah ke Cloudflare; hasil lookup NS dan `api.imuii.id` menuju VPS dilaporkan benar.
- Tunnel **remotely managed** bernama `homelab-docker` digunakan oleh `infra/compose.yaml` di Docker Desktop.
- `scripts/start-infra.ps1 -WithExamples` dilaporkan berhasil menjalankan socket proxy, Traefik, Cloudflared, Hello, dan Status.
- `scripts/test-routing.ps1`: Hello HTTP 200, Status HTTP 200, unknown host HTTP 404.
- `https://hello.imuii.id` menampilkan "Hello dari Docker".
- WAHA existing `projects/waha/compose.yaml` dilaporkan berjalan; `waha-v2.imuii.id/dashboard` dan worker API menampilkan **1 connected**, belum ada bukti sesi WhatsApp aktif.
- Cloudflare Access app `waha-v2` dengan policy Allow telah dibuat; jangan menghapus/mengubah tanpa approval.
- Wildcard DNS `A *` tetap dipakai VPS. **Jangan dialihkan ke Tunnel.** Subdomain HCC harus record spesifik.
- PRD baru: [docs/19-HOMELAB-CONTROL-CENTER-PRD.md](19-HOMELAB-CONTROL-CENTER-PRD.md); Cursor execution: [docs/20-CURSOR-CONTROL-CENTER-IMPLEMENTATION.md](20-CURSOR-CONTROL-CENTER-IMPLEMENTATION.md). **Dashboard belum dibuat.**

## Objective
Laptop Windows 11 Home berperan sebagai host multi-project; satu wildcard Cloudflare Tunnel → satu Traefik → container berdasarkan host; WAHA dan 9Router hanya private network; migratable ke laptop lain / Ubuntu VPS.

## Asumsi yang HARUS diperiksa di laptop
- Docker Desktop sudah terpasang atau bisa dipasang dan memakai Linux container + WSL2.
- Domain sudah aktif pada Cloudflare DNS, memiliki hak membuat tunnel, wildcard CNAME.
- Windows tidak dalam mode sleep selama server dipakai.
- Resource disk/RAM memadai untuk jumlah sesi WAHA dan apps.
- Tidak ada Traefik/tunnel/ports lain yang konflik.

## Langkah berikutnya (prioritas)
- [x] Audit Docker Desktop dan WSL2. Docker Desktop 4.95.0, engine 29.8.2, Compose v5.6.0, WSL2 `docker-desktop` Running.
- [x] `.env` lokal dibuat dari contoh, `TUNNEL_TOKEN` diganti token palsu mode uji. Domain masih `example.com`.
- [ ] Konfigurasi tunnel remotely managed, published app wildcard → `http://traefik:80`.
- [x] `scripts/start-infra.ps1 -WithExamples` berhasil. `cloudflared` utama restart karena token palsu; itu diharapkan di mode uji.
- [x] `scripts/test-routing.ps1`: HTTP 200 untuk `hello.example.com` dan `status.example.com`.
- [ ] Browser internet mengakses dua subdomain bernama. Yang teruji publik hanya Quick Tunnel `hello`.
- [x] Test unknown host → HTTP 404 di Traefik internal.
- [ ] Atur sleep Windows, auto-start Docker Desktop setelah login.
- [ ] Review kebutuhan WAHA/9Router, backup, dan security sebelum mengaktifkan.

## Perubahan agent berikutnya
Isi log dengan format:
`YYYY-MM-DD | Agent | Perubahan | Tes yang benar-benar dijalankan | Hasil | Risiko/rollback`.

2026-11-11 | Agent | Mode uji Quick Tunnel. `.env` lokal dengan token palsu (tidak di-commit). Skrip PowerShell di `scripts/` mengizinkan stderr Docker pada Windows PowerShell 5.1 agar `network not found` tidak menggagalkan start. | `start-infra.ps1 -WithExamples`; `test-routing.ps1`; `quick-tunnel.ps1 -Target hello`; curl HTTPS dan browser ke URL trycloudflare. | Infra, hello, status Up. Traefik 200/200/404. Browser menampilkan "Hello dari Docker!". Tidak ada port `0.0.0.0` pada container homelab. | Rollback: `scripts/quick-tunnel.ps1 -Stop`, lalu `docker compose --env-file .env -f projects/hello/compose.yaml stop`, sama untuk status, lalu `scripts/stop-infra.ps1`. Jangan `down -v`. `cloudflared` utama tetap restart sampai token asli diisi.

2026-11-11 | Agent | Mematikan stack uji dan menulis ulang `docs/18-QUICKSTART-LAPTOP-BARU.md` dari perintah yang benar-benar lolos. | `quick-tunnel.ps1 -Stop`; `docker compose stop` untuk hello dan status; `stop-infra.ps1`; `docker ps -a --filter name=homelab-`. | Quick tunnel hilang. Lima container homelab Exited. Volume tidak dihapus. | Untuk menyalakan lagi, ikuti bagian C–E di `docs/18`. Jangan `down -v`.

2026-11-11 | Agent | Menyiapkan hostname dashboard: `WAHA_HOST` ke WAHA port 3000 dan `ROUTER_HOST` ke 9Router port 20128. Tanpa `ports:` host. Secret placeholder tidak dijalankan. | `docker compose config --quiet` untuk `projects/waha/compose.yaml` dan `projects/9router/compose.yaml` dengan `.env.example` dan `.env` lokal. | Config valid. Router `Host(waha-admin.example.com)` dan `Host(ai.example.com)`. Container tidak di-start. | Rollback: kembalikan dua compose itu ke network `homelab_services` saja dan hapus label Traefik. Jangan `down -v` jika volume sesi sudah terisi.

2026-11-11 | Agent | Gate P0-A/P0-B Homelab Control Center: ADR-007, API Go, agent Windows baca-saja, UI Next.js. Tidak mengubah `infra/compose.yaml`, DNS, atau volume. | `docker version`; `docker ps`; `docker network ls`; `go test` pada `control-center/pkg`, `services/api`, dan `agent`; `npm run lint`; `npm run build`; `docker compose -f control-center/infra/compose.yaml config --quiet`; agent `-once`; UI loopback login, ringkasan, tab Dikelola kosong. | API lulus. Compose valid. Observasi: 33 container, 0 managed. Hello, WAHA, CVAT, 9Router Unmanaged. Docker 29.8.2 Running, Traefik Running, container cloudflared Running, control plane Cloudflare `not_configured` karena token manajemen tidak diisi. Deploy anonim 401. | Rollback: hentikan proses `hcc-api`, `hcc-agent`, dan `next`. Jangan `down -v`. Stack homelab tidak di-restart.

2026-11-11 | Agent | Gate P0-C: deploy Nginx internal-only milik HCC. Tidak menulis Cloudflare, tidak mengubah `infra/compose.yaml`, tidak menghapus volume. | `go test` manifest, API, dan agent; `POST /api/v1/apps/validate` lalu deploy slug `smoke-internal`; `docker inspect` proyek `hcc-smoke-internal`; stop, start, delete; `docker compose -f control-center/infra/compose.yaml config --quiet`; wizard `/deploy`. | Validate 200, proyek `hcc-smoke-internal`, network `homelab_services`, `publicUrlReady` false, YAML tanpa Traefik dan tanpa `ports`. Job deploy, stop, start, dan delete `succeeded`. Container hanya `80/tcp`, `PortBindings` kosong, bukan `homelab_edge`, label `hcc.managed`/`hcc.project-id`/`hcc.spec-hash`. Folder spec terhapus. Hello dan WAHA tetap Up. Compose HCC valid. Wizard menampilkan pratinjau tanpa pilihan hostname. Hello tetap tanpa aksi mutasi. | Rollback: `docker compose -p hcc-smoke-internal -f managed-projects/smoke-internal/compose.yaml down` tanpa `-v`. Jangan sentuh proyek `homelab-*`. Stack uji sudah di-down tanpa volume.

## Acceptance criteria
Dua subdomain `hello.*` dan `status.*` berjalan di dua service berbeda lewat tunnel yang sama; unknown host tidak dilayani; WAHA/9Router tanpa published host ports; no secrets committed.

## Blockers yang belum diketahui
Docker di laptop ini sudah aktif. Yang belum ada: domain asli, token tunnel Cloudflare, dan DNS wildcard. Quick Tunnel tidak menggantikan published application `*.domain` → `http://traefik:80`.
