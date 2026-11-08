# 05 — Menambahkan project/subdomain baru

## Pola
1. Buat folder `projects/<project>/` berisi `compose.yaml` dan file aplikasi.
2. Tentukan container app internal port (mis. Next.js 3000, Go 8080).
3. Hubungkan ke `homelab_edge`; tambahkan label Traefik `Host()` dan port internal.
4. Jika backend butuh WAHA/9Router, hubungkan **tambahan** ke `homelab_services` (tanpa mengekspos service private).
5. Deploy dengan `docker compose --env-file .env -f projects/<project>/compose.yaml up -d`.
6. Periksa logs, HTTP 200 internal, dan akses domain.

Gunakan `templates/backend.compose.example.yaml` sebagai basis, tapi **wajib** mengganti image, nama project, router, port, dan domain placeholder.

## Contoh Next.js
App di container harus listen di `0.0.0.0:3000` (bukan localhost-only container). Label service port = 3000, tidak perlu `ports: '3000:3000'`.

## Contoh backend Go
Backend listen `0.0.0.0:8080`. Dari Go container pada `homelab_services`, gunakan URL `http://waha:3000` atau `http://ai-router:20128/v1` dengan autentikasi.

## Penamaan dan pemeliharaan
- Satu project Compose per project agar restart/update independen.
- Router names unik, hostname unik, network name konsisten.
- Pin image version/tag untuk stabilitas; update dan rollback terjadwal.
- DNS wildcard cukup sekali, asalkan hostname tidak diblok oleh DNS record yang lebih spesifik.
- Unknown subdomain tanpa label akan menghasilkan no matching router (bukan membuka semua container).
