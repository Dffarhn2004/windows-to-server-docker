# 18 — Laptop baru, langkah demi langkah

Gunakan panduan ini saat memindahkan homelab ke laptop Windows lain.

Diuji pada 2026-11-11 di Windows 11 dengan Docker Desktop 4.95.0, Compose v5.6.0, dan WSL2. Yang lolos: infra + `hello` + `status` naik, Traefik internal mengembalikan 200/200/404, dan Quick Tunnel `hello` tampil di browser. Tunnel bernama dengan domain asli belum diuji.

## Yang dibawa ke laptop baru

Salin **folder proyek ini**, termasuk `scripts/`. Perbaikan di skrip itu perlu: Windows PowerShell 5.1 menganggap pesan stderr Docker sebagai error fatal, sehingga skrip lama berhenti pada `network homelab_services not found`.

Jangan mengandalkan `git clone` dari GitHub sebelum perbaikan skrip itu di-push. Jangan ikut menyalin `.env` lewat Git, dan jangan commit `.env`.

Di laptop baru, mulai dari dua demo saja. WAHA, 9Router, dan database menunggu secret sungguhan. Lihat `docs/06-WAHA-9ROUTER.md` dan `docs/17-DATABASE-REDIS.md` nanti.

## A. Pasang Docker (sekali)

1. Install Docker Desktop untuk Windows. Pilih backend **WSL2** dan container **Linux**.
2. Jika installer meminta WSL, jalankan PowerShell sebagai Administrator:

```powershell
wsl --install --no-distribution
wsl --update
```

Restart jika diminta. Tidak perlu memasang distro Ubuntu terpisah. Docker Desktop memakai WSL-nya sendiri.

3. Buka Docker Desktop sampai engine-nya siap. Di Settings > General, aktifkan *Start Docker Desktop when you sign in*.
4. Saat laptop dipakai sebagai server, matikan sleep Windows, termasuk saat tutup layar.
5. Buka PowerShell baru dan pastikan ketiga perintah ini berhasil:

```powershell
docker version
docker compose version
wsl --status
```

`docker version` harus menampilkan `OS/Arch: linux/amd64` pada bagian Server. Context-nya `desktop-linux`.

## B. Siapkan proyek dan `.env` uji

1. Letakkan folder proyek di path pendek, misalnya `C:\homelab`.
2. Di PowerShell, masuk ke folder itu lalu buat env lokal:

```powershell
cd C:\homelab
Copy-Item .env.example .env
notepad .env
```

3. Ubah hanya baris token menjadi:

```
TUNNEL_TOKEN=dummy-token-untuk-uji-lokal
```

`ROOT_DOMAIN`, `HELLO_HOST`, dan `STATUS_HOST` boleh tetap `example.com` untuk mode uji. Jangan mengganti secret WAHA, 9Router, atau database.

Token palsu membuat container `cloudflared` utama terus restart. Itu diharapkan, dan tidak menghalangi Quick Tunnel.

## C. Nyalakan infra dan dua demo

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start-infra.ps1 -WithExamples
```

Tunggu sampai image selesai diunduh. Lalu cek:

```powershell
docker ps --filter name=homelab-
```

Yang harus terlihat Up: `docker-socket-proxy`, `traefik`, `hello`, dan `status`. `cloudflared` utama boleh berstatus Restarting. Kolom PORTS tidak boleh berisi `0.0.0.0`. Tulisan `80/tcp` atau `2375/tcp` tanpa alamat host artinya port itu hanya ada di dalam Docker.

## D. Uji routing Traefik di dalam Docker

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\test-routing.ps1
```

Hasil yang benar:

- `hello.example.com` → `PASS - HTTP 200`
- `status.example.com` → `PASS - HTTP 200`
- `unknown.invalid` → `PASS - HTTP 404`

Tes ini yang membuktikan dua aplikasi berbeda dipilih lewat header Host. Quick Tunnel di langkah berikutnya tidak melewati pemilihan host Traefik.

## E. Buka satu aplikasi ke internet, tanpa domain

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\quick-tunnel.ps1 -Target hello
```

Buka URL `https://....trycloudflare.com` yang tercetak. Halaman harus menampilkan **Hello dari Docker!**

Satu Quick Tunnel hanya melayani satu aplikasi, dan URL berubah setiap tunnel dinyalakan ulang. Untuk `status`, hentikan tunnel yang ada lalu jalankan `-Target status`. Jangan mengarahkan Quick Tunnel ke WAHA, 9Router, atau database.

## F. Matikan setelah uji

Urutan ini sudah dipakai pada 2026-11-11. Container berhenti, volume tidak dihapus:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\quick-tunnel.ps1 -Stop
docker compose --env-file .env -f projects/hello/compose.yaml stop
docker compose --env-file .env -f projects/status/compose.yaml stop
powershell -ExecutionPolicy Bypass -File .\scripts\stop-infra.ps1
```

`docker ps -a --filter name=homelab-` boleh masih menampilkan container berstatus `Exited`. Itu sisa container yang di-stop, bukan layanan yang masih berjalan. Jangan memakai `docker compose down -v`.

## G. Nanti, setelah punya domain

Bagian ini belum diuji di laptop sumber. Tunnel Cloudflare-nya gratis, tetapi domain harus kamu kelola di Cloudflare.

1. Tambahkan domain ke Cloudflare dan tunggu sampai statusnya aktif.
2. Buat tunnel **remotely managed**. Salin token ke `.env` sebagai `TUNNEL_TOKEN`, menggantikan token palsu. Jangan mengirim token itu lewat chat atau Git.
3. Ubah juga:

```
ROOT_DOMAIN=domainmu.com
HELLO_HOST=hello.domainmu.com
STATUS_HOST=status.domainmu.com
WAHA_HOST=waha-admin.domainmu.com
ROUTER_HOST=ai.domainmu.com
```

Ganti juga setiap secret `REPLACE_WITH_...` untuk WAHA dan 9Router dengan nilai acak yang panjang. Jangan menempelkan nilai itu ke chat atau Git.

4. Di tunnel, buat published application hostname `*.domainmu.com`, service HTTP, URL `http://traefik:80`. Pastikan DNS memiliki CNAME `*` ke tunnel dan statusnya Proxied. Jangan membuat CNAME ganda. Apex `domainmu.com` tidak ikut wildcard.
5. Nyalakan ulang:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start-infra.ps1 -WithExamples
```

`cloudflared` utama sekarang harus Up, bukan Restarting. Dari internet, `https://hello.domainmu.com` dan `https://status.domainmu.com` harus menampilkan halaman dari container yang berbeda. Detail ada di `docs/03-CLOUDFLARE.md`.

Quick Tunnel tidak diperlukan lagi setelah tunnel bernama sehat.

6. Setelah `hello` dan `status` benar dari internet, nyalakan dua dashboard:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start-optional.ps1 -Service waha
powershell -ExecutionPolicy Bypass -File .\scripts\start-optional.ps1 -Service 9router
```

Skrip menolak jalan jika secret masih placeholder. Wildcard `*.domainmu.com` sudah mencakup `waha-admin` dan `ai`, jadi tidak perlu tunnel baru.

- `https://waha-admin.domainmu.com` membuka login WAHA. Username dan password ada di `.env`.
- `https://ai.domainmu.com` membuka login 9Router. Password pertama adalah `ROUTER_INITIAL_PASSWORD`.
- API untuk aplikasi lain tetap `http://waha:3000` dan `http://ai-router:20128/v1` dari dalam Docker, bukan lewat alamat publik.

Di Cloudflare Zero Trust, buat Access policy untuk dua hostname itu supaya hanya akun kamu yang mencapai halaman login. Bagian dashboard ini belum diuji di laptop sumber karena secret sungguhan belum diisi.

## Jangan dilakukan

- Jangan `docker compose down -v` pada data yang ingin dipertahankan.
- Jangan membuka port host untuk Traefik, WAHA, 9Router, PostgreSQL, atau Redis.
- Jangan menyalakan layanan opsional selama secret-nya masih `REPLACE_WITH_...`.
- Jangan commit `.env`, token, session WhatsApp, atau password.
