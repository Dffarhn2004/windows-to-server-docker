# 08 — Developer dan tata kelola project

## Awal (satu admin tepercaya)
Admin menggunakan Docker CLI dari Windows PowerShell atau WSL2. Developer menyerahkan PR pada repository project, proses deploy tetap dilakukan oleh admin.

## Tahap berikutnya (beberapa developer)
1. Setiap project punya repo dan Compose sendiri.
2. GitHub Actions / GitOps mengirim artefak atau build image ke registry.
3. Server menggunakan deploy agent terkontrol untuk hanya update project yang berhak.
4. Secrets terpisah tiap project; network dibatasi berdasarkan kebutuhannya.
5. Domain/subdomain dikelola melalui template Traefik label, tanpa edit cloudflared setiap deploy.

## Batasan
Container Ubuntu ber-SSH **tidak** otomatis mengisolasi developer jika socket Docker mounted. Jangan memberikan akses Docker daemon penuh dan menyebutnya restricted access. Traefik/Docker Compose adalah alat routing/orchestration, bukan RBAC multi-tenant.

## Future enhancement
Dedicated VM/Linux host + deploy service RBAC, isolated runner, role-specific policies, observability, audit log, quota, secret manager.
