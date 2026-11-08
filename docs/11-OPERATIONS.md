# 11 — Operasional harian

## Monitoring cepat
```powershell
docker ps
docker stats
docker compose --env-file .env -f infra/compose.yaml ps
docker compose --env-file .env -f infra/compose.yaml logs --tail=100 traefik cloudflared
```

## Update image
Sebelum update: backup dan catat image yang dipakai. Lalu:
```powershell
docker compose --env-file .env -f infra/compose.yaml pull
docker compose --env-file .env -f infra/compose.yaml up -d
```
Pin versions dan uji dampak sebelum update service stateful. Jangan pakai `docker system prune --volumes` sembarangan.

## Startup Windows
Aktifkan Docker Desktop autostart **upon Windows user sign-in**. `restart: unless-stopped` berlaku setelah Docker Engine benar-benar berjalan. Sleep Windows, reboot update, atau listrik mati tetap menyebabkan downtime.

## Logging / backup
Jadwalkan backup file dan database sensitif ter-enkripsi. Batasi log, jangan mencetak key. Untuk availability serius gunakan UPS/backup internet atau pindah ke VPS.
