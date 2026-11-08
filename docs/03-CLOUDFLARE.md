# 03 — Cloudflare DNS, Tunnel, dan wildcard host

## Persiapan awal
- Domain dikelola di Cloudflare nameservers (verified active).
- Cloudflare Zero Trust dashboard: buat **remotely managed tunnel**, misal `windows-homelab`.
- Pilih connector Docker dan salin **tunnel token** ke file `.env` LOKAL sebagai `TUNNEL_TOKEN`. Jangan kirim token di chat/Git.

## Published application
Cloudflare Zero Trust → Networks → Tunnels → pilih tunnel → Add published application route (nama menu dapat berubah).

| Field | Nilai contoh |
|---|---|
| Hostname | `*.example.com` |
| Service type | `HTTP` |
| Service URL | `http://traefik:80` |

`traefik` di sini adalah DNS nama Docker service pada `homelab_edge`. `localhost` **tidak** tepat untuk cloudflared container karena mengarah ke container itu sendiri.

## Wildcard DNS
Pastikan DNS zone memiliki record `CNAME` dengan Name `*`, target `<TUNNEL_UUID>.cfargotunnel.com` (Proxied), bila tidak dibuat otomatis dari published application. Jangan menciptakan CNAME ganda; periksa dulu hasilnya. Contoh domain `example.com` harus diganti.

Cloudflare dapat membuat DNS otomatis ketika published application ditambahkan. Jika ada existing A/CNAME tepat `hello.example.com`, record spesifik dapat mengambil prioritas di atas wildcard. Akar domain `example.com` tidak otomatis tercakup.

## Tanda sukses
- Tunnel status *Healthy/Connected* di dashboard.
- `https://hello.example.com` benar-benar menampilkan "Hello dari Docker".
- `https://status.example.com` benar-benar menampilkan "Status service aktif".
- Tidak perlu open inbound port 80, 443, 3000, 20128, atau port forwarding router.

## Keamanan
- HTTPS public berhenti di Cloudflare edge; origin `cloudflared -> Traefik` HTTP di network Docker lokal. Lindungi host Docker.
- Cloudflare Access hanya dibutuhkan bila mempublikasikan panel/route private; tunnel **sendiri bukan otorisasi user**.
- Untuk WAHA/9Router: lebih baik private tanpa route sama sekali. Jika nanti butuh dashboard admin, buat dedicated protected app + Access, autentikasi aplikasi, dan evaluasi konfigurasi.

Docs: https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/routing-to-tunnel/ dan https://developers.cloudflare.com/dns/manage-dns-records/reference/wildcard-dns-records/
