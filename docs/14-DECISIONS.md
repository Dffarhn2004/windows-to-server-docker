# 14 — Architecture Decision Records (ADR)

## ADR-001: Docker Desktop WSL2, bukan VirtualBox
Status: Accepted. User memilih Windows 11 Home dan tidak ingin VM VirtualBox. Konsekuensi: startup dipengaruhi Windows/login, bukan setara Ubuntu VPS production.

## ADR-002: Cloudflare Tunnel + Traefik
Status: Accepted. Wildcard routing Cloudflare hanya perlu dikonfigurasi sekali; Traefik discovery berbasis Docker labels. Cloudflare bisa routing multi-host tanpa Traefik, namun maintainability project banyak lebih baik dengan Traefik.

## ADR-003: Private services tidak diberi public ports
Status: Accepted. WAHA/9Router/data private by default; hanya backend dapat mengakses lewat shared network. Bukan pengganti authorization setiap service.

## ADR-004: Docker socket proxy
Status: Accepted untuk starter. Read-only limited proxy mengurangi attack surface Traefik dibanding direct Docker socket, meskipun masih ada metadata exposure.

## ADR-005: Satu active host tunnel untuk migrasi
Status: Accepted untuk cold migration. Dua laptop boleh aktif untuk skenario HA di masa depan, tetapi butuh shared state, traffic management, dan security khusus.

## ADR-006: Compose per project
Status: Accepted. Infastruktur + dua demo + tiap optional service file terpisah. Menambah project tidak perlu mengedit Compose utama; cukup join shared network dan buat Traefik labels.

## ADR-007: Homelab Control Center P0-A/B (control plane baca-saja)
Status: Accepted untuk Gate P0-A dan P0-B. P0-C/P0-D belum dibuka.

Audit baca-saja 2026-11-11 (docker ps, tanpa start/stop): Docker Desktop 4.95.0, engine 29.8.2, Compose v5.6.0. Container homelab yang Up: `homelab-infra-docker-socket-proxy-1`, `homelab-infra-traefik-1`, `homelab-infra-cloudflared-1`, `homelab-hello-hello-1`, `homelab-status-status-1`, `homelab-waha-waha-1`. Network repo tetap `homelab_edge` dan `homelab_services`. Network `homelab_proxy` ada di host tetapi tidak dipakai. CVAT Exited. `9router` Up dan sudah mem-publish `127.0.0.1` sebelum HCC; tidak diubah.

- **Auth UI:** sesi admin loopback, cookie HttpOnly `SameSite=Strict`, password dari `HCC_ADMIN_PASSWORD`, pembanding constant-time. Verifier JWT Cloudflare Access hanya interface. Tidak ada hostname publik dashboard.
- **Agent:** proses Go di host Windows, poll keluar ke Control API dengan HMAC-SHA256, timestamp, dan nonce. Next.js dan Go API tidak memegang `docker.sock`. Socket proxy Traefik bukan jalur deploy.
- **Secret:** rahasia agent di env atau file di luar Git, bukan plaintext SQLite. Token manajemen Cloudflare (`HCC_CF_API_TOKEN`) terpisah dari `TUNNEL_TOKEN` dan tidak disimpan di database. Tanpa token itu, integrasi tetap read-only dan berstatus tidak dikonfigurasi.
- **Kepemilikan:** container tanpa label `hcc.managed=true` adalah Unmanaged. HCC tidak mengadopsi `hello.imuii.id`, `waha-v2.imuii.id`, `api.imuii.id`, atau record `A *`. Fase ini tidak menulis DNS, Tunnel, atau Access.
- **Status:** label Running hanya dari state Docker `running`. Label Healthy hanya dari healthcheck Docker. Tunnel tidak ditandai Healthy hanya karena container cloudflared Up.
