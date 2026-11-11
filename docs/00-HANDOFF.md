# 00 — Handoff dan status

**Status pada 2026-11-11:** Mode uji Quick Tunnel sudah dijalankan lalu dimatikan. Container homelab berstatus Exited; volume tidak dihapus. Routing Traefik internal lolos (200/200/404). URL publik sementara hanya sempat membuka `hello`. Tunnel bernama belum dikonfigurasi. Langkah laptop baru ada di `docs/18-QUICKSTART-LAPTOP-BARU.md`.

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

## Acceptance criteria
Dua subdomain `hello.*` dan `status.*` berjalan di dua service berbeda lewat tunnel yang sama; unknown host tidak dilayani; WAHA/9Router tanpa published host ports; no secrets committed.

## Blockers yang belum diketahui
Docker di laptop ini sudah aktif. Yang belum ada: domain asli, token tunnel Cloudflare, dan DNS wildcard. Quick Tunnel tidak menggantikan published application `*.domain` → `http://traefik:80`.
