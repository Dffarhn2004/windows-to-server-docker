import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Homelab Control Center",
  description: "Dasbor baca-saja untuk host homelab",
  robots: { index: false, follow: false },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="id">
      <body className="bg-ink-50 font-sans text-ink-700 antialiased">{children}</body>
    </html>
  );
}
