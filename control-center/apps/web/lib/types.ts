export type Host = {
  id: string;
  name: string;
  os: string;
  arch: string;
  status: string;
  lastCheckedAt: string | null;
  agentVersion: string;
  dockerVersion: string;
  dockerOk: boolean;
  dockerError: string;
  cpuPercent: number;
  memUsedBytes: number;
  memTotalBytes: number;
  diskUsedBytes: number;
  diskTotalBytes: number;
};

export type AppRow = {
  id: string;
  slug: string;
  name: string;
  hostId: string;
  image: string;
  state: string;
  stateLabel: string;
  health: string;
  healthLabel: string;
  managed: boolean;
  ownership: string;
  hostname: string;
  servicePort: number;
  observedAt: string;
  publicUrlReady: boolean;
  actions: string[];
  projectSlug?: string;
  networks?: string[];
  desiredSpec?: null;
};

export type Overview = {
  host: Host | null;
  docker: { status: string; version: string; error?: string; lastCheckedAt: string | null };
  traefik: { status: string; lastCheckedAt: string | null };
  cloudflared: { container: string; controlPlane: string; lastCheckedAt?: string | null };
  socketProxy: { status: string };
  counts: { managed: number; unmanaged: number };
  integration: {
    configured: boolean;
    zoneReadable: boolean;
    tunnelControlPlane: string;
    error?: string;
    remediation?: string;
  };
  lastCheckedAt: string | null;
};

export type RouteRow = {
  name: string;
  type: string;
  content: string;
  proxied: boolean;
  ownership: string;
  protected: boolean;
};

export type Routes = {
  configured: boolean;
  zoneReadable: boolean;
  tunnelControlPlane: string;
  lastCheckedAt: string;
  error?: string;
  remediation?: string;
  routes: RouteRow[];
  truncated?: boolean;
};

export type Job = {
  id: string;
  type: string;
  appId: string;
  hostId: string;
  state: string;
  errorCode: string;
  summary: string;
  createdAt: string;
  updatedAt: string;
};

export type Audit = {
  id: string;
  actor: string;
  action: string;
  target: string;
  result: string;
  requestId: string;
  createdAt: string;
};

export type Logs = {
  appId: string;
  tail: number;
  text: string;
  redacted: boolean;
  observedAt: string | null;
  pending: boolean;
  jobId: string | null;
};

export type Problem = {
  code?: string;
  message?: string;
  remediation?: string;
};
