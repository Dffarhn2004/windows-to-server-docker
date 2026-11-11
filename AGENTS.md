# AGENT OPERATING CONTRACT — baca sebelum perubahan apa pun

## Tujuan
Membantu mengimplementasikan homelab Windows 11 Home + Docker Desktop WSL2 + Cloudflare Tunnel + Traefik, yang portabel ke host lain serta aman untuk banyak project.

## Hal yang WAJIB dipertahankan
1. Tidak menggunakan VirtualBox/Hyper-V VM terpisah. Docker Desktop WSL2 tetap boleh menggunakan virtualisasi internalnya.
2. Domain publik di Cloudflare; wildcard tunnel mengarah ke `http://traefik:80`, bukan ke masing-masing container.
3. Traefik `exposedByDefault=false`. Hanya aplikasi yang **secara eksplisit** diberi label dapat publik.
4. Database, cache, WAHA, dan 9Router **tidak** diberi `ports:` host. Dashboard WAHA dan 9Router boleh punya label Traefik hanya untuk `WAHA_HOST` dan `ROUTER_HOST`. Login aplikasi tetap wajib. Jangan nyalakan keduanya selama secret masih placeholder.
5. Jangan pernah melakukan commit, mencetak log, atau menampilkan nilai nyata `TUNNEL_TOKEN`, API key, password, auth session WAHA, ataupun 9Router secret.
6. Jangan memberikan mount Docker socket penuh ke aplikasi user/developer. Traefik memakai socket proxy yang dibatasi.
7. Selalu tulis langkah backup/rollback sebelum mengubah volumes dan data aktif. Jangan otomatis menghapus volumes.
8. Verifikasi command dan service name terhadap Docker Compose versi terpasang; jangan mengasumsikan docker tersedia sebelum memeriksa.
9. Endpoint backend ke private service dari network bersama menggunakan `http://waha:3000` dan `http://ai-router:20128/v1`; perhatikan kebijakan API key.

## Urutan kerja agent berikutnya
A. Baca README + docs/00-HANDOFF + docs/02-WINDOWS-SETUP + docs/03-CLOUDFLARE.
B. Audit `docker version`, `docker compose version`, `wsl --status`, status DNS/tunnel (tanpa mengungkap secrets).
C. Validasi semua compose (`docker compose config --quiet`) menggunakan env yang benar.
D. Deploy infra, hello, status. Test internal Host header dan akses publik dua subdomain.
E. Hanya setelah berhasil, on-board WAHA/9Router dan database sesuai kebutuhan user.
F. Tulis catatan perubahan, test results, blockers, dan rollback pada `docs/00-HANDOFF.md`.

## Definition of done minimal
- `docker compose` valid untuk infra + hello + status.
- Tunnel healthy dan `*.domainmu` menuju Traefik.
- `https://hello.domainmu` dan `https://status.domainmu` mengembalikan konten yang benar dari container berbeda.
- Unknown host tidak membuka service sensitif.
- Tidak ada port host database/WAHA/9Router yang dipublikasikan.
- Tidak ada secrets dalam Git dan ZIP.

## Jangan melakukan
- Tidak mengklaim sudah teruji di laptop tanpa hasil perintah nyata.
- Tidak membuat dashboard Traefik insecure pada 0.0.0.0:8080.
- Tidak mengaktifkan Docker remote API melalui TCP host.
- Tidak menjalankan deploy WAHA/9Router dengan placeholder secret.
- Tidak menempatkan banyak developer pada satu Docker daemon dengan `docker` group/socket dan menyebutnya isolasi aman.
