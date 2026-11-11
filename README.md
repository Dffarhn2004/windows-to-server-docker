# Windows 11 Home Homelab — Docker + Cloudflare + Traefik

> **Update 11 Nov 2026 (berdasarkan tes yang dilaporkan pengguna):** Docker + Traefik + remote Tunnel `homelab-docker` aktif; `hello.imuii.id` dapat dibuka; WAHA di `waha-v2.imuii.id` menunjukkan worker connected. Skrip routing internal lulus 200/200/404. Status live harus selalu diperiksa ulang dari host; bukti awal PoC lama ada di dokumen handoff.

## Next project — Homelab Control Center (PRD + Cursor)

**Baca lebih dulu:** [Full PRD](docs/19-HOMELAB-CONTROL-CENTER-PRD.md) → [Cursor Implementation Handoff](docs/20-CURSOR-CONTROL-CENTER-IMPLEMENTATION.md) → [Prompt Agent](NEXT_AGENT_PROMPT.md).

Target: dashboard Next.js + Go untuk melihat host/container, deploy aplikasi dari template aman, manajemen log/operasi, dan provisioning **exact hostname** Cloudflare DNS/Tunnel/Access via agent terkontrol. **Belum diimplementasikan**; dokumen ini adalah rencana. Jangan membuat ulang Traefik/Tunnel, jangan mengubah wildcard DNS karena `A *` masih digunakan VPS.

## Publikasi ke GitHub (Windows)

GitHub connector yang dipakai saat paket dibuat menolak penulisan (HTTP 403). Untuk mempublikasikannya dari laptop kamu:

1. Download dan ekstrak `windows-to-server-docker-github-ready.zip`.
2. Install Git for Windows dan login ke GitHub (Git Credential Manager, jika diminta).
3. Buka PowerShell di folder hasil ekstrak.
4. Jalankan `powershell -ExecutionPolicy Bypass -File .\publish-to-github.ps1`.

Script menolak overwrite jika repository sudah memiliki tracked files, dan tidak melakukan `force push`. **Repository `Dffarhn2004/windows-to-server-docker` saat ini public.** Simpan `.env`, tunnel tokens, WhatsApp sessions dan password hanya di laptop, bukan Git.

## Apa yang tersedia?

- **Satu Cloudflare Tunnel** + domain wildcard yang dikonfigurasi sekali.
- **Traefik** untuk memilih container berdasarkan hostname dari Docker labels.
- **Docker socket proxy** read-only API sebagai pengurang risiko dibanding langsung memberikan socket penuh ke Traefik.
- **Dua service demo (hello dan status)** untuk membuktikan dua subdomain berbeda lewat satu tunnel.
- **WAHA, 9Router, serta PostgreSQL/Redis** tersedia sebagai Compose opsional tanpa membuka port host.
- Dokumentasi Markdown untuk instalasi, keamanan, testing, backup, migrasi laptop, dan handoff agent.

> **Belum punya domain / laptop baru?** Ikuti `docs/18-QUICKSTART-LAPTOP-BARU.md` (mode uji dengan URL acak Quick Tunnel, tanpa token asli).

## Quick start — PowerShell Windows

1. Install WSL2 + Docker Desktop, aktifkan Linux containers. Baca `docs/02-WINDOWS-SETUP.md`.
2. Kelola DNS domain di Cloudflare dan buat tunnel bertipe **remotely managed** (token). Baca `docs/03-CLOUDFLARE.md`.
3. Ekstrak paket misalnya `C:\homelab`, lalu buka PowerShell di direktori proyek.
4. Jalankan:

```powershell
Copy-Item .env.example .env
notepad .env
```

Ganti `ROOT_DOMAIN`, `HELLO_HOST`, `STATUS_HOST`, dan `TUNNEL_TOKEN`. Jangan pernah commit `.env`.

5. Jalankan:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start-infra.ps1 -WithExamples
powershell -ExecutionPolicy Bypass -File .\scripts\test-routing.ps1
```

6. Di Cloudflare Tunnel, buat published application `*.domainmu.com` menuju **`http://traefik:80`**, serta pastikan DNS wildcard menuju tunnel. Setelah itu akses `https://hello.domainmu.com` dan `https://status.domainmu.com`.

## Command manual tanpa script

```powershell
docker network create homelab_services
docker compose --env-file .env -f infra/compose.yaml up -d
docker compose --env-file .env -f projects/hello/compose.yaml up -d
docker compose --env-file .env -f projects/status/compose.yaml up -d
docker compose --env-file .env -f infra/compose.yaml ps
```

`docker network create` hanya sekali. Jika network sudah ada, gunakan network tersebut. **Jangan pakai `docker compose down -v` untuk data penting.**

## Jika memerlukan WAHA / 9Router

Pelajari `docs/06-WAHA-9ROUTER.md` dan `docs/17-DATABASE-REDIS.md`, isi semua env secret, lalu jalankan `scripts/start-optional.ps1 -Service waha`, `-Service 9router`, atau `-Service data`. Dashboard WAHA ada di `WAHA_HOST` dan 9Router di `ROUTER_HOST`. Tidak ada port host yang dibuka. Database tetap tanpa alamat publik.

## Dokumentasi untuk agent berikutnya

1. **`AGENTS.md`** — aturan yang wajib ditaati agent.
2. **`NEXT_AGENT_PROMPT.md`** — instruksi siap salin untuk melanjutkan setup.
3. **`docs/00-HANDOFF.md`** — status, asumsi, langkah berikutnya, dan acceptance criteria.
4. **`docs/01-ARCHITECTURE.md`** sampai **`docs/16-SOURCES.md`** — rincian desain.

## Keterbatasan

- Windows + Docker Desktop **bukan VPS Ubuntu penuh**, dan startup Docker Desktop pada Windows Home bergantung pada startup/login pengguna.
- Templat ini tidak menyediakan remote SSH multi-user dengan privilege separation; desainnya dijelaskan di dokumen keamanan.
- Wildcard untuk `*.example.com` tidak otomatis menangani apex `example.com`.
- Dashboard WAHA dan 9Router memakai login aplikasinya sendiri. Tambahkan Cloudflare Access pada dua hostname itu sebelum menganggapnya aman untuk internet.
- Tidak ada jaminan zero-downtime jika berpindah laptop; semua data persisten harus dimigrasikan dan diuji.

Gunakan Markdown di dalam ZIP sebagai sumber kebenaran teknis. Jangan menganggap token atau domain contoh sebagai nilai riil.

## Referensi visual yang disertakan

`references/Arsitektur_Homelab_Windows_Docker_Cloudflare.pdf` (PDF diskusi awal) dan `references/windows_docker_cloudflare_homelab_routing_flow.png` (infografik flow) adalah visual pendamping. **Konfigurasi YAML dan Markdown terbaru adalah sumber kebenaran**, bukan gambar lama.
