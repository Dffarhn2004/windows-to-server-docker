"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Shell } from "@/components/Shell";
import { Alert, Card, btnPrimary } from "@/components/ui";
import { ApiError, api, csrfToken, idempotencyKey } from "@/lib/api";
import type { Job } from "@/lib/types";

type Preview = {
  ok: boolean;
  compose: string;
  specHash: string;
  project: string;
  network: string;
  publicUrlReady: boolean;
};

export default function DeployPage() {
  const router = useRouter();
  const [slug, setSlug] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [preview, setPreview] = useState<Preview | null>(null);
  const [error, setError] = useState("");
  const [hint, setHint] = useState("");
  const [busy, setBusy] = useState(false);
  const [job, setJob] = useState<Job | null>(null);

  function specBody() {
    return JSON.stringify({ slug, displayName, template: "nginx" });
  }

  async function review() {
    setBusy(true);
    setError("");
    setHint("");
    setJob(null);
    try {
      const body = await api<Preview>("/api/v1/apps/validate", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": csrfToken(),
          "Idempotency-Key": idempotencyKey(),
        },
        body: specBody(),
      });
      setPreview(body);
    } catch (err) {
      setPreview(null);
      if (err instanceof ApiError) {
        setError(err.message);
        setHint(err.remediation);
      }
    } finally {
      setBusy(false);
    }
  }

  async function deploy() {
    setBusy(true);
    setError("");
    setHint("");
    try {
      const created = await api<{ jobId: string }>("/api/v1/apps", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": csrfToken(),
          "Idempotency-Key": idempotencyKey(),
        },
        body: specBody(),
      });
      for (let i = 0; i < 40; i++) {
        const current = await api<Job>(`/api/v1/jobs/${created.jobId}`);
        setJob(current);
        if (current.state === "succeeded" || current.state === "failed") {
          if (current.state === "succeeded") router.push("/apps");
          return;
        }
        await new Promise((resolve) => setTimeout(resolve, 2000));
      }
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
        setHint(err.remediation);
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <Shell title="Tambah aplikasi" subtitle="Nginx internal saja. Tidak ada hostname publik dan tidak ada perubahan Cloudflare.">
      {error ? <Alert message={error} hint={hint} /> : null}
      <form className="grid gap-4 md:grid-cols-2" onSubmit={(event) => event.preventDefault()}>
        <label className="text-sm font-medium text-ink-700" htmlFor="slug">
          Slug
          <input id="slug" required pattern="[a-z0-9][a-z0-9-]{0,38}[a-z0-9]" value={slug} onChange={(e) => setSlug(e.target.value)} className="mt-1 w-full rounded-lg border border-ink-200 px-3 py-2 font-normal" />
        </label>
        <label className="text-sm font-medium text-ink-700" htmlFor="name">
          Nama tampilan
          <input id="name" required maxLength={80} value={displayName} onChange={(e) => setDisplayName(e.target.value)} className="mt-1 w-full rounded-lg border border-ink-200 px-3 py-2 font-normal" />
        </label>
        <p className="md:col-span-2 text-sm text-ink-500">Template tetap nginx:alpine, port kontainer 80, network homelab_services, tanpa port host.</p>
        <button type="button" onClick={review} disabled={busy} className={btnPrimary}>Tinjau</button>
      </form>
      {preview ? (
        <Card className="mt-6 p-6">
          <h2 className="text-base font-semibold text-ink-900">Ringkasan</h2>
          <p className="mt-1 text-sm text-ink-500">Proyek {preview.project} · network {preview.network} · URL publik {preview.publicUrlReady ? "siap" : "tidak"}</p>
          <pre className="mt-4 max-h-80 overflow-auto rounded-lg bg-ink-900 p-4 font-mono text-xs text-ink-100">{preview.compose}</pre>
          <button type="button" onClick={deploy} disabled={busy} className={`${btnPrimary} mt-4`}>
            {busy ? "Menunggu agent…" : "Deploy internal"}
          </button>
          {job ? <p className="mt-3 text-sm text-ink-600">Job {job.state}: {job.summary || job.errorCode || "berjalan"}</p> : null}
        </Card>
      ) : null}
    </Shell>
  );
}
