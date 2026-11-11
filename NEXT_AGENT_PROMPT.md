# Prompt Cursor Agent — implementasi Homelab Control Center

**Mulai di sini untuk tugas dashboard.** Baca `docs/19-HOMELAB-CONTROL-CENTER-PRD.md` dan `docs/20-CURSOR-CONTROL-CENTER-IMPLEMENTATION.md` secara penuh, lalu `AGENTS.md`, `README.md`, `docs/00-HANDOFF.md`, `infra/compose.yaml`, dan docs keamanan yang direferensikan.

Saya ingin membangun dashboard **Homelab Control Center** pada repo yang sudah bekerja, **bukan mengulang setup manual**. Implementasi bertahap P0-A (fondasi) → P0-B (read-only Docker dashboard) → P0-C (safe deploy wizard) → P0-D (Cloudflare exact-hostname + Access); catat bukti testing per gate. Stack target UI Next.js/TypeScript/Tailwind, Control API Go, agent Go native Windows terpisah, SQLite, integrasi Docker/Traefik/Cloudflare.

**Hard safety rules:** jangan mengubah VPS dan wildcard `*.imuii.id`, jangan overwrite `infra/compose.yaml` tanpa approval, jangan menghapus volume/session WAHA/9Router/CVAT, jangan memount Docker socket ke frontend/API, jangan membuka host port ke publik, jangan mengungkap/commit `.env` atau secret. Jika Cloudflare management API token belum ada, gunakan mock dan read-only mode, jangan meminta token di chat.

Ikuti task tracker, acceptance criteria, dan format pelaporan pada `docs/20-CURSOR-CONTROL-CENTER-IMPLEMENTATION.md`. Audit keadaan mesin terlebih dulu; jangan menganggap hasil percakapan lama adalah status runtime saat ini. Update `docs/00-HANDOFF.md` dengan perubahan, tes nyata, risiko, dan rollback. Stop setelah P0-D kecuali saya meminta scope tambahan.

## Prompt awal (arsip / konteks PoC)

Panduan PoC manual awal disimpan di `docs/01-ARCHITECTURE.md`, `docs/03-CLOUDFLARE.md`, dan `docs/18-QUICKSTART-LAPTOP-BARU.md`. Jangan mengikuti langkah pembuatan Tunnel baru untuk tugas dashboard karena Tunnel existing sudah diuji.
