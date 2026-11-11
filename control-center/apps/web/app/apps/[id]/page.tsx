"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Badge } from "@/components/Badge";
import { Shell } from "@/components/Shell";
import { Alert, Card, Skeleton, SectionTitle, btnGhost, btnPrimary } from "@/components/ui";
import { ApiError, api, csrfToken, idempotencyKey, when } from "@/lib/api";
import type { AppRow, Logs } from "@/lib/types";

export default function AppDetailPage() {
  const params = useParams<{ id: string }>();
  const [app, setApp] = useState<AppRow | null>(null);
  const [logs, setLogs] = useState<Logs | null>(null);
  const [error, setError] = useState("");
  const [hint, setHint] = useState("");
  const [loadingLogs, setLoadingLogs] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [acting, setActing] = useState(false);

  useEffect(() => {
    api<AppRow>(`/api/v1/apps/${params.id}`)
      .then(setApp)
      .catch((err: unknown) => {
        if (err instanceof ApiError && err.code !== "AUTH_REQUIRED") setError(err.message);
      });
  }, [params.id]);

  async function runAction(action: string, confirm = false) {
    setError("");
    setHint("");
    setActing(true);
    try {
      const path = action === "delete" ? `/api/v1/apps/${params.id}` : `/api/v1/apps/${params.id}/actions`;
      const created = await api<{ jobId: string }>(path, {
        method: action === "delete" ? "DELETE" : "POST",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": csrfToken(),
          "Idempotency-Key": idempotencyKey(),
        },
        body: JSON.stringify(action === "delete" ? { confirm } : { action }),
      });
      for (let i = 0; i < 20; i++) {
        const job = await api<{ state: string }>(`/api/v1/jobs/${created.jobId}`);
        if (job.state === "succeeded" || job.state === "failed") break;
        await new Promise((resolve) => setTimeout(resolve, 1500));
      }
      const fresh = await api<AppRow>(`/api/v1/apps/${params.id}`);
      setApp(fresh);
    } catch (err) {
      if (err instanceof ApiError && err.code === "NOT_FOUND" && action === "delete") {
        window.location.assign("/apps");
        return;
      }
      if (err instanceof ApiError) {
        setError(err.message);
        setHint(err.remediation);
      }
    } finally {
      setActing(false);
      setConfirmDelete(false);
    }
  }

  async function loadLogs() {
    setError("");
    setHint("");
    setLoadingLogs(true);
    try {
      const first = await api<Logs>(`/api/v1/apps/${params.id}/logs?tail=200`);
      if (!first.pending) {
        setLogs(first);
        return;
      }
      for (let i = 0; i < 8; i++) {
        await new Promise((resolve) => setTimeout(resolve, 1500));
        const next = await api<Logs>(`/api/v1/apps/${params.id}/logs?tail=200`);
        if (!next.pending || next.text) {
          setLogs(next);
          return;
        }
      }
      setLogs(first);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
        setHint(err.remediation);
      }
    } finally {
      setLoadingLogs(false);
    }
  }

  return (
    <Shell
      title={app?.name || "Aplikasi"}
      subtitle={app ? app.image : undefined}
      action={
        <Link href="/apps" className="text-sm font-medium text-brand-600 hover:text-brand-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-600">
          ← Semua aplikasi
        </Link>
      }
    >
      {error ? <Alert message={error} hint={hint} /> : null}
      {!app && !error ? <Skeleton className="h-56" /> : null}
      {app ? (
        <div className="space-y-6">
          <p className="rounded-lg bg-brand-50 px-4 py-3 text-sm text-brand-700">
            {app.managed
              ? "Aplikasi ini milik Control Center dan hanya internal. Tidak ada hostname publik."
              : "Spec yang diinginkan belum ada. Yang tampil adalah keadaan Docker terakhir."}
          </p>

          <Card className="p-6">
            <dl className="grid gap-x-8 gap-y-5 sm:grid-cols-2 lg:grid-cols-4">
              <Item label="State" value={app.stateLabel} badge />
              <Item label="Health" value={app.healthLabel} badge />
              <Item label="Kepemilikan" value={app.ownership} />
              <Item label="URL publik siap" value={app.publicUrlReady ? "ya" : "tidak"} />
              <Item label="Hostname" value={app.hostname || "tidak ada"} />
              <Item label="Port layanan" value={String(app.servicePort || "—")} />
              <Item label="Network" value={(app.networks || []).join(", ") || "tidak ada"} />
              <Item label="Diperiksa" value={when(app.observedAt)} />
            </dl>
          </Card>

          {app.actions.length === 0 ? (
            <p className="text-sm text-ink-500">Tidak ada aksi mutasi untuk aplikasi ini.</p>
          ) : (
            <div className="flex flex-wrap items-center gap-2">
              {app.actions.filter((name) => name !== "delete").map((name) => (
                <button key={name} type="button" disabled={acting} onClick={() => runAction(name)} className={btnGhost}>
                  {name === "start" ? "Mulai" : name === "stop" ? "Hentikan" : "Mulai ulang"}
                </button>
              ))}
              {app.actions.includes("delete") && !confirmDelete ? (
                <button type="button" disabled={acting} onClick={() => setConfirmDelete(true)} className={btnGhost}>Hapus</button>
              ) : null}
              {confirmDelete ? (
                <>
                  <span className="text-sm text-ink-600">Hapus proyek tanpa menghapus volume?</span>
                  <button type="button" disabled={acting} onClick={() => runAction("delete", true)} className={btnPrimary}>Ya, hapus</button>
                  <button type="button" disabled={acting} onClick={() => setConfirmDelete(false)} className={btnGhost}>Batal</button>
                </>
              ) : null}
            </div>
          )}

          <section>
            <SectionTitle hint={logs ? (logs.pending ? "Menunggu agent." : `Diamati ${when(logs.observedAt)}`) : undefined}>
              Log ter-redaksi
            </SectionTitle>
            {!logs ? (
              <button type="button" onClick={loadLogs} disabled={loadingLogs} className={btnPrimary}>
                {loadingLogs ? "Menunggu agent…" : "Muat log"}
              </button>
            ) : (
              <div className="overflow-hidden rounded-xl border border-ink-800 bg-ink-900 shadow-card">
                <div className="flex items-center justify-between border-b border-ink-800 px-4 py-2">
                  <span className="text-xs text-ink-400">tail 200 · rahasia disamarkan</span>
                  <button type="button" onClick={loadLogs} disabled={loadingLogs} className="text-xs font-medium text-brand-300 hover:text-brand-200 focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-400">
                    {loadingLogs ? "Memuat…" : "Muat ulang"}
                  </button>
                </div>
                <pre className="max-h-96 overflow-auto p-4 font-mono text-xs leading-relaxed text-ink-200">{logs.text || "Log kosong."}</pre>
              </div>
            )}
          </section>
        </div>
      ) : null}
    </Shell>
  );
}

function Item({ label, value, badge }: { label: string; value: string; badge?: boolean }) {
  return (
    <div className="min-w-0">
      <dt className="text-xs font-medium uppercase tracking-wide text-ink-500">{label}</dt>
      <dd className="mt-1.5 break-words text-sm font-medium text-ink-900">{badge ? <Badge value={value} /> : value}</dd>
    </div>
  );
}
