# 17 — PostgreSQL dan Redis (opsional, internal)

Kedua service **tidak** dijalankan saat bootstrap. File `projects/data/compose.yaml` menerapkan named volumes dan tidak mengaktifkan port host atau Traefik labels.

## Aktifkan
1. Isi `POSTGRES_PASSWORD`, `REDIS_PASSWORD` di root `.env` dengan secret random kuat, jangan placeholder.
2. Jalankan setelah `scripts/start-infra.ps1`:

```powershell
.\scripts\start-optional.ps1 -Service data
```

3. Backend harus bergabung ke Docker network `homelab_services` dan dapat mengakses:
- PostgreSQL: `postgres:5432`, username `POSTGRES_USER`, db `POSTGRES_DB`, password dari secret app.
- Redis: `redis:6379`, auth menggunakan `REDIS_PASSWORD`.

## Security dan backup
- Tidak ada `ports:` untuk DB/Redis, sehingga tidak tersedia secara otomatis dari Windows host/internet.
- Semua container di shared network tetap bisa mencoba berkomunikasi. Segmentasikan network jika banyak tenant/developer.
- `homelab_postgres_data` dan `homelab_redis_data` adalah Docker named volumes.
- PostgreSQL butuh backup konsisten `pg_dump`/`pg_restore`; jangan menyalin live volume sebagai pengganti logical backup.
- Redis persistence (AOF) perlu diuji dan backup mengikuti kebutuhan data dan restart.
- Password environment masih bisa terlihat oleh administrator Docker; gunakan secret manager untuk production.
- Hindari menjalankan database di laptop yang sering sleep sebagai datastore production 24/7.
