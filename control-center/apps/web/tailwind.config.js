/** @type {import('tailwindcss').Config} */
// Palet SaaS dibatasi tepat 3 warna: brand (indigo), ink (slate netral), alert (rose).
// `theme.colors` sengaja menimpa (bukan extend) agar warna lain tidak bisa dipakai tak sengaja.
module.exports = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    colors: {
      transparent: "transparent",
      current: "currentColor",
      white: "#ffffff",
      brand: {
        50: "#eef2ff",
        100: "#e0e7ff",
        200: "#c7d2fe",
        300: "#a5b4fc",
        400: "#818cf8",
        500: "#6366f1",
        600: "#4f46e5",
        700: "#4338ca",
        800: "#3730a3",
        900: "#312e81",
      },
      ink: {
        50: "#f8fafc",
        100: "#f1f5f9",
        200: "#e2e8f0",
        300: "#cbd5e1",
        400: "#94a3b8",
        500: "#64748b",
        600: "#475569",
        700: "#334155",
        800: "#1e293b",
        900: "#0f172a",
      },
      alert: {
        50: "#fff1f2",
        100: "#ffe4e6",
        200: "#fecdd3",
        500: "#f43f5e",
        600: "#e11d48",
        700: "#be123c",
      },
    },
    extend: {
      fontFamily: {
        sans: ["Inter", "Segoe UI", "system-ui", "sans-serif"],
        mono: ["ui-monospace", "Cascadia Code", "Consolas", "monospace"],
      },
      boxShadow: {
        card: "0 1px 2px 0 rgb(15 23 42 / 0.04), 0 1px 3px 0 rgb(15 23 42 / 0.06)",
        pop: "0 10px 30px -10px rgb(79 70 229 / 0.25)",
      },
    },
  },
  plugins: [],
};
