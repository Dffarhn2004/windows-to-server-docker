# 07 — Model keamanan dan ancaman

## Asumsi trust boundary
Internet/Cloudflare adalah lapisan publik; Tunnel adalah jaringan transport, **bukan autentikasi**. Traefik bisa routing ke host publik berlabel. Container backend yang mengakses WAHA/9Router berada di shared private service network. Administrator host dan Docker Engine memiliki hak luas.

## Checklist mandatory
- [ ] Tidak publish `ports:` untuk WAHA, 9Router, PostgreSQL, Redis, Docker socket, atau Traefik dashboard.
- [ ] Traefik `exposedByDefault=false`.
- [ ] Cloudflare Access + auth aplikasi untuk dashboard private yang sengaja dipublish.
- [ ] Root `.env` tetap lokal, `.gitignore` aktif; rotate secret bila bocor.
- [ ] Docker socket proxy `POST=0`, hanya Traefik akses network `docker-control`.
- [ ] Host Windows updated; user Windows menggunakan lock screen, BitLocker/device encryption jika tersedia.
- [ ] Pin image versions/digests sebelum production; audit CVE & updates.
- [ ] API key WAHA, 9Router, dan webhook signature aktif dan tidak masuk logs.
- [ ] Ada backup offline + uji restore.
- [ ] Gunakan firewall Windows dan jangan buka port WAN router.

## Risiko penting
1. **Docker group/socket setara admin daemon**: jangan diberikan ke developer yang tidak tepercaya.
2. **Local machine compromise**: seluruh data WAHA dan 9Router dapat ikut terungkap.
3. **DNS hijack/Cloudflare token leak**: segera revoke/rotate tunnel credential.
4. **Public app vulnerability**: label Traefik tidak menambah autentikasi aplikasi; tambahkan security control framework/WAF/rate limits sesuai risiko.
5. **Network flatness**: `homelab_services` adalah network bersama, bukan isolasi tenant yang kuat. Pisahkan network bila kebutuhan privacy multi-tenant meningkat.

## Khusus akun developer
Jangan langsung menyediakan 1 Ubuntu container ber-SSH dengan akses `/var/run/docker.sock` untuk semua developer. Alternatif yang lebih aman: Git repository per project → CI/CD deployment pipeline → policy/review → deployment runner yang haknya terbatas; host shell tetap untuk admin tepercaya.
