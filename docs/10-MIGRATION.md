# 10 — Memindahkan server laptop A → B atau Ubuntu VPS

## Mode disarankan: cold migration
1. Tentukan window downtime dan backup point; catat image versions dan config.
2. Siapkan Docker Desktop di laptop B / Docker Engine di VPS, clone repo yang sama, `.env` baru yang aman.
3. Matikan penulis data di laptop A (WAHA, 9Router, DB) dan buat backup **konsisten**.
4. Copy arsip encrypted, restore di laptop B; validasi container tanpa tunnel publik.
5. Jalankan Traefik + app service di laptop B, uji Host-header request di Docker network.
6. **Matikan cloudflared lama** pada laptop A sebelum mengaktifkan connector baru menggunakan tunnel yang sama. Hindari dua connector menerima traffic berbeda dengan data tidak sinkron.
7. Aktifkan cloudflared laptop B dan uji subdomain publik.
8. Monitor log, kesehatan WAHA sessions dan auth 9Router; simpan rollback A hingga yakin.

## Kenapa DNS tidak perlu diubah?
Kalau memakai tunnel yang sama, wildcard CNAME masih menuju `<UUID>.cfargotunnel.com`. Yang berganti adalah konektor/host yang menjalankan tunnel dan app data.

## Caveat
- Tidak otomatis zero downtime, terutama WAHA dan aplikasi stateful.
- Image architecture perlu cocok, misalnya amd64 vs arm64.
- Secrets tidak boleh ikut Git; gunakan secret store/transfer encrypted.
- Jika komputer A/B berjalan bersamaan dalam tunnel yang sama, Cloudflare dapat meneruskan ke connector lain; tidak ada jaminan per-application sticky-to-machine tanpa desain HA.
