# 14 — Architecture Decision Records (ADR)

## ADR-001: Docker Desktop WSL2, bukan VirtualBox
Status: Accepted. User memilih Windows 11 Home dan tidak ingin VM VirtualBox. Konsekuensi: startup dipengaruhi Windows/login, bukan setara Ubuntu VPS production.

## ADR-002: Cloudflare Tunnel + Traefik
Status: Accepted. Wildcard routing Cloudflare hanya perlu dikonfigurasi sekali; Traefik discovery berbasis Docker labels. Cloudflare bisa routing multi-host tanpa Traefik, namun maintainability project banyak lebih baik dengan Traefik.

## ADR-003: Private services tidak diberi public ports
Status: Accepted. WAHA/9Router/data private by default; hanya backend dapat mengakses lewat shared network. Bukan pengganti authorization setiap service.

## ADR-004: Docker socket proxy
Status: Accepted untuk starter. Read-only limited proxy mengurangi attack surface Traefik dibanding direct Docker socket, meskipun masih ada metadata exposure.

## ADR-005: Satu active host tunnel untuk migrasi
Status: Accepted untuk cold migration. Dua laptop boleh aktif untuk skenario HA di masa depan, tetapi butuh shared state, traffic management, dan security khusus.

## ADR-006: Compose per project
Status: Accepted. Infastruktur + dua demo + tiap optional service file terpisah. Menambah project tidak perlu mengedit Compose utama; cukup join shared network dan buat Traefik labels.
