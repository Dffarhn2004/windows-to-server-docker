# 00 — Handoff dan status

**Status pada 2026-11-08:** Starter artifact dibuat dan file Compose lolos pemeriksaan parser YAML offline (termasuk database/Redis opsional). **Belum dilakukan docker image pull, `docker compose config`, deploy Cloudflare, atau browser E2E pada laptop pengguna.**

## Objective
Laptop Windows 11 Home berperan sebagai host multi-project; satu wildcard Cloudflare Tunnel → satu Traefik → container berdasarkan host; WAHA dan 9Router hanya private network; migratable ke laptop lain / Ubuntu VPS.

## Asumsi yang HARUS diperiksa di laptop
- Docker Desktop sudah terpasang atau bisa dipasang dan memakai Linux container + WSL2.
- Domain sudah aktif pada Cloudflare DNS, memiliki hak membuat tunnel, wildcard CNAME.
- Windows tidak dalam mode sleep selama server dipakai.
- Resource disk/RAM memadai untuk jumlah sesi WAHA dan apps.
- Tidak ada Traefik/tunnel/ports lain yang konflik.

## Langkah berikutnya (prioritas)
- [ ] Audit Docker Desktop dan WSL2.
- [ ] Buat `.env` lokal dan isi domain, host uji, dan TUNNEL_TOKEN tanpa membocorkannya.
- [ ] Konfigurasi tunnel remotely managed, published app wildcard → `http://traefik:80`.
- [ ] `scripts/start-infra.ps1 -WithExamples` berhasil.
- [ ] `scripts/test-routing.ps1` menghasilkan 200 untuk dua hostname.
- [ ] Browser internet mengakses dua subdomain dengan konten tepat.
- [ ] Test unknown host → 404/no matched router.
- [ ] Atur sleep Windows, auto-start Docker Desktop setelah login.
- [ ] Review kebutuhan WAHA/9Router, backup, dan security sebelum mengaktifkan.

## Perubahan agent berikutnya
Isi log dengan format:
`YYYY-MM-DD | Agent | Perubahan | Tes yang benar-benar dijalankan | Hasil | Risiko/rollback`.

## Acceptance criteria
Dua subdomain `hello.*` dan `status.*` berjalan di dua service berbeda lewat tunnel yang sama; unknown host tidak dilayani; WAHA/9Router tanpa published host ports; no secrets committed.

## Blockers yang belum diketahui
Domain asli, token Cloudflare, status DNS, status Docker di laptop; semuanya harus diperiksa secara lokal.
