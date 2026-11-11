# PRD — Homelab Control Center

> **Status:** Proposed / ready for implementation planning  
> **Version:** 1.0 · **Tanggal:** 11 November 2026  
> **Repo:** https://github.com/Dffarhn2004/windows-to-server-docker  
> **Dokumen eksekusi Cursor:** [20-CURSOR-CONTROL-CENTER-IMPLEMENTATION.md](./20-CURSOR-CONTROL-CENTER-IMPLEMENTATION.md)  
> **Bahasa produk:** Bahasa Indonesia (nama teknis tetap English).  
> **Peran dokumen:** spesifikasi produk, perilaku, keamanan, API, model data, acceptance criteria. **Bukan klaim bahwa dashboard sudah dibuat.**

## 1. Executive summary

Bangun **Homelab Control Center (HCC)**: dashboard web yang mengurangi proses manual menambahkan aplikasi ke homelab Windows 11 Home. Admin dapat melihat aplikasi dan host, memilih template/image, mengatur hostname, keamanan akses, lalu melakukan deploy, start/stop/restart, melihat log, dan rollback dari UI. Proses teknis Docker Compose + Traefik + Cloudflare DNS/Tunnel dijalankan secara terkontrol oleh backend dan deploy agent.

**Prinsip:** extend repositori/infrastruktur yang sudah ada, jangan rebuild. HCC adalah *control plane*, bukan pengganti Docker Desktop, Traefik, Cloudflare Tunnel, atau VPS. Desain data mendukung multi-host, tetapi MVP mengelola **satu Windows host tepercaya** terlebih dahulu.

**Asumsi & batas kenyataan:** berdasarkan pengujian pengguna pada 11 Nov 2026, routing publik `hello.imuii.id` → Tunnel → Traefik → Docker sudah menampilkan Hello; `waha-v2.imuii.id` berhasil membuka WAHA dan API worker berstatus connected; `scripts/test-routing.ps1` menghasilkan 200/200/404. Repo belum memiliki dashboard HCC. Sebagian `README.md`/`docs/00-HANDOFF.md` masih mencatat PoC lama; **jangan menganggap dokumen historis itu status live**. Agent wajib revalidasi lingkungan.

## 2. Problem statement

Proses manual saat ini berulang: edit `.env`, edit/clone Compose + Traefik labels, deploy Docker via terminal, masuk Cloudflare untuk DNS + published application, buat Cloudflare Access untuk aplikasi privat, cek HTTPS/health, periksa log dan volume. Ini memakan waktu, rawan typo/salah port/salah hostname, dan berisiko menimpa wildcard DNS yang sudah dipakai VPS.

HCC harus mengubah workflow menjadi **satu wizard terkontrol** dengan preview, validasi, status langkah, audit, dan rollback; tetap memungkinkan pembacaan kondisi container existing tanpa mengambil alih secara diam-diam.

## 3. Goals, non-goals, prinsip

### 3.1 Goals terukur

| ID | Sasaran | Metrik penerimaan |
|---|---|---|
| G-01 | Satu tampilan untuk kondisi host & aplikasi homelab | Status host, Tunnel, Traefik, apps, health, last deployment dapat ditampilkan |
| G-02 | Deploy aplikasi template ke Docker tanpa edit Compose manual | Aplikasi demo baru berhasil deploy dari wizard tanpa mengedit infra existing |
| G-03 | Subdomain spesifik otomatis dan aman | Cloudflare DNS + route tercipta, tanpa menyentuh `A *` yang menuju VPS |
| G-04 | Private-by-default | App privat baru tidak bisa diakses publik tanpa policy Access yang valid |
| G-05 | Operasi harian sederhana | Start, stop, restart, logs, inspect dan delete non-destruktif dari UI |
| G-06 | Troubleshooting terbaca | Job steps, error, suggested resolution, audit tanpa menampilkan secrets |
| G-07 | Portabilitas arsitektur | Model `HostNode`/`Agent` siap untuk laptop/VPS kedua; implementasi multi-host bukan syarat MVP |

**Target operasional (target desain, belum benchmark):** deploy aplikasi stateless sederhana dalam <5 menit setelah image tersedia dan koneksi normal; feedback job pertama muncul dalam <3 detik; polling status maksimum 15 detik. Jangan menjanjikan performa yang belum diuji.

### 3.2 Non-goals MVP

- Tidak membuat Kubernetes, Swarm, Nomad, atau multi-host HA.
- Tidak menghapus/memigrasi WAHA existing, CVAT, 9Router, atau layanan VPS otomatis.
- Tidak otomatis mengubah nameserver, wildcard DNS `*`, apex domain, DNSSEC, MX/SPF/DKIM.
- Tidak memberikan unrestricted terminal/SSH, editor Compose mentah, `docker run --privileged`, atau akses `docker.sock` ke browser/frontend.
- Tidak mendukung banyak developer tidak tepercaya pada satu Docker daemon.
- Tidak menyediakan billing, deployment marketplace, CI/CD lengkap, secret vault produksi, database migration wizard, atau zero-downtime.
- Tidak menganggap login Cloudflare Access sebagai satu-satunya lapisan autentikasi HCC; auth aplikasi/control API tetap wajib.
- Tidak menyediakan penghapusan volume/backup otomatis dalam operasi delete biasa.

### 3.3 Prinsip desain

1. **Reuse infrastructure:** `infra/compose.yaml`, `homelab_edge`, `homelab_services`, `docker-control`, Socket Proxy, label Traefik, Compose per project.
2. **Least privilege & explicit opt-in:** `exposedByDefault=false`, dashboard publik dan WAHA privat wajib Access, secrets tidak muncul di UI response/log.
3. **Read before write:** preview perubahan dan deteksi konflik sebelum perubahan Docker/Cloudflare.
4. **Owned resources only:** HCC hanya otomatis memodifikasi/hapus sumber daya yang dibuat/adopt secara eksplisit oleh HCC.
5. **Fail closed:** kegagalan Access, DNS, Tunnel, atau validasi menahan publikasi.
6. **Reversible:** persist job steps + last-known-good snapshot + rollback terencana.
7. **Idempotent & observable:** retry tidak menduplikasi apps/DNS/routes; setiap operasi punya job ID dan audit event.

## 4. Konteks infrastruktur existing (source of truth)

| Komponen existing | Referensi repo | Konsekuensi untuk HCC |
|---|---|---|
| Infra Compose | `infra/compose.yaml` | **Jangan overwrite**; dashboard stack terpisah |
| Docker edge network | `homelab_edge` | Web app publik hanya lewat label Traefik di network ini |
| Service network | `homelab_services` | Komunikasi backend/private apps sesuai kebutuhan |
| Docker socket proxy | `docker-socket-proxy` | Khusus discovery Traefik, **bukan** tempat pemberian hak deploy |
| WAHA | `projects/waha/compose.yaml` | Existing, memiliki volumes `homelab_waha_sessions` dan `homelab_waha_media`; read-only discovery dulu |
| Hello & Status | `projects/hello/compose.yaml`, `projects/status/compose.yaml` | Golden test untuk routing |
| Tunnel | `homelab-docker` (Cloudflare, remotely managed) | Reuse; jangan buat tunnel baru otomatis |
| Domain | `imuii.id` | Domain di Cloudflare; `A api` dan `A *` historically ke VPS; verify live before any write |
| CLI/skrip | `scripts/*.ps1` | Existing dapat dipanggil/ditest; hindari fork logika berulang |

**Kondisi khusus:** ada Docker container non-homelab (misalnya CVAT, 9Router existing) di host; HCC boleh menampilkan sebagai *Unmanaged*, tetapi dilarang memodifikasi tanpa proses adoption dan validasi ownership.

## 5. Persona, akses, use cases

**Primary:** owner/admin homelab (satu operator tepercaya), ingin deploy banyak aplikasi dari GUI.

**Secondary (fase lanjut):** operator/developer terbatas yang hanya melihat atau meminta deploy project yang diizinkan. Perbedaan hak di tabel:

| Aksi | Admin MVP | Operator P2 | Viewer P2 |
|---|---:|---:|---:|
| Lihat status/health/log ter-redaksi | Ya | Per project | Per project |
| Deploy dari template yang disetujui | Ya | Dengan approval | Tidak |
| Operasi start/stop/restart | Ya | Project sendiri | Tidak |
| Ubah hostname, Cloudflare, Access | Ya + confirmation | Tidak | Tidak |
| Kelola credential, secrets, host | Ya | Tidak | Tidak |
| Hapus project (tanpa volume) | Ya + double confirmation | Tidak | Tidak |

Use cases: lihat kesehatan host; tambah nginx demo; tambah backend HTTP image; onboard domain spesifik `foo.imuii.id`; pilih private + Access policy; lihat errors rollout; restart service; hapus demo tanpa menghapus volume; identifikasi drift jika edit Docker/Cloudflare dari luar.

## 6. UX & information architecture

### 6.1 Navigasi utama

- **Overview:** host status, CPU/RAM/disk, Docker daemon, tunnel connected, jumlah apps by state, recent jobs/errors.
- **Applications:** Managed/Unmanaged tabs, name, image, hostname, service port, host, state, visibility, health, actions.
- **Deploy Application (wizard):** source → runtime → domain/visibility → env/volumes → review & security → deploy progress → result.
- **Deployments/Jobs:** queued/running/succeeded/failed/rolled_back, progress step, logs sanitasi, retry safely.
- **Domains & Routes:** DNS/routes yang owned vs external, namespace conflicts, Access badge, read-only VPS-bound records.
- **Hosts:** Windows host saat ini; add/register host P2; heartbeat, Docker info, capacity.
- **Secrets & Integrations:** Cloudflare credentials health/permission state; masked secret references (no reveal); token rotation instructions.
- **Settings & Audit:** administrator login, maintenance mode, protected hostnames, history/rollback logs.

### 6.2 Wizard Add Application

Step 1 **Identitas:** unique project slug (`[a-z0-9-]`, lowercase, length limit), display name, target host, owner scope.

Step 2 **Source:** MVP: allowlisted `Docker image:tag/digest` atau preset template (Nginx static/basic HTTP). Fase selanjutnya GitHub repo/build context, registry credential. Hindari arbitrary remote Compose in MVP.

Step 3 **Runtime:** container internal port, health endpoint optional, env values (secret values write-only), named volume presets, network profile public-web/internal-only. Tidak ada host port by default.

Step 4 **Publishing:** toggle **Internal only (default)** atau **Expose via Cloudflare**. Jika expose: subdomain satu-level (mis. `foo.imuii.id`), destination tunnel existing, Access: *Private* (default) atau *Public* dengan explicit risk confirmation. Tampilkan konflik existing DNS + routes + Traefik label.

Step 5 **Review:** ringkasan Compose YAML yang dihasilkan (redacted), planned Cloudflare changes, security validation, disk footprint, risiko downtime; tombol `Deploy` (dengan nonce/idempotency key).

Step 6 **Progress:** langkah job realtime (polling MVP), actionable errors, links to logs, rollback status, tombol Open App hanya setelah health check + DNS/route verification.

**UX errors wajib:** invalid image/port, duplicate project name, unavailable host, conflicting DNS CNAME/A, existing VPS wildcard override check, Cloudflare permission missing, tunnel degraded, missing Access policy, image pull failure, unhealthy container, request timed out.

### 6.3 Product wording

Berikan label "Running" hanya jika Docker running; label "Healthy" hanya jika health condition/check terbukti; label "Public URL ready" hanya jika route + DNS + HTTPS terbukti. Bedakan "Unknown", "Pending", "Unmanaged", "Degraded", "Failed".

## 7. Functional requirements & acceptance

### EPIC HCC-01 — Host & Overview (P0)

| ID | Requirement | Acceptance |
|---|---|---|
| F-001 | Read Docker Engine info, version, host disk/CPU/memory availability | Dashboard shows state with last_checked_at, error on disconnected |
| F-002 | Read docker containers scoped to host | Managed/unmanaged labeled; never imply all belong to HCC |
| F-003 | Show Traefik & cloudflared status | Distinguish container running vs Cloudflare tunnel healthy |
| F-004 | Show last 20 jobs and failures | Filter by state and project; timestamps timezone-safe |
| F-005 | Refresh with retry/timeouts | Slow/failed host returns explicit unknown state, no false green |

### EPIC HCC-02 — App discovery & operations (P0)

| ID | Requirement | Acceptance |
|---|---|---|
| F-006 | Import metadata of existing Docker containers read-only | CVAT/9Router visible as Unmanaged, no mutating actions |
| F-007 | List created HCC apps with metadata | slug, image, tag/digest, host, port, hostname, status, creator, createdAt |
| F-008 | Start/stop/restart managed services | Executes selected service only, audit + state refreshed |
| F-009 | Bounded logs | tail/limit, streaming later; redact secrets/key-looking strings |
| F-010 | Inspect service | Safe subset config, network, mount metadata; never reveal secret contents |
| F-011 | Delete managed app (preserve data default) | Graceful stop, remove owned route/DNS/Access when safe; leave volumes intact and report leftovers |

### EPIC HCC-03 — Application deploy (P0)

| ID | Requirement | Acceptance |
|---|---|---|
| F-012 | Template/image deploy wizard | Nginx demo or approved HTTP image deploys without manual Compose edit |
| F-013 | Generate project Compose in controlled directory | `managed-projects/<slug>/compose.yaml` with labels & networks; never overwrite `infra/compose.yaml` |
| F-014 | Validate config before apply | `docker compose config --quiet` + custom policy validation, no API secrets in output |
| F-015 | Label only explicitly published apps | Internal-only app has no Traefik router |
| F-016 | Prevent unsafe Compose options | Reject privileged, host network/pid, host mounts, socket mounts, host port publishing by default |
| F-017 | Track async job & artifact version | Job steps persisted, rerunning same request does not duplicate project |

### EPIC HCC-04 — Domain / Tunnel automation (P0 for one hostname)

| ID | Requirement | Acceptance |
|---|---|---|
| F-018 | Read Cloudflare zone/DNS & tunnel config | Shows live vs planned resources; inaccessible API = no mutation |
| F-019 | Add exact hostname CNAME via API | `foo.imuii.id` only; proxied target current tunnel UUID; no wildcard change |
| F-020 | Add corresponding published application ingress | Rule `foo.imuii.id -> http://traefik:80`; preserve all existing ingress and final catch-all 404 |
| F-021 | Conflict detection | If same hostname exists at DNS/tunnel/Traefik/unmanaged, BLOCK with explanation |
| F-022 | Owned-resource reconciliation & removal | Delete only HCC-owned resources after ID/target match, otherwise stop for review |
| F-023 | Verify route + DNS + HTTP | Only mark Published after successful post-deploy check |

**Cloudflare remote tunnel config is a full replacement PUT, not append:** GET current configuration, preserve route order and all `originRequest` fields + catch-all, merge exactly one hostname, write, read back and compare. On concurrent edits: reread and compare before write, serialize HCC Cloudflare writes per tunnel, refuse if changed unexpectedly; document provider no generalized atomic compare-and-swap guarantee. Never overwrite manual route `hello.imuii.id` or `waha-v2.imuii.id`.

**Real constraint:** Current wildcard `A *` points to the VPS. **MVP only creates explicit hostnames; NEVER flip, delete, or reinterpret wildcard.** Exact DNS wins over wildcard, but do not assume unknown VPS hostnames can be adopted.

### EPIC HCC-05 — Access, secrets, and security (P0/P1)

| ID | Requirement | Acceptance |
|---|---|---|
| F-024 | HCC dashboard itself behind Cloudflare Access + app auth | Unknown user denied; no anonymous deployment endpoint |
| F-025 | "Private" requires Access application + Allow policy linked to configured admins | Incomplete Access provisioning blocks DNS publication |
| F-026 | Public publishing requires explicit confirmation | Confirmation records actor, hostname, risk, time |
| F-027 | Cloudflare API token integration | Separate least-privilege API token, never reuse `TUNNEL_TOKEN` for management |
| F-028 | Secret write-only fields | Value never returned through GET, audit, logs, exceptions or preview |
| F-029 | Local agent request authentication & authorization | Control-plane spoofing/replay or arbitrary job JSON cannot reach Docker |

### EPIC HCC-06 — Audit, failures, recovery (P0/P1)

| ID | Requirement | Acceptance |
|---|---|---|
| F-030 | Append-only audit | who/when/what/project/status/correlationID, no sensitive payload |
| F-031 | Deployment step tracking | progress, timestamps, external IDs, masked diagnostics |
| F-032 | Safe compensating rollback | On failure restore owned resource only and preserve original config snapshot |
| F-033 | Drift detection | Differences between local DB, Docker, DNS, tunnel, Access marked; no destructive auto-sync |
| F-034 | Explicit retry | Failed steps can retry idempotently, no duplicate CNAME/Compose stacks |

### EPIC HCC-07 — Backups & multi-host (P2; architecture now)

- Backup per managed volume with restore checklist and no deletion/overwrite of active volumes; verify archive exists & is encrypted out of repo.
- Multiple hosts each run own authenticated agent. UI supports host selection and node heartbeat, per-host capacity and offline status.
- Agent on laptop B can be registered without copying credentials to Git; identical tunnel must not have two independent active hosts with unsynchronized WAHA/session state.
- Project ownership + approval/RBAC staged later; explicit feature flags off until implemented.

## 8. Technical architecture (proposed baseline)

### 8.1 Logical topology

`Browser (Cloudflare Access + HCC login) -> Next.js UI -> Go Control API -> persistent job queue/SQLite -> Windows Host Agent (outbound authenticated polling) -> Docker Engine/Compose`

Separately: `Go Control API -> Cloudflare scoped API token -> DNS/Tunnel/Access APIs`.

Existing data plane stays: `Internet -> Cloudflare DNS/Tunnel -> cloudflared Docker -> Traefik -> app container`.

**Why split agent:** controlling Docker is highly privileged. UI, Next.js, and Go Control API **must not mount the Docker socket** or pass arbitrary Docker commands. Agent runs as a separately installed, trusted process on Windows host with the minimum practical Docker permissions (caveat: Docker Engine access is effectively administrator-level). For one-host MVP, agent communicates over **authenticated outbound TLS polling** with control API; no inbound 0.0.0.0 agent endpoint. Windows service/autostart options documented for Windows Home, no SSH required.

### 8.2 Components & recommended locations

| Component | Deployment | Responsibility |
|---|---|---|
| `control-center/web` | Next.js/TypeScript/Tailwind Docker, protected route | Dashboard UI, no privileged secrets |
| `control-center/api` | Go Docker service, internal network | REST API, authz, jobs, DB, Cloudflare integration |
| `control-center/agent` | Go binary on Windows host for MVP | Poll job, validate manifest, execute allowlisted Docker/Compose ops, report status |
| `control-center/shared` | Optional contracts/schema | API DTO and manifest schema |
| SQLite (initial) | Named volume bound only to Control API | Metadata/jobs/audit; WAL, migrations, backup |
| Existing `infra/*` | Unchanged baseline | Tunnel, Traefik, socket proxy, network |

Next.js acts as UI/BFF only; privileged ops are Go API → agent. For initial local development, mock agent/Cloudflare provider to test UI without touching running services.

**Host connectivity:** agent initiates connection to control API via HTTPS service endpoint, authenticated with a distinct agent credential. Use Cloudflare Access Service Auth when Control API is exposed through Access, plus application-level signed agent identity. Never bypass Access globally. API endpoint only exposes narrowly defined authenticated agent polling/ack/result; no Docker Engine TCP port. Multiple hosts future use separate credentials and host IDs.

**Scope note:** Do not treat Traefik's existing read-only `docker-socket-proxy` as a deployment API. It intentionally has `POST=0`; the deploy agent is a distinct privileged trust boundary.

### 8.3 Deploy engine & manifest

Dashboard creates validated declarative `ApplicationSpec`, *not arbitrary shell*. Example (non-executable conceptual schema):

`apiVersion: hcc/v1; kind: Application; metadata:{slug,displayName,hostId}; image:{ref,digest?}; runtime:{port,healthPath?,envRefs[],volumeSpecs[],networkProfile}; route:{visibility,hostname?,tunnelId?,accessPolicyId?}`

Validate against allowlist, slug/hostname grammar, image registry policy, internal container port [1,65535], resources, ownership, policy. Generate deterministic Compose with project name `hcc-<slug>`, labels `hcc.managed=true`, `hcc.project-id`, `hcc.spec-hash`, Traefik router names unique/stable, and no user-supplied raw Docker arguments. Compose target stored in **separate** `managed-projects/` (gitignored runtime/generated state), not `projects/waha` or `infra/compose.yaml`.

### 8.4 Provisioning state machine

`DRAFT → VALIDATING → AWAITING_APPROVAL? → PROVISIONING_ACCESS → PULLING → STARTING → HEALTHCHECK → PUBLISHING_ROUTE → VERIFYING_PUBLIC → ACTIVE`

Failure branches: `FAILED_PRECHECK` / `FAILED_WITH_ROLLBACK` / `NEEDS_MANUAL_RECOVERY`; stop/restart/delete have their own jobs.

Order:
1. Validate admin/host/API token policy and available resources.
2. Reserve hostname (DB) & inspect live Cloudflare DNS/tunnel/Access + Docker Traefik ownership for conflicts.
3. If private, ensure Access app/policy first, test configuration; **no DNS yet**.
4. Generate Compose to temporary directory, validate, persist desired spec & deployment snapshot, have agent apply, health-check via Docker network.
5. For exposed: add route while preserving config; create exact proxied CNAME; verify both and external HTTPS. For internal: finish without public route.
6. Commit ownership IDs and ACTIVE state; emit audit.

On any error, stop creation of new public resources; compensate only HCC-created objects, preserve existing manual rules, retain diagnostic state and report manual recovery when safe rollback cannot be guaranteed. All destructive operations need separate confirmation and safe defaults.

### 8.5 Reconciliation & drift

- Desired state in DB, actual Docker from agent, actual Cloudflare from API.
- Compare every 1–5 minutes (configurable); cache most recent observation with timestamp.
- If manual edit detected (e.g. user updates route in Cloudflare dashboard), surface conflict; do **not** blindly overwrite.
- Explicit Reconcile action shows diff and requests confirmation.
- Avoid health checks relying solely on container Up; support HTTP status, configured healthcheck, app 401/302 intentionally acceptable only if explicitly expected.
- Unknown/unmanaged containers remain read-only until safe adoption workflow later.

## 9. Proposed API contract (HTTP JSON v1)

Base: `/api/v1`, auth and CSRF on browser mutations, RFC7807-like error responses, consistent request IDs; all writes require `Idempotency-Key`.

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness/readiness (no secrets; private deployment) |
| GET | `/overview` | Aggregated status |
| GET | `/hosts`, `/hosts/{id}` | Nodes + observed capacity |
| GET | `/apps?managed=true` | Managed and discovered apps |
| GET | `/apps/{id}` | Detail, desired vs observed |
| POST | `/apps/validate` | Dry run manifest, conflict & preview |
| POST | `/apps` | Submit deploy job |
| POST | `/apps/{id}/actions` | Start/stop/restart, controlled action |
| DELETE | `/apps/{id}` | Safe delete (preserve volumes default) |
| GET | `/apps/{id}/logs?tail=200` | Bounded redacted logs |
| GET | `/jobs`, `/jobs/{id}` | Job status/step events |
| POST | `/jobs/{id}/retry` | Idempotent retry with preflight |
| GET | `/routes` | Owned/external DNS/tunnel routes |
| GET | `/audit` | Sanitized immutable audit |
| GET | `/integrations/cloudflare/status` | Permissions/status, no token |
| POST | `/agent/v1/register` | Admin-issued one-time enrollment flow (P1) |
| POST | `/agent/v1/poll` | Signed agent requests for allowed jobs |
| POST | `/agent/v1/jobs/{id}/result` | Signed job step result |

REST paths are a contract proposal; Cursor may refine naming with ADR *before* implementing. Critical policy semantics and auth must not change silently.

**Error response fields:** `code`, `message`, `requestId`, `retryable`, `details` (redacted), `remediation`. Typical codes: `DNS_CONFLICT`, `TUNNEL_CONFIG_CHANGED`, `ACCESS_REQUIRED`, `HOST_OFFLINE`, `DOCKER_POLICY_DENIED`, `IMAGE_PULL_FAILED`, `HEALTHCHECK_FAILED`, `ROLLBACK_INCOMPLETE`.

## 10. Data model & persistence

| Entity | Key fields (conceptual) |
|---|---|
| User | id, email/subject, role, active, createdAt |
| HostNode | id, name, os, arch, status, lastHeartbeatAt, agentVersion, capabilities |
| App | id, slug, name, hostId, managed, ownershipTag, desiredSpecJSON, state, createdBy |
| Deployment | id, appId, version, specHash, composePath, imageRef, startedAt, endedAt, state |
| AppRoute | id, appId, fqdn, visibility, zoneId, dnsRecordId, tunnelId, ingressFingerprint, accessAppId, ownership |
| SecretReference | id, scope, name, secretStoreRef, createdAt, rotatedAt; **no plaintext in API** |
| Job | id, type, appId, hostId, idempotencyKey, state, retryCount, errorCode, timestamps |
| JobStep | jobId, sequence, name, state, startedAt, endedAt, sanitizedSummary |
| AuditEvent | id, actor, action, target, result, requestId, timestamp, redactedDiff |
| Integration | provider, account/zone/tunnel metadata, permissionCheck, lastVerifiedAt; secret reference only |

**Persistence:** Go API with SQLite + migrations; unique indexes on (hostId, app slug), globally scoped hostname, idempotency key, external owned ID; WAL/busy timeouts and serialized writes. Backup DB to encrypted destination before schema migration; ensure data path not committed. Large scale can move to PostgreSQL later; no multi-writer DB in MVP.

**Security policy:** secrets live in OS secret store/encrypted file with restricted ACL or proven secret manager; do not put Cloudflare API tokens into SQLite as plaintext, Compose env output, Docker labels, log, browser localStorage, or repo. JWT/service token values masked; reference IDs may be stored. Cloudflare **Tunnel connector token** is not equivalent to **Cloudflare management API token**.

## 11. Cloudflare API integration details

- DNS read/list and create/delete: `GET/POST/DELETE /zones/{zone_id}/dns_records` with scoped `DNS Read/Write` as required.
- Tunnel list: `GET /accounts/{account_id}/cfd_tunnel` or current supported endpoint; ensure `config_src=cloudflare` for remote management.
- Tunnel config: `GET/PUT /accounts/{account_id}/cfd_tunnel/{tunnel_id}/configurations`; PUT **replaces entire configuration**; preserve unknown fields and final `http_status:404`.
- Access apps/policies: supported Account Access API; reusable policy reference recommended. Only create/edit app owned by HCC. Existing `waha-v2`/`WAHA Admin Only` may be referenced read-only only after verification, never overwritten.
- All Cloudflare API calls must use distinct least-privilege token with explicit zone/account scopes; timeouts, backoff for 429/5xx, retries with idempotency/GET reconciliation.
- Verify DNS exact name, no collisions with an existing A/CNAME; Cloudflare may not allow CNAME and A on same name.
- Never create `*` route, edit `@`, MX/NS/TXT/DNSSEC, or transfer registrar from dashboard MVP.
- **Private access:** policy must explicitly enumerate authorized principals; no Everyone defaults. Machine clients require separate Service Auth flow, not bypassing Access.
- Show integration status when permissions insufficient instead of requesting Global API Key.

## 12. Authentication, authorization, threat model

**HCC admin UI:** Cloudflare Access in front of hostname + own session/authorization, secure HttpOnly/SameSite cookies, CSRF protection, session expiration, auth audit; for local dev explicit dev-mode restricted to loopback (never auto enable on public route).

**Privileged agent:** native Go on Windows host; no remote shell or arbitrary command input; only typed commands with approved schema and restrictive allowlist. Authenticating to control plane via per-host credential + mutual validation, short-lived nonce/time window, replay protection, job signature, bounded permissions. Agent must reject requests whose `hostId`, signature, requested image/network/volume and project policy mismatch. Authenticate and authorize *each* operation (not merely UI page access).

**Unsafe compose blocked:** reject `privileged`, `pid: host`, `network_mode: host`, `devices`, `cap_add`, direct Docker socket, host root mounts, docker build with untrusted custom Dockerfile, unrestricted `volumes` paths, host port binding, env key exfiltration via logs, sidecar injection. Operator should not be given general Docker socket; sandboxing of arbitrary image is NOT guaranteed by Docker alone.

**Cloudflare:** Access for management and sensitive apps, image supply-chain policy, separate tokens, revoke/rotate; do not expose WAHA without API auth. Never auto-enable public WAHA/9Router/CVAT based on detection.

**Operations:** every change must be attributable, support dry run, redacted logs, rate limit, per-host serialized mutation jobs; timeouts; no mutation when integration unhealthy; secure CORS, CSP, validation, constant-time token checks, password hashing if local credentials used, dependency scans.

## 13. Non-functional requirements

| Area | Requirement |
|---|---|
| Compatibility | Windows 11 Home, Docker Desktop Linux/WSL2, PowerShell 5.1 invoked via Git Bash; Go cross-platform path for future Ubuntu |
| Availability | If host/Docker offline, dashboard may show stale cached state with timestamp; no false healthy |
| Performance | Overview initial load target <3s on local network under 30 containers, after warmup |
| Reliability | restart-safe SQLite jobs, idempotent operations, serialized mutation per host and per tunnel |
| Observability | JSON structured logs with jobId/traceId, error codes, metrics; no credentials/log spam |
| Accessibility | keyboard navigable, semantic forms, readable errors, responsive desktop-first |
| Recovery | snapshots, compensating steps, safe backup of metadata, no deleting existing volumes |
| Maintainability | clear Go packages/interfaces, typed TS API client, domain logic independent of Cloudflare SDK |
| Testing | unit tests, contract tests, mocked Docker/Cloudflare, real local Docker integration, guarded public smoke test |
| Privacy | redact email lists and PII in logs/export; service credentials never persisted to localStorage |
| Cost | no assumption of paid Cloudflare add-ons; provider plan limitations surfaced before feature enable |

## 14. Test plan & acceptance criteria

### Test matrix P0 (must pass)

1. Existing `hello.imuii.id` and `waha-v2.imuii.id` remain untouched and functional across HCC install/upgrade.
2. `scripts/test-routing.ps1` still reports 200 for Hello + Status and 404 for unknown host.
3. UI shows all managed apps; foreign CVAT/9Router containers read-only with no start/stop/delete.
4. Wizard creates a new stateless demo with unique hostname `hcc-smoke.imuii.id` via exact DNS CNAME + ingress to Traefik; HTTPS opens correct app.
5. Wizard for internal-only app creates **no Cloudflare DNS/route** and no Traefik router.
6. Attempt to deploy onto already-owned hostname `api.imuii.id` or `waha-v2.imuii.id` blocked, no partial mutation.
7. Attempt to create `*.imuii.id`, apex, nameserver, MX or arbitrary root host bind rejected.
8. Private route cannot go live before Access Allow policy verified; unauthorized browser blocked.
9. Failure injecting Cloudflare 429/timeout yields clear retryable error and no duplicate route.
10. Failure during ingress PUT (simulated concurrent edit) does not discard `hello`, `waha-v2`, or catch-all.
11. Failed deploy rollback leaves previously existing volumes and manual DNS untouched; generated artifacts/partial owned resources reconciled.
12. Start/stop/restart only for HCC-owned app; audit entry and Docker state observed.
13. Job replay/retry with same idempotency key returns existing job and does not create duplicate Compose/CNAME.
14. API endpoint without auth, bad role, or bad agent signature returns denied; unknown host stays 404.
15. Secrets cannot be found in browser payload, generated public compose, log, audit, or repo git history.
16. Agent off/disconnected results in `HOST_OFFLINE` and queued/failed-safe job, not success.
17. No deployment allows public host-bound Docker Engine, Traefik dashboard, database, Redis, WAHA or 9Router ports.
18. Do not modify `A *` and existing VPS-owned DNS under any test or normal flow.

### Verification gates

- **Gate A:** design + mocks pass.
- **Gate B:** read-only dashboard and agent local pass; zero infrastructure regression.
- **Gate C:** template deploy internal-only passes with rollback.
- **Gate D:** exact-domain protected/public deploy and DNS/Tunnel Access checks pass.
- **Gate E:** operational documentation and handoff approved; only then expand feature flags.

Use a dedicated test hostname, never overwrite production apps. Integration tests against live Cloudflare must be explicit opt-in, use disposable app name and compensating cleanup.

## 15. MVP scope vs roadmap

| Phase | Included | Exit criteria |
|---|---|---|
| P0-A Foundation | Monorepo folders, Go API, Next.js shell, DB migration, auth, typed contracts, mock providers | Build/test/lint pass |
| P0-B Read-only monitoring | Host agent, Docker discovery, overview, logs, status | Shows managed vs unmanaged correctly |
| P0-C Controlled deployment | Specs/templates, policies, Compose generation, safe agent operations, jobs/rollback | Internal demo deploy/stop/delete safe |
| P0-D Cloudflare integration | Exact CNAME + ingress, Access private, health verification, drift detection | Public protected demo end-to-end |
| P1 Operational | Backups, snapshot/version history, selective adopt, release updates, resource charts, notifications | Verified restore + controlled upgrade |
| P2 Multi-host | Agent enrollment, per-host deploy, project RBAC, approval, Git repo deployments | Two hosts independently observed/deployed without split-brain |

**Priority hard rule:** Stop after P0-D unless explicit instruction expands scope. Do not implement multi-host UI with fake remote control capabilities. Only design data model/interface for future hosts.

## 16. Suggested repo organization (additive)

`control-center/apps/web/` — Next.js + Tailwind dashboard  
`control-center/services/api/` — Go Control API (auth, jobs, repository, Cloudflare provider)  
`control-center/agent/` — native Go Windows agent  
`control-center/contracts/` — OpenAPI / JSON Schema + generated types  
`control-center/infra/` — separate Compose and deployment docs for dashboard (never replace `infra/compose.yaml`)  
`control-center/tests/` — unit, contract, integration, e2e  
`managed-projects/` — generated runtime Compose (gitignored)  
`docs/19-HOMELAB-CONTROL-CENTER-PRD.md` — this PRD  
`docs/20-CURSOR-CONTROL-CENTER-IMPLEMENTATION.md` — execution brief and phased agent tasks.

Controller technology choices may be refined before code by ADR, but must preserve privileged agent separation and current infrastructure.

## 17. Risks and mitigations

| Risk | Severity | Mitigation |
|---|---|---|
| Agent exploit permits Docker root-equivalent control | Critical | Authenticated typed agent jobs, no shell input, image/network/volume allowlist, host-local privilege boundary, reviews |
| Cloudflare PUT overwrites existing routes | Critical | Read/merge/re-read/diff + backup, serialize per tunnel, preserve catch-all, conflict halt |
| User shifts wildcard VPS to Tunnel | Critical | Disallow wildcard edits entirely MVP |
| Broken Cloudflare Access exposes admin/API | Critical | Access first, fail closed, explicit check, never "Everyone" |
| Concurrent/manual config change | High | Observe+drift; no blind overwrite; clear recovery |
| SQLite corruption/disk full | High | Backup/restore, migrations, disk warning, atomic writes |
| Windows sleeping/rebooting | Medium | autostart/heartbeat indicators; no guarantee 24/7 |
| Leaking secrets into repo/UI/logs | Critical | secret refs + redaction + scanning; .gitignore + audit |
| Stateful WAHA duplicated on host B | High | cold migration only until HA designed |
| Breaking existing Docker services | Critical | no auto-adopt, owner labels & prefix, dry-run tests |

## 18. Definition of done (MVP)

**Product:** Admin can log in; inspect host & apps; deploy one supported HTTP image via wizard with safe configuration; optionally publish exact subdomain with Access; see progress/logs/errors; stop/restart/remove owned app safely.

**Technical:** Existing infra files preserved, automatic tests green, agent isolated from UI, API creds scoped, foreign containers read-only, existing VPS wildcard untouched, cloudflared/Traefik stay healthy, startup/runbook/rollback documented.

**Evidence:** Cursor Agent must provide code paths, tests run and outcomes, screenshot/browser checks if possible, known gaps, rollback commands. No claim of live success unless tested against the actual host.

## 19. Open decisions (record ADR before coding)

1. **UI auth:** Cloudflare Access JWT verification + local admin session (preferred) vs separate OIDC; document plan without silently adding login bypass.
2. **Agent transport:** outbound authenticated HTTPS polling first; exact endpoint and credential lifecycle must be specified.
3. **Secret store Windows:** DPAPI-backed local encrypted store vs established secret manager; never plaintext SQLite.
4. **Deployment image policy:** strict approved registry/image list MVP; document onboarding new image policy.
5. **Access policy management:** reference verified reusable policy vs create app-scoped Allow policy for each owned app; define non-destructive ownership.
6. **Public availability checks:** authenticated private routes may return 302/login; declare successful check criteria without weakening auth.
7. **Backup schedule:** manual first, automated P1.
8. **Control Center hostname:** choose **later** and protect before any public DNS route; avoid hardcoding `dashboard.imuii.id` prematurely.

Any ADR must preserve all safety invariants here and may not widen MVP without user approval.

## 20. References / documentation

**Internal:** `AGENTS.md`; `infra/compose.yaml`; `projects/waha/compose.yaml`; `docs/00-HANDOFF.md`; `docs/03-CLOUDFLARE.md`; `docs/04-TRAEFIK-ROUTING.md`; `docs/07-SECURITY.md`; `docs/09-PERSISTENCE-BACKUP.md`; `docs/10-MIGRATION.md`; `docs/12-VALIDATION.md`; `docs/14-DECISIONS.md`; `docs/18-QUICKSTART-LAPTOP-BARU.md`.

**Official / verified 2026-11-11:**
- Cloudflare Published Applications: https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/routing-to-tunnel/
- Cloudflare Tunnel configuration API (GET/PUT): https://developers.cloudflare.com/api/resources/zero_trust/subresources/tunnels/subresources/cloudflared/subresources/configurations/
- Cloudflare DNS record creation API: https://developers.cloudflare.com/api/resources/dns/subresources/records/methods/create/
- Cloudflare Access application API: https://developers.cloudflare.com/api/resources/zero_trust/subresources/access/subresources/applications/
- Cloudflare Access service tokens: https://developers.cloudflare.com/cloudflare-one/access-controls/service-credentials/service-tokens/
- Docker daemon protection: https://docs.docker.com/engine/security/protect-access/
- Traefik Docker provider: https://doc.traefik.io/traefik/reference/install-configuration/providers/docker/

> **Next action for Cursor:** Follow `docs/20-CURSOR-CONTROL-CENTER-IMPLEMENTATION.md`. First audit and write ADR, then implement gated P0 phases. Do not rebuild the working homelab or change wildcard DNS.
