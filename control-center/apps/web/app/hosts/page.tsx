"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/Badge";
import { Shell } from "@/components/Shell";
import { Alert, Card, EmptyState, Skeleton, td, th } from "@/components/ui";
import { ApiError, api, bytes, when } from "@/lib/api";
import type { Host } from "@/lib/types";

export default function HostsPage() {
  const [hosts, setHosts] = useState<Host[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ hosts: Host[] }>("/api/v1/hosts")
      .then((body) => setHosts(body.hosts))
      .catch((err: unknown) => {
        if (err instanceof ApiError && err.code !== "AUTH_REQUIRED") setError(err.message);
      });
  }, []);

  return (
    <Shell title="Host" subtitle="Mesin yang menjalankan agent dan Docker">
      {error ? <Alert message={error} /> : null}
      {!hosts && !error ? <Skeleton className="h-48" /> : null}
      {hosts && hosts.length === 0 ? <EmptyState title="Belum ada host yang melapor" body="Host muncul setelah agent mengirim heartbeat pertama." /> : null}
      {hosts && hosts.length > 0 ? (
        <Card className="overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <caption className="sr-only">Daftar host</caption>
              <thead className="border-b border-ink-200 bg-ink-50">
                <tr>
                  <th className={th}>Nama</th>
                  <th className={th}>Status</th>
                  <th className={th}>OS</th>
                  <th className={th}>Docker</th>
                  <th className={th}>Memori</th>
                  <th className={th}>Disk</th>
                  <th className={th}>Heartbeat</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-100">
                {hosts.map((host) => (
                  <tr key={host.id} className="hover:bg-ink-50">
                    <td className={`${td} font-medium text-ink-900`}>{host.name}</td>
                    <td className={td}><Badge value={host.status} /></td>
                    <td className={td}>{host.os}/{host.arch}</td>
                    <td className={td}>{host.dockerOk ? host.dockerVersion : host.dockerError || "Offline"}</td>
                    <td className={td}><Usage used={host.memUsedBytes} total={host.memTotalBytes} /></td>
                    <td className={td}><Usage used={host.diskUsedBytes} total={host.diskTotalBytes} /></td>
                    <td className={`${td} whitespace-nowrap`}>{when(host.lastCheckedAt)}</td>
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

function Usage({ used, total }: { used: number; total: number }) {
  const pct = total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0;
  return (
    <div className="min-w-[140px]">
      <p className="text-xs">{bytes(used)} / {bytes(total)}</p>
      <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-ink-100" role="img" aria-label={`${pct}% terpakai`}>
        <div className={`h-full rounded-full ${pct >= 90 ? "bg-alert-500" : "bg-brand-500"}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}
