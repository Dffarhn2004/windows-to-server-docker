"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/Badge";
import { Shell } from "@/components/Shell";
import { Alert, Card, EmptyState, LoadingGrid, SectionTitle, td, th } from "@/components/ui";
import { api, when, ApiError } from "@/lib/api";
import type { Overview, Routes } from "@/lib/types";

export default function OverviewPage() {
  const [data, setData] = useState<Overview | null>(null);
  const [routes, setRoutes] = useState<Routes | null>(null);
  const [error, setError] = useState("");
  const [hint, setHint] = useState("");

  useEffect(() => {
    let stop = false;
    async function load() {
      try {
        const [overview, routeData] = await Promise.all([
          api<Overview>("/api/v1/overview"),
          api<Routes>("/api/v1/routes"),
        ]);
        if (!stop) {
          setData(overview);
          setRoutes(routeData);
          setError("");
        }
      } catch (err) {
        if (!stop && err instanceof ApiError && err.code !== "AUTH_REQUIRED") {
          setError(err.message);
          setHint(err.remediation);
        }
      }
    }
    load();
    const timer = setInterval(load, 15000);
    return () => {
      stop = true;
      clearInterval(timer);
    };
  }, []);

  return (
    <Shell title="Ringkasan" subtitle={data ? `Diperbarui ${when(data.lastCheckedAt)} · otomatis tiap 15 detik` : "Kondisi host dan layanan inti"}>
      {error ? <Alert message={error} hint={hint} /> : null}
      {!data && !error ? <LoadingGrid /> : null}
      {data && !data.host ? (
        <EmptyState title="Belum ada observasi" body="Jalankan agent di host Windows. Status tetap Unknown, bukan sehat." />
      ) : null}

      {data?.host ? (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <Stat title="Host" value={data.host.status} extra={`${data.host.name} · CPU ${data.host.cpuPercent.toFixed(0)}%`} />
          <Stat title="Docker" value={data.docker.status} extra={data.docker.version || data.docker.error || "versi belum ada"} />
          <Stat title="Traefik" value={data.traefik.status} extra="container saja" />
          <Stat title="cloudflared" value={data.cloudflared.container} extra={`control plane ${data.cloudflared.controlPlane}`} />
        </div>
      ) : null}

      {data ? (
        <section className="mt-6 grid gap-4 md:grid-cols-3">
          <Card className="p-5">
            <p className="text-sm font-medium text-ink-500">Aplikasi</p>
            <div className="mt-3 flex items-baseline gap-6">
              <div>
                <p className="text-3xl font-semibold text-ink-900">{data.counts.managed}</p>
                <p className="text-xs text-ink-500">Dikelola</p>
              </div>
              <div>
                <p className="text-3xl font-semibold text-ink-400">{data.counts.unmanaged}</p>
                <p className="text-xs text-ink-500">Luar kendali</p>
              </div>
            </div>
          </Card>
          <Card className="p-5">
            <p className="text-sm font-medium text-ink-500">Socket proxy</p>
            <p className="mt-3"><Badge value={data.socketProxy.status} /></p>
            <p className="mt-3 text-xs text-ink-500">Bukan jalur deploy.</p>
          </Card>
          <Card className="p-5">
            <p className="text-sm font-medium text-ink-500">Pemeriksaan terakhir</p>
            <p className="mt-3 text-base font-semibold text-ink-900">{when(data.lastCheckedAt)}</p>
            <p className="mt-2 text-xs text-ink-500">Waktu ditampilkan di Asia/Jakarta.</p>
          </Card>
        </section>
      ) : null}

      {routes ? (
        <section className="mt-10">
          <SectionTitle hint={routes.configured ? `${routes.routes.length} record` : undefined}>DNS dan tunnel</SectionTitle>
          {!routes.configured ? (
            <EmptyState title="Integrasi Cloudflare belum dikonfigurasi" body={routes.remediation} />
          ) : (
            <Card className="overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-left">
                  <caption className="sr-only">Record DNS yang terbaca</caption>
                  <thead className="border-b border-ink-200 bg-ink-50">
                    <tr>
                      <th className={th}>Nama</th>
                      <th className={th}>Tipe</th>
                      <th className={th}>Kepemilikan</th>
                      <th className={th}>Dilindungi</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-ink-100">
                    {routes.routes.map((route) => (
                      <tr key={`${route.type}-${route.name}`} className="hover:bg-ink-50">
                        <td className={`${td} font-medium text-ink-900`}>{route.name}</td>
                        <td className={td}><span className="rounded bg-ink-100 px-1.5 py-0.5 font-mono text-xs">{route.type}</span></td>
                        <td className={td}>{route.ownership}</td>
                        <td className={td}>{route.protected ? <Badge value="Healthy" /> : <span className="text-ink-400">tidak</span>}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </Card>
          )}
        </section>
      ) : null}
    </Shell>
  );
}

function Stat({ title, value, extra }: { title: string; value: string; extra: string }) {
  return (
    <Card className="p-5">
      <p className="text-sm font-medium text-ink-500">{title}</p>
      <p className="mt-3"><Badge value={value} /></p>
      <p className="mt-3 truncate text-xs text-ink-500" title={extra}>{extra}</p>
    </Card>
  );
}
