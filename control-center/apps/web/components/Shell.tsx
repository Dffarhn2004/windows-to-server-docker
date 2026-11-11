"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { csrfToken } from "@/lib/api";
import { Icon, PageHeader, type IconName } from "@/components/ui";

const links: { href: string; label: string; icon: IconName }[] = [
  { href: "/overview", label: "Ringkasan", icon: "overview" },
  { href: "/apps", label: "Aplikasi", icon: "apps" },
  { href: "/deploy", label: "Tambah", icon: "plus" },
  { href: "/hosts", label: "Host", icon: "host" },
  { href: "/jobs", label: "Job & audit", icon: "jobs" },
];

export function Shell({
  title,
  subtitle,
  action,
  children,
}: {
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  const path = usePathname();
  const router = useRouter();

  async function logout() {
    await fetch("/api/v1/logout", {
      method: "POST",
      credentials: "include",
      headers: { "X-CSRF-Token": csrfToken() },
    });
    router.push("/login");
  }

  return (
    <div className="min-h-screen bg-ink-50 text-ink-700 lg:grid lg:grid-cols-[256px_1fr]">
      <a
        href="#isi"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-brand-600 focus:px-3 focus:py-2 focus:text-white"
      >
        Lewati ke isi
      </a>

      <aside className="hidden border-r border-ink-200 bg-white lg:sticky lg:top-0 lg:flex lg:h-screen lg:flex-col">
        <div className="flex items-center gap-3 px-6 py-6">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-brand-600 text-white shadow-pop">
            <Icon name="shield" className="h-5 w-5" />
          </span>
          <div className="leading-tight">
            <p className="text-sm font-semibold text-ink-900">Control Center</p>
            <p className="text-xs text-ink-500">Homelab</p>
          </div>
        </div>
        <nav className="flex-1 space-y-1 px-3" aria-label="Utama">
          {links.map((link) => {
            const active = path.startsWith(link.href);
            return (
              <Link
                key={link.href}
                href={link.href}
                aria-current={active ? "page" : undefined}
                className={`group flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-600 ${
                  active ? "bg-brand-50 text-brand-700" : "text-ink-600 hover:bg-ink-100 hover:text-ink-900"
                }`}
              >
                <Icon name={link.icon} className={`h-5 w-5 ${active ? "text-brand-600" : "text-ink-400 group-hover:text-ink-600"}`} />
                {link.label}
              </Link>
            );
          })}
        </nav>
        <div className="border-t border-ink-200 p-3">
          <div className="mb-2 flex items-center gap-2 rounded-lg bg-ink-50 px-3 py-2 text-xs text-ink-500">
            <Icon name="globe" className="h-4 w-4 text-brand-600" />
            Mode internal · Asia/Jakarta
          </div>
          <button
            type="button"
            onClick={logout}
            className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-ink-600 transition hover:bg-ink-100 hover:text-ink-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-600"
          >
            <Icon name="logout" className="h-5 w-5 text-ink-400" />
            Keluar
          </button>
        </div>
      </aside>

      <div className="min-w-0">
        <header className="sticky top-0 z-20 border-b border-ink-200 bg-white/90 backdrop-blur lg:hidden">
          <div className="flex items-center justify-between px-4 py-3">
            <div className="flex items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-600 text-white">
                <Icon name="shield" className="h-4 w-4" />
              </span>
              <span className="text-sm font-semibold text-ink-900">Control Center</span>
            </div>
            <button
              type="button"
              onClick={logout}
              className="rounded-lg px-2 py-1.5 text-sm text-ink-600 hover:bg-ink-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-600"
            >
              Keluar
            </button>
          </div>
          <nav className="flex gap-1 overflow-x-auto px-3 pb-2" aria-label="Utama">
            {links.map((link) => {
              const active = path.startsWith(link.href);
              return (
                <Link
                  key={link.href}
                  href={link.href}
                  aria-current={active ? "page" : undefined}
                  className={`flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium ${
                    active ? "bg-brand-50 text-brand-700" : "text-ink-600 hover:bg-ink-100"
                  }`}
                >
                  <Icon name={link.icon} className="h-4 w-4" />
                  {link.label}
                </Link>
              );
            })}
          </nav>
        </header>

        <main id="isi" className="mx-auto max-w-7xl px-4 py-8 md:px-8 md:py-10">
          <PageHeader title={title} subtitle={subtitle} action={action} />
          {children}
        </main>
      </div>
    </div>
  );
}
