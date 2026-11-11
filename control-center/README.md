# Homelab Control Center

Dashboard baca-saja untuk satu host Windows. Gate ini adalah P0-A dan P0-B. Deploy, start/stop, dan penulisan Cloudflare belum dibuka.

Control API dan UI tidak menerima Docker socket. Agent di host yang membaca Docker lewat CLI, lalu mengirim observasi yang ditandatangani.

## Menjalankan secara lokal

Dari dua terminal, di repo ini:

```bash
cd control-center/services/api
export HCC_ADMIN_PASSWORD='ganti-dengan-kata-sandi'
export HCC_AGENT_SECRET='ganti-dengan-rahasia-agent'
go run ./cmd/api
```

```bash
cd control-center/agent
export HCC_AGENT_SECRET='rahasia-yang-sama'
go run ./cmd/agent -once
```

```bash
cd control-center/apps/web
npm install
npm run dev
```

Buka `http://127.0.0.1:3000`. API menolak bind di luar loopback kecuali `HCC_ALLOW_CONTAINER_BIND=1`, dan itu hanya untuk container yang port-nya dipublikasikan ke `127.0.0.1`.

## Compose terpisah

`control-center/infra/compose.yaml` tidak mengubah `infra/compose.yaml`. Salin `control-center/infra/.env.example` menjadi file env yang tidak di-commit, isi kata sandi, lalu:

```bash
docker compose --env-file control-center/infra/.env -f control-center/infra/compose.yaml config --quiet
```

Jangan menjalankan `docker compose down -v`.

## Cloudflare

Isi `HCC_CF_API_TOKEN`, `HCC_CF_ACCOUNT_ID`, dan `HCC_CF_ZONE_ID` hanya jika token itu token manajemen dengan izin baca. Jangan menyalin token konektor tunnel ke variabel ini. Jika token tidak ada, kartu integrasi berstatus tidak dikonfigurasi dan tidak ada permintaan tulis.

## Rollback

Hentikan proses `hcc-api`, `hcc-agent`, dan `next`, atau `docker compose -f control-center/infra/compose.yaml stop`. Jangan hapus volume `hcc_data` bila ingin menyimpan metadata. Stack homelab yang sudah ada tidak perlu diubah.
