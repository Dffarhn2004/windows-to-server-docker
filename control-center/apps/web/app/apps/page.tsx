"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Badge } from "@/components/Badge";
import { Shell } from "@/components/Shell";
import { Alert, Card, EmptyState, Icon, Skeleton, btnPrimary, td, th } from "@/components/ui";
import { ApiError, api, when } from "@/lib/api";
import type { AppRow } from "@/lib/types";

export default function AppsPage() {
  const [apps, setApps] = useState<AppRow[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ apps: AppRow[] }>("/api/v1/apps")
      .then((body) => setApps(body.apps))
      .catch((err: unknown) => {
        if (err instanceof ApiError && err.code !== "AUTH_REQUIRED") setError(err.message);
      });
  }, []);

  return (
    <Shell
      title="Aplikasi"
      subtitle="Container yang terdeteksi di host, dikelola maupun tidak"
      action={<Link href="/deploy" className={btnPrimary}>Tambah aplikasi</Link>}
    >
      {error ? <Alert message={error} /> : null}
      {!apps && !error ? <Skeleton className="h-64" /> : null}
      {apps && apps.length === 0 ? (
        <EmptyState title="Belum ada aplikasi" body="Aplikasi muncul setelah agent melaporkan container dari host." />
      ) : null}
      {apps && apps.length > 0 ? (
        <Card className="overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <caption className="sr-only">Daftar aplikasi</caption>
              <thead className="border-b border-ink-200 bg-ink-50">
                <tr>
                  <th className={th}>Aplikasi</th>
                  <th className={th}>State</th>
                  <th className={th}>Health</th>
                  <th className={th}>Kepemilikan</th>
                  <th className={th}>Hostname</th>
                  <th className={th}>Diperiksa</th>
                  <th className={th}><span className="sr-only">Detail</span></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-100">
                {apps.map((app) => (
                  <tr key={app.id} className="hover:bg-ink-50">
                    <td className={td}>
                      <p className="font-medium text-ink-900">{app.name}</p>
                      <p className="max-w-[240px] truncate font-mono text-xs text-ink-500" title={app.image}>{app.image}</p>
                    </td>
                    <td className={td}><Badge value={app.stateLabel} /></td>
                    <td className={td}><Badge value={app.healthLabel} /></td>
                    <td className={td}>{app.ownership}</td>
                    <td className={td}>{app.hostname || <span className="text-ink-400">—</span>}</td>
                    <td className={`${td} whitespace-nowrap`}>{when(app.observedAt)}</td>
                    <td className={`${td} text-right`}>
                      <Link
                        href={`/apps/${app.id}`}
                        aria-label={`Detail ${app.name}`}
                        className="inline-flex rounded-lg p-1.5 text-ink-400 hover:bg-ink-100 hover:text-brand-600 focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-600"
                      >
                        <Icon name="arrow" className="h-5 w-5" />
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      ) : null}
    </Shell>
  );
}
