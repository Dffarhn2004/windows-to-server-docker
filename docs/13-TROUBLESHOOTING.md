# 13 — Troubleshooting

| Gejala | Pemeriksaan | Perbaikan umum |
|---|---|---|
| Tunnel 1033 / offline | `docker compose ... logs cloudflared`, dashboard tunnel healthy | Token benar, outbound internet aktif, container running |
| 502/Bad Gateway | `docker compose ... logs traefik`, Docker DNS `traefik`, target port app | Port container sesuai label, network `homelab_edge` terhubung |
| Traefik 404 | Host rules tidak match; wildcard route/DNS | Domain sama persis, label router ada, contoh kontainer aktif |
| DNS resolves wrong service | Exact DNS record overriding wildcard | Audit Cloudflare records, hapus hanya yang dipahami |
| Docker daemon not running | `docker info` | Start Docker Desktop, pilih Linux containers |
| Socket proxy 403 | `docker-socket-proxy` permissions | Debug endpoint read-only yang dibutuhkan saja, jangan enable POST |
| WAHA session hilang | Volume `/app/.sessions` terpasang? | Restore volume dan cek WAHA engine/config |
| 9Router data hilang | `/app/data` dan `DATA_DIR=/app/data` | Restore named volume, backup SQLite konsisten |
| App redirect http | Public base URL/cookies behind Cloudflare HTTPS | Set `https://` external origin, cek trusted proxy/cookie framework |
| Server offline setelah reboot | Docker Desktop belum login / Windows sleep | Autostart on sign-in, disable sleep, monitor uptime |

Command inspeksi:
```powershell
docker network inspect homelab_edge
docker network inspect homelab_services
docker compose --env-file .env -f infra/compose.yaml logs --tail 150 traefik cloudflared
docker ps -a
```

Jika `docker compose` error, jangan langsung menghapus volumes. Uji `config` dan pahami syntax/version mismatch.
