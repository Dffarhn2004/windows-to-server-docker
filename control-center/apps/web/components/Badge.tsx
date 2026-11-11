type Tone = "good" | "bad" | "warn" | "neutral";

const good = new Set(["running", "online", "healthy", "succeeded"]);
const bad = new Set(["failed", "offline"]);
const warn = new Set(["degraded", "pending"]);

const styles: Record<Tone, { pill: string; dot: string }> = {
  good: { pill: "bg-brand-50 text-brand-700 ring-brand-200", dot: "bg-brand-500" },
  bad: { pill: "bg-alert-50 text-alert-700 ring-alert-200", dot: "bg-alert-600" },
  warn: { pill: "bg-alert-50 text-alert-600 ring-alert-100", dot: "bg-alert-500 animate-pulse" },
  neutral: { pill: "bg-ink-100 text-ink-600 ring-ink-200", dot: "bg-ink-400" },
};

export function Badge({ value }: { value: string }) {
  const key = value.toLowerCase();
  const tone: Tone = good.has(key) ? "good" : bad.has(key) ? "bad" : warn.has(key) ? "warn" : "neutral";
  const s = styles[tone];
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset ${s.pill}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${s.dot}`} aria-hidden="true" />
      {value}
    </span>
  );
}
