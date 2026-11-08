# 18 — Panduan langkah demi langkah: laptop kosong (tanpa domain dulu)

Tujuan: membuktikan Docker + Traefik + demo app berjalan di laptop baru, memakai URL acak Cloudflare Quick Tunnel. Tanpa akun, domain, atau token asli. Setelah punya domain, lanjut ke `docs/03-CLOUDFLARE.md`.

> Status: panduan ini belum diuji end-to-end. Catat hasil nyata di `docs/00-HANDOFF.md`.

## 1. Install prasyarat (sekali)
1. Install **Git for Windows**.
2. Install **Docker Desktop** (WSL2 backend, Linux containers). Jika diminta, jalankan `wsl --install` di PowerShell Admin lalu restart.
3. Docker Desktop > Settings > General: aktifkan *Start Docker Desktop when you sign in*.
4. Windows: matikan sleep saat dicolok listrik.
5. Buka PowerShell baru dan cek:
   ```powershell
   docker version
   docker compose version
   ```

## 2. Ambil project
```powershell
cd C:\
git clone https://github.com/Dffarhn2004/windows-to-server-docker.git homelab
cd homelab
```

## 3. Buat `.env` untuk mode uji
```powershell
Copy-Item .env.example .env
notepad .env
```
Ubah hanya baris berikut (domain contoh boleh dibiarkan):
```
TUNNEL_TOKEN=dummy-token-untuk-uji-lokal
```
Token palsu ini membuat cloudflared utama gagal terhubung. Itu normal dan tidak berbahaya selama mode uji. Jangan isi secret WAHA/9Router/DB sekarang.

## 4. Nyalakan infra dan demo
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start-infra.ps1 -WithExamples
```
Cek dengan `docker ps`: kolom PORTS tidak boleh berisi `0.0.0.0:`.

## 5. Uji routing internal
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\test-routing.ps1
```
Harus tampil PASS 200 untuk `hello` dan `status`, dan PASS 404 untuk unknown host.

## 6. Buka lewat internet dengan Quick Tunnel
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\quick-tunnel.ps1 -Target hello
```
Buka URL `https://xxxx.trycloudflare.com` yang tercetak. Untuk app lain: `-Target status`. Satu tunnel hanya melayani satu app, dan URL berubah setiap dijalankan.

Matikan setelah selesai:
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\quick-tunnel.ps1 -Stop
```

## 7. Berhenti / bersih-bersih
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\stop-infra.ps1
docker compose --env-file .env -f projects/hello/compose.yaml stop
docker compose --env-file .env -f projects/status/compose.yaml stop
```
Jangan memakai `down -v` jika sudah ada data penting.

## 8. Lanjut ke produksi
Beli domain murah, tambahkan ke Cloudflare, buat tunnel (token asli), published app `*` ke `http://traefik:80`, dan CNAME wildcard. Ganti `TUNNEL_TOKEN`, `ROOT_DOMAIN`, `HELLO_HOST`, `STATUS_HOST` di `.env`, lalu jalankan ulang `start-infra.ps1`. Detail: `docs/03-CLOUDFLARE.md`.
