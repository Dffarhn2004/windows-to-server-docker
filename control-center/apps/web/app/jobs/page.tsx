"use client";

import { useEffect, useState } from "react";
import { Badge } from "@/components/Badge";
import { Shell } from "@/components/Shell";
import { Alert, Card, EmptyState, SectionTitle, Skeleton } from "@/components/ui";
import { ApiError, api, when } from "@/lib/api";
import type { Audit, Job } from "@/lib/types";

export default function JobsPage() {
  const [jobs, setJobs] = useState<Job[] | null>(null);
  const [events, setEvents] = useState<Audit[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([api<{ jobs: Job[] }>("/api/v1/jobs"), api<{ events: Audit[] }>("/api/v1/audit")])
      .then(([jobBody, auditBody]) => {
        setJobs(jobBody.jobs);
        setEvents(auditBody.events);
      })
      .catch((err: unknown) => {
        if (err instanceof ApiError && err.code !== "AUTH_REQUIRED") setError(err.message);
      });
  }, []);

  return (
    <Shell title="Job & audit" subtitle="Permintaan observasi dan jejak aktivitas">
      {error ? <Alert message={error} /> : null}

      <section className="mb-10">
        <SectionTitle hint={jobs ? `${jobs.length} job` : undefined}>Job observasi</SectionTitle>
        {!jobs && !error ? <Skeleton className="h-32" /> : null}
        {jobs && jobs.length === 0 ? <EmptyState title="Belum ada job" body="Permintaan log akan muncul di sini." /> : null}
        {jobs && jobs.length > 0 ? (
          <Card>
            <ul className="divide-y divide-ink-100">
              {jobs.map((job) => (
                <li key={job.id} className="flex flex-wrap items-center gap-x-4 gap-y-1 px-5 py-3.5 text-sm hover:bg-ink-50">
                  <Badge value={job.state} />
                  <span className="font-medium text-ink-900">{job.type}</span>
                  <span className="text-ink-500">{job.summary || job.errorCode || "tanpa ringkasan"}</span>
                  <span className="ml-auto text-xs text-ink-500">{when(job.createdAt)}</span>
                </li>
              ))}
            </ul>
          </Card>
        ) : null}
      </section>

      <section>
        <SectionTitle hint={events ? `${events.length} peristiwa` : undefined}>Audit</SectionTitle>
        {!events && !error ? <Skeleton className="h-32" /> : null}
        {events && events.length === 0 ? <EmptyState title="Belum ada peristiwa" /> : null}
        {events && events.length > 0 ? (
          <Card>
            <ol className="divide-y divide-ink-100">
              {events.map((event) => (
                <li key={event.id} className="grid gap-1 px-5 py-3.5 text-sm hover:bg-ink-50 md:grid-cols-[180px_1fr_auto] md:items-center md:gap-4">
                  <span className="text-xs text-ink-500">{when(event.createdAt)}</span>
                  <span className="min-w-0 break-words text-ink-700">
                    <span className="font-medium text-ink-900">{event.actor}</span> · {event.action} · {event.target || "—"}
                  </span>
                  <span className="w-fit rounded-md bg-ink-100 px-2 py-0.5 text-xs font-medium text-ink-600">{event.result}</span>
                </li>
              ))}
            </ol>
          </Card>
        ) : null}
      </section>
    </Shell>
  );
}
