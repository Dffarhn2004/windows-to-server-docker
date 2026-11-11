# Cursor Implementation Handoff — Homelab Control Center

> **Input:** [PRD lengkap](./19-HOMELAB-CONTROL-CENTER-PRD.md)  
> **Repo:** `Dffarhn2004/windows-to-server-docker` · default branch `main`  
> **Tujuan:** meminta Cursor Agent **mengimplementasikan**, bukan mengulangi setup manual Cloudflare/Docker.  
> **Tanggal handoff:** 11 November 2026  
> **Status:** Design only. Dashboard belum diimplementasikan.

## 0. Prompt siap salin ke Cursor Agent

> Saya ingin mengimplementasikan **Homelab Control Center (HCC)** di repo ini berdasarkan `docs/19-HOMELAB-CONTROL-CENTER-PRD.md` dan `docs/20-CURSOR-CONTROL-CENTER-IMPLEMENTATION.md`. Baca `AGENTS.md`, `README.md`, `infra/compose.yaml`, `docs/00-HANDOFF.md`, `docs/07-SECURITY.md`, `docs/14-DECISIONS.md`, dan compose project existing **sebelum mengubah kode**. Jangan mendesain ulang homelab.  
>
> **Prioritas:** P0-A → P0-B → P0-C → P0-D, kerjakan per gate dan laporkan hasil tes nyata di setiap gate. Buat control plane Next.js + Go, agent Go Windows terpisah dengan job typed/allowlisted (tidak memberikan Docker socket ke browser/control plane), SQLite, deploy wizard, Docker discovery/operations, dan Cloudflare exact-hostname automation.  
>
> **Keamanan hard stop:** jangan ubah `A *` / apex / DNS production di `imuii.id`, jangan modifikasi `infra/compose.yaml` yang sudah bekerja tanpa bukti kebutuhan dan approval, jangan menghapus volume WAHA/9Router/CVAT, jangan commit `.env`, rahasia, atau kredensial. Existing WAHA `waha-v2.imuii.id`, Hello `hello.imuii.id`, dan VPS harus tetap berfungsi. Jika credentials Cloudflare management API belum tersedia, implementasikan provider interface + mock dan tandai live deploy blocked, **jangan mengarang API token**.  
>
> **Workflow:** audit → ADR singkat/struktur folder → test-first mocks → implementasi → lint/test/build → local Docker tests → Cloudflare integration **hanya setelah explicit opt-in** → update `docs/00-HANDOFF.md` dengan status terverifikasi, commands, hasil, risiko, rollback. Kode dan docs disimpan di repo. Jangan menyatakan selesai sebelum acceptance PRD lulus.

## 1. Fakta existing yang sudah diketahui

Dari sesi pengguna 2026-11-11 (bukan live shell yang diakses agent saat membaca file):
- `docker version`: Docker Desktop Linux Engine berfungsi.
- Network `homelab_proxy` pernah dibuat manual, tetapi **repo menggunakan** `homelab_edge` dan `homelab_services`. Jangan menggantinya dengan `homelab_proxy`.
- `scripts/start-infra.ps1 -WithExamples` menjalankan `docker-socket-proxy`, `traefik`, `cloudflared`, `hello`, `status`.
- `scripts/test-routing.ps1`: Hello 200, Status 200, unknown host 404.
- Cloudflare Zone `imuii.id` dengan Tunnel `homelab-docker` (remotely managed); spesifik hostname `hello.imuii.id` dan `waha-v2.imuii.id` dipublikasikan.
- `WAHA` Compose berhasil start dengan image `devlikeapro/waha:latest-2026.10.1`; Dashboard Cloudflare Access + WAHA worker 1 connected, 0 WhatsApp sessions saat screenshot.
- Existing independent Docker services meliputi CVAT dan 9Router; jangan diasumsikan managed oleh HCC.
- Wildcard `A *` masih menunjuk VPS; record `api.imuii.id` juga VPS. User **belum meminta migrasi wildcard**.
- Repo `docs/00-HANDOFF.md` dan sebagian README masih mendeskripsikan pre-domain PoC, jadi lakukan audit ulang, lalu catat status terbaru secara append-only.

**Sekarang:** HCC belum ada. Jangan menyatakan HCC running hingga kode ditulis dan benar-benar dites.

## 2. Batasan yang tidak bisa dinegosiasikan

1. Jangan menimpa `infra/compose.yaml`, jalur Traefik, Socket Proxy, atau naming networks. Jika perlu perubahan infra, pisah PR dan minta review dengan bukti regresi.
2. Semua fitur yang bisa mengeksekusi Compose hanya lewat **Agent** dengan operasi bertipe, validation policies dan auth kuat. Tidak ada raw terminal input dari browser dan tidak ada Docker socket di Next.js / Go Control API.
3. VPS production & wildcard DNS tidak disentuh. Hanya tambah **specific hostname** (mis. `hcc-smoke.imuii.id`) setelah preflight conflict scan.
4. Cloudflare API token khusus, bukan `TUNNEL_TOKEN`, jangan gunakan Global API Key. Read-only mode harus tetap berfungsi bila API token tidak ada.
5. Deploy web dashboard harus dilindungi Cloudflare Access + aplikasi login/session. Jangan buat public route dashboard sebelum auth berfungsi.
6. Never expose container ports ke `0.0.0.0`; dashboard components harus mengikuti network/no-host-port policy.
7. Semua stateful volumes (WAHA session/media, 9Router, databases, CVAT) dianggap data berharga. Tidak pernah `down -v`, `volume prune`, atau reset di automated scripts.
8. Multi-host hanya domain-model/API-ready, bukan deploy ke dua laptop bersamaan pada MVP.
9. Existing containers ditampilkan **Unmanaged** read-only; jangan auto-adopt.
10. Cloudflare tunnel config PUT adalah **replace all**: preserve existing ingress, originRequest, catch-all, and order. Sebelum write: GET, diff, validate, snapshot; setelah: GET/compare. Jika perubahan eksternal terdeteksi, ABORT.
11. Simpan setiap keputusan yang materially berbeda dari PRD dalam ADR; jangan mengubah scope diam-diam.

## 3. Struktur kode yang disarankan

`control-center/`
- `apps/web/`: Next.js + TypeScript + Tailwind, reusable UI components, typed API client, no secrets.
- `services/api/`: Go HTTP server, modules `auth`, `hosts`, `apps`, `deployments`, `jobs`, `audit`, `cloudflare`, `storage`.
- `agent/`: Go native Windows daemon (service or foreground dev), Docker/Compose command executor, typed job schema, heartbeat, signed request validation.
- `contracts/`: OpenAPI and JSON schemas / fixtures.
- `infra/`: additive Compose for web/api plus local development dependencies; don't modify root infra by default.
- `tests/`: contract tests, mocked Cloudflare/Docker, e2e scaffolding.
- `README.md`: installation, prerequisites, one-host connection, dev workflow, safety, rollback.

`managed-projects/`: generated runtime Compose + metadata, **gitignored**; no secret values stored in version control.

Prefer Go standard lib / well-maintained packages, clear handler/service/repository/provider interfaces, structured logging; TS form validation & typed errors. Do not over-engineer a platform with multiple queues/brokers at P0.

## 4. Gate plan & task tracker

### Gate P0-A — Scaffold + architecture, zero live mutations

| Task | Deliverable | Verify |
|---|---|---|
| A01 | Read repo/docs, record 2026-11-11 baseline & exact running names | No changes to live containers |
| A02 | Short ADR on control plane/agent transport, secret store and Cloudflare ownership | Reviewable design notes |
| A03 | Next.js UI shell & navigation | `lint`/`build` pass |
| A04 | Go API skeleton + OpenAPI v1 + standard error contract | `go test ./...` pass |
| A05 | SQLite schema + migrations + seed test fixtures | Migration up/down and concurrency tests |
| A06 | Authn/authz middleware + session threat model | Anonymous mutation denied |
| A07 | DockerProvider / CloudflareProvider / AgentTransport interfaces + mock implementations | Contract tests pass |
| A08 | Gitignore runtime/secret artifacts + app README | No creds in repo |

Gate outcome: a running UI/API **in development mode only**, all external mutation providers mocked, no production route changed.

### Gate P0-B — Read-only inventory & operator dashboard

| Task | Deliverable | Verify |
|---|---|---|
| B01 | Windows native Go Agent polling endpoint / heartbeat with per-host secret | Reject invalid signature/replay |
| B02 | Docker discovery (containers, network state, metadata, versions) | Unmanaged vs managed properly marked |
| B03 | Overview cards + Hosts table + Applications table | Handles offline/unknown states |
| B04 | Logs API (bounded tail + redaction) | No secrets in response |
| B05 | Tunnel status + DNS route read-only via scoped API | No write when token not configured |
| B06 | Audit events & job list UI with fixtures | View-only path works |

Gate outcome: read-only HCC views actual host and foreign CVAT/9Router without mutating their state. Docker Engine must not be mounted in UI/API.

### Gate P0-C — Controlled app deploy, Docker-only

| Task | Deliverable | Verify |
|---|---|---|
| C01 | Manifest schema / image allowlist / Compose generator | Invalid input rejected, deterministic output |
| C02 | "Add Application" wizard + review diff | No shell input, no arbitrary Compose |
| C03 | Agent deploy command allowlist; target project path sandbox | Reject `privileged`, bind mount, socket, host net, user-provided commands |
| C04 | SQLite jobs, steps and idempotency logic | Same idempotency key = one deploy |
| C05 | Deploy local internal-only Nginx fixture | No public route, no host port |
| C06 | Start/stop/restart/log/remove (preserve volumes) | Only owned project mutable |
| C07 | Rollback and failure simulation | Existing services untouched |

Gate outcome: internal-only app deploys from HCC; arbitrary Docker control from browser impossible. Do not connect Cloudflare writes yet.

### Gate P0-D — Exact Cloudflare hostname + Access

| Task | Deliverable | Verify |
|---|---|---|
| D01 | Cloudflare read-only snapshot: DNS/ingress/Access + token permission checks | Existing manual routes preserved |
| D02 | Exact FQDN conflict checker (DNS/tunnel/Traefik/DB) | `api`, `waha-v2` conflict blocked |
| D03 | Access app/policy provisioning or verified reusable policy linking | Private route fails closed |
| D04 | Tunnel ingress config safe merge, immutable snapshot, serialization | Full config preserved incl. catch-all 404 |
| D05 | DNS exact CNAME create/delete owned-only | Wildcard `*` unchanged |
| D06 | End-to-end wizard deploy protected demo | HTTPS works via Tunnel → Traefik → new container |
| D07 | Drift detection + owner-aware rollback | Manual changes not overwritten |
| D08 | E2E auth/security regression, backups/docs | Reproducible proof, approved cleanup |

Gate outcome: operator can deploy `hcc-smoke.imuii.id` **after explicit user opt-in**. Until opt-in, use mocked Cloudflare API and dry run. For destructive cleanup, ownership and targeted IDs must be verified.

### P1/P2 backlog (do not implement in current scope)

- Backups/restore; safe app update/release history.
- GitHub build & registry pipeline, scoped project access, notifications.
- Multi-host onboarding, secure node identity rotation, agent upgrade, approvals, separate developer access.
- Prometheus resource history, richer monitoring, UX improvements.
- Blue/green rollout and HA only after real storage/state coordination design.

## 5. Minimum API contract & test helpers

Follow PRD §9 endpoint proposal. Maintain:
- Web mutations require authenticated admin session + anti-CSRF + `Idempotency-Key`.
- Agent actions require host-specific auth, job signature, expiry, replay defense.
- `GET /health` must not expose secrets; include readiness checks in private endpoint.
- `GET /apps` returns owner flag; unmanaged actions disabled both UI **and server**.
- `POST /apps/validate` returns structured preview, conflicts, policy violations **without side effects**.
- `POST /apps` returns `jobId` + safe state; asynchronous status through `GET /jobs/{id}`.
- Persist names of external owned resources: DNS record ID, Access app ID, tunnel ID and ingress fingerprint, Compose path and project ID.
- Error codes match PRD §9.

Recommended tests: Go table tests for validation/policy/merge, TypeScript unit for wizard, API contract test, Docker sandbox integration with unique HCC-owned project, mocked Cloudflare 429/409/5xx, failure-injection rollback, browser workflow E2E.

**Important:** before running any test touching real DNS, show the proposed hostname and require explicit operator opt-in and cleanup plan. Don't run live tests from CI without secrets/approved sandbox.

## 6. Cloudflare implementation traps

- Tunnel connector token is **NOT** a management token. Store separately.
- Tunnel `PUT /configurations` **replaces** remote ingress. Get current config first; merge carefully; preserve existing rules and trailing catch-all; abort on drift.
- If user has manual hostname route (e.g. `hello.imuii.id` or `waha-v2.imuii.id`), do not attempt ownership or delete it.
- Exact DNS record must not collide with existing A/CNAME; Cloudflare DNS rejects mixed A + CNAME same name.
- VPS wildcard `A *` still owns unknown hostnames. HCC must create individual CNAMEs; do not create wildcard CNAME until user separately approves a migration plan, outside MVP.
- Access auth should exist before routing a private app publicly. Distinguish browser login versus API machine authentication via Access service tokens.
- Do not use Global API Key or an all-zones token.
- Recheck official API docs rather than assuming older SDK version response shapes.

Official:
- https://developers.cloudflare.com/api/resources/zero_trust/subresources/tunnels/subresources/cloudflared/subresources/configurations/
- https://developers.cloudflare.com/api/resources/dns/subresources/records/methods/create/
- https://developers.cloudflare.com/api/resources/zero_trust/subresources/access/subresources/applications/
- https://developers.cloudflare.com/cloudflare-one/access-controls/service-credentials/service-tokens/

## 7. Local execution commands (do not run destructive commands)

From repo root on Git Bash:

`docker version`  
`docker compose version`  
`powershell.exe -NoProfile -ExecutionPolicy Bypass -File ./scripts/validate-core.ps1`  
`powershell.exe -NoProfile -ExecutionPolicy Bypass -File ./scripts/test-routing.ps1`

Existing homelab can be inspected read-only with `docker ps`, `docker network ls`, `docker logs --tail=100 <name>`. **Do not restart the existing infra simply to develop HCC**.

For new Go code, build/test commands are defined by implementation module path, e.g. `cd control-center/services/api && go test ./...`; for Next.js use npm scripts from `control-center/apps/web`. Don't assume a package manager until lockfile is chosen.

**No-go:** `docker compose down -v`, `docker system prune --volumes`, `docker volume rm`, deleting foreign containers, altering tunnel token, changing DNS wildcard.

## 8. Done/blocked decision table

| Situation | Behavior |
|---|---|
| Go API cannot authenticate Agent | Show Offline / deny operations |
| Docker daemon unavailable | No deploy; show troubleshooting |
| Cloudflare API token absent | Monitoring/deploy internal-only works, public wizard disabled with clear guidance |
| Access policy/API permissions missing | Private publish blocked |
| DNS collision with existing VPS/WAHA | Block, show conflict details; never overwrite |
| Tunnel read/write config drift | Abort, snapshot, ask admin to resolve |
| New app healthcheck fails | Compensate owned changes; preserve existing services |
| Unmanaged container action requested | 403/409, no mutation |
| Database migration fails | Do not deploy; rollback DB backup, report |
| Tests skipped | Report explicitly as **not tested**, never mark green |

## 9. Cursor deliverable format for each gate

1. **Implemented:** files added/changed and brief API/UI summary.
2. **Commands actually run:** exact commands + pass/fail outputs, not estimates.
3. **Acceptance tests:** PRD items checked, pending, evidence.
4. **Existing infrastructure safety:** changes to `infra/compose.yaml`? external Cloudflare? Docker? secrets? Answer explicitly.
5. **Rollback:** commands/instructions & backup assets.
6. **Next gate:** prerequisites, blockers, user approvals required.
7. **Update** `docs/00-HANDOFF.md` with append-only progress ledger (include date and accurate tested statuses).

**Definition of success:** safe functional dashboard, not just nice screenshots or mocked cards. The original VPS services, WAHA, and the Hello route must remain intact.
