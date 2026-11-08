# 04 — Traefik dan routing berbasis Docker labels

## Konfigurasi pusat
`infra/compose.yaml` menjalankan Traefik di network `homelab_edge`, dan socket-proxy hanya untuk membaca Docker metadata. `exposedByDefault=false` berarti semua container tersembunyi sampai explicit opt-in melalui label.

## Label minimum
```yaml
labels:
  - "traefik.enable=true"
  - "traefik.docker.network=homelab_edge"
  - "traefik.http.routers.myapp.rule=Host(`myapp.example.com`)"
  - "traefik.http.routers.myapp.entrypoints=web"
  - "traefik.http.services.myapp.loadbalancer.server.port=3000"
```

Nama router/service **harus unik** di seluruh daemon. Port 3000 adalah port **di dalam container**, tidak dipublish ke Windows host.

## Mengapa socket proxy?
Montase Docker socket walaupun `:ro` masih sensitif. Karena itu Traefik memakai `tecnativa/docker-socket-proxy` di network `docker-control` yang private, `POST=0`, dan API sections terbatas. Layanan user jangan ditempatkan dalam network `docker-control`. Ini pengurangan risiko, bukan isolasi mutlak; izin Docker metadata tetap mengungkap detail container.

## HTTP dan HTTPS
Pengguna mengakses `https://myapp.example.com` pada Cloudflare. Koneksi dari `cloudflared` ke Traefik adalah HTTP internal. Jika framework butuh origin eksternal, atur public base URL menjadi HTTPS dan validasi redirect/cookies. Jangan set `forwardedHeaders.insecure=true` hanya demi bypass kesalahan konfigurasi.

## Beda port dan subdomain
Banyak hostname diterima Cloudflare pada 443, Traefik memilih service name & port internal. **Tidak ada batas satu port per tunnel**.

Docs: https://doc.traefik.io/traefik/reference/install-configuration/providers/docker/ dan https://github.com/Tecnativa/docker-socket-proxy
