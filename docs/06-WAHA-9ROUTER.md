# 06 — WAHA dan 9Router sebagai private services

## Keamanan default
File `projects/waha/compose.yaml` dan `projects/9router/compose.yaml` **opsional** dan tidak dijalankan oleh `start-infra.ps1`. Keduanya tidak membuka port host. Setelah domain dan secret sungguhan diisi, Traefik mengarahkan:

- `https://waha-admin.<domain>` ke dashboard WAHA
- `https://ai.<domain>` ke dashboard 9Router

Backend lain tetap memanggil `http://waha:3000` dan `http://ai-router:20128/v1` lewat `homelab_services`. Database dan Redis tidak mendapat label publik.

## WAHA
Dokumentasi resmi: https://waha.devlike.pro/docs/how-to/security/ dan https://waha.devlike.pro/docs/how-to/storages/

- Isi `WAHA_API_KEY`, `WAHA_DASHBOARD_USERNAME`, dan `WAHA_DASHBOARD_PASSWORD` di `.env`; gunakan random yang kuat (jangan placeholder).
- WAHA API di container: `http://waha:3000`, API memakai header `X-Api-Key`.
- `homelab_waha_sessions` di-mount ke `/app/.sessions` dan `homelab_waha_media` ke `/app/.media`.
- Perhatikan aturan WhatsApp, persetujuan pemilik akun, limit rate, privasi data, dan stabilitas sesi.
- WAHA documentation **tidak menyarankan expose API secara publik**.

Jalankan dari root folder setelah infra membentuk `homelab_services`:

```powershell
docker compose --env-file .env -f projects/waha/compose.yaml config --quiet
docker compose --env-file .env -f projects/waha/compose.yaml up -d
```

## 9Router
Proyek 9Router yang dirujuk: https://github.com/decolua/9router ; Docker docs: https://github.com/decolua/9router/blob/master/DOCKER.md

- Image starter `decolua/9router:0.5.112`; review release terbaru dan security advisory sebelum deploy.
- Isi `ROUTER_JWT_SECRET`, `ROUTER_API_KEY_SECRET`, `ROUTER_MACHINE_ID_SALT`, `ROUTER_INITIAL_PASSWORD` dengan random yang kuat.
- `REQUIRE_API_KEY=true` untuk melindungi `/v1/*`.
- Container alias: `ai-router`, URL internal: `http://ai-router:20128/v1`.
- Data state 9Router tersimpan di volume `homelab_9router_data` pada `/app/data`.
- Dashboard terbuka di `ROUTER_HOST` lewat Traefik. `REQUIRE_API_KEY=true` tetap melindungi `/v1/*`. `AUTH_COOKIE_SECURE=true` karena pengunjung datang lewat HTTPS Cloudflare.

```powershell
docker compose --env-file .env -f projects/9router/compose.yaml config --quiet
docker compose --env-file .env -f projects/9router/compose.yaml up -d
```

## Akses dashboard private saat inisialisasi
Jika dibutuhkan, untuk debugging lokal **sementara** dapat dibuat override Compose dengan binding `127.0.0.1:20128:20128` (9Router) atau `127.0.0.1:3000:3000` (WAHA), diuji hanya di host Windows dan dihapus lagi selesai setup. Hindari memasukkan `ports` tersebut ke Compose shared/permanen. Pastikan tidak konflik dengan aplikasi lain.

## Manajemen resources
Aktifkan WAHA/9Router setelah dua demo lulus, ukur `docker stats`, pastikan backup secrets/volumes, lalu uji integrasi hanya dari backend private.

## Script preflight untuk service opsional

Gunakan `./scripts/start-optional.ps1 -Service waha` atau `-Service 9router` dari PowerShell. Script memeriksa secret non-placeholder, keberadaan network, dan validitas Compose sebelum deploy.
