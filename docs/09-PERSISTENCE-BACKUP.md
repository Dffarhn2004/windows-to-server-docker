# 09 — Storage, backup, dan restore

## Apa yang harus dibackup?
1. Repository config + versi Compose (Git; **tanpa secrets**).
2. `.env` dan secret nyata (secret manager/offline encrypted backup, **tidak ke Git**).
3. Named volumes: `homelab_waha_sessions`, `homelab_waha_media`, `homelab_9router_data`.
4. Database menggunakan **logical backup** (`pg_dump`, redis persistence sesuai mode), bukan hanya tar volume aktif.
5. Inventaris DNS, Cloudflare tunnel ID dan kebijakan Access.

## Export volume (stop writers dulu)
```powershell
# Contoh: hentikan WAHA sebelum melakukan export agar snapshot tidak berubah.
docker compose --env-file .env -f projects/waha/compose.yaml stop
.\scripts\export-volume.ps1 -Volume homelab_waha_sessions
.\scripts\export-volume.ps1 -Volume homelab_waha_media
docker compose --env-file .env -f projects/waha/compose.yaml start
```

Arsip ditulis ke folder `backups/` dan **tidak** dimasukkan ke Git/ZIP ini. Arsip dapat berisi auth/session sensitif dan harus dienkripsi ketika dipindahkan. Cek file hasil dan **uji restore** secara terpisah.

## Import volume (contoh prinsip)
1. Pastikan container yang memakai volume dimatikan.
2. Buat volume tujuan kosong: `docker volume create homelab_waha_sessions`.
3. Jalankan `alpine` dengan mount volume target dan mount direktori arsip, extract tar (jangan menimpa data aktif sembarangan).
4. Periksa owner/permission dan uji service; WAHA bisa meminta re-link perangkat.

## Catatan DB
Untuk PostgreSQL: gunakan `pg_dump`/`pg_restore` dengan versi yang kompatibel, backup dengan jadwal, retensi, dan test restore. Archive file database SQLite saat live dapat tidak konsisten — hentikan writer / gunakan metode backup konsisten menurut aplikasi.

## Jangan lakukan
`docker compose down -v`, `docker volume prune`, menghapus Docker Desktop data disk tanpa backup, atau sinkronisasi WAHA sessions dua laptop secara bersamaan tanpa manajemen concurrency.
