"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError } from "@/lib/api";
import { Alert, Icon, btnPrimary } from "@/components/ui";

export default function LoginPage() {
  const router = useRouter();
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [hint, setHint] = useState("");
  const [busy, setBusy] = useState(false);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    setHint("");
    try {
      const res = await fetch("/api/v1/session", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ password }),
      });
      const body = await res.json();
      if (!res.ok) throw new ApiError(body);
      router.push("/overview");
    } catch (err) {
      const problem = err instanceof ApiError ? err : new ApiError({ message: "Masuk gagal" });
      setError(problem.message);
      setHint(problem.remediation);
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="grid min-h-screen bg-white lg:grid-cols-2">
      <section className="flex flex-col justify-center px-6 py-12 sm:px-12 lg:px-20">
        <div className="mx-auto w-full max-w-sm">
          <div className="mb-10 flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-600 text-white shadow-pop">
              <Icon name="shield" className="h-5 w-5" />
            </span>
            <span className="text-lg font-semibold text-ink-900">Control Center</span>
          </div>
          <h1 className="text-3xl font-semibold tracking-tight text-ink-900">Selamat datang kembali</h1>
          <p className="mt-2 text-sm text-ink-500">
            Masuk untuk memantau host dan aplikasi homelab. Kata sandi tidak disimpan di peramban.
          </p>

          <form onSubmit={onSubmit} className="mt-8 space-y-5">
            {error ? <Alert message={error} hint={hint} /> : null}
            <div>
              <label htmlFor="password" className="mb-1.5 block text-sm font-medium text-ink-700">
                Kata sandi admin
              </label>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full rounded-lg border border-ink-300 bg-white px-3.5 py-2.5 text-sm text-ink-900 shadow-card placeholder:text-ink-400 focus:border-brand-500 focus:outline-none focus:ring-4 focus:ring-brand-100"
              />
            </div>
            <button type="submit" disabled={busy} className={`${btnPrimary} w-full py-2.5`}>
              {busy ? "Memeriksa…" : "Masuk"}
            </button>
          </form>
        </div>
      </section>

      <aside className="relative hidden overflow-hidden bg-brand-700 lg:flex lg:flex-col lg:justify-center lg:px-20">
        <div className="pointer-events-none absolute -right-24 -top-24 h-96 w-96 rounded-full bg-brand-600/60" />
        <div className="pointer-events-none absolute -bottom-32 -left-16 h-96 w-96 rounded-full bg-brand-800/70" />
        <div className="relative max-w-md text-white">
          <span className="inline-flex items-center gap-2 rounded-full bg-white/10 px-3 py-1 text-xs font-medium ring-1 ring-inset ring-white/20">
            <span className="h-1.5 w-1.5 rounded-full bg-white" /> Loopback saja
          </span>
          <h2 className="mt-6 text-3xl font-semibold leading-tight">Satu pandangan untuk seluruh homelab Anda.</h2>
          <ul className="mt-8 space-y-4 text-sm text-brand-100">
            <li className="flex gap-3"><Icon name="host" className="h-5 w-5 shrink-0 text-white" />Status host, Docker, dan Traefik secara berkala.</li>
            <li className="flex gap-3"><Icon name="apps" className="h-5 w-5 shrink-0 text-white" />Daftar aplikasi beserta kepemilikan dan kesehatannya.</li>
            <li className="flex gap-3"><Icon name="shield" className="h-5 w-5 shrink-0 text-white" />Log ter-redaksi dan jejak audit, tanpa aksi berisiko.</li>
          </ul>
        </div>
      </aside>
    </main>
  );
}
