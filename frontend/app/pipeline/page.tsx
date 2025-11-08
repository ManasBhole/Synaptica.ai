'use client';

import { usePipelineStatuses, usePipelineActivity } from "../../hooks/useSystemMetrics";
import { Suspense } from "react";
import { MetricCard } from "../../components/metric-card";

const PipelineContent = () => {
  const { data } = usePipelineStatuses();
  const stages = data ?? [];
  const { data: activity } = usePipelineActivity();
  const summary = activity?.summary ?? {
    accepted: 0,
    published: 0,
    failed: 0,
    dlq: 0,
    backlog: 0,
    throughputPerMin: 0
  };
  const events = activity?.events ?? [];

  return (
    <div className="space-y-8">
      <div className="grid gap-6 md:grid-cols-4">
        <MetricCard label="Throughput" value={`${summary.throughputPerMin}/min`} change="Realtime ingestion" accent="brand" />
        <MetricCard label="Accepted" value={summary.accepted.toLocaleString()} change="Last hour" accent="accent" />
        <MetricCard label="Published" value={summary.published.toLocaleString()} change="Last hour" accent="sunset" />
        <MetricCard label="DLQ" value={summary.dlq.toLocaleString()} change={`Backlog ${summary.backlog}`} />
      </div>
      <div className="glass-panel px-8 py-6">
        <h2 className="text-xl font-semibold text-white">Live Topology</h2>
        <p className="mt-2 max-w-2xl text-sm text-white/60">
          End-to-end observability of the Synaptica data mesh. Each stage streams real-time metrics, lag, and data privacy
          posture to help SREs detect drift before downstream AI workloads feel it.
        </p>
        <div className="mt-6 grid gap-4 md:grid-cols-2">
          {stages.map((stage) => (
            <div
              key={stage.id}
              className="rounded-3xl border border-white/5 bg-surface-raised/70 px-6 py-5 shadow-[rgba(15,23,42,0.35)_0px_15px_35px_-25px]"
            >
              <p className="text-[11px] uppercase tracking-[0.3em] text-white/45">{stage.stage}</p>
              <p className="mt-3 text-sm text-white/75">{stage.details}</p>
              <div className="mt-4 flex items-center justify-between text-xs text-white/50">
                <span>Updated {new Date(stage.updatedAt).toLocaleTimeString()}</span>
                <span
                  className={`rounded-full px-3 py-1 text-xs font-semibold ${
                    stage.status === "healthy"
                      ? "bg-brand-500/15 text-brand-200"
                      : stage.status === "degraded"
                        ? "bg-amber-500/15 text-amber-200"
                        : "bg-rose-500/15 text-rose-200"
                  }`}
                >
                  {stage.status}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
      <div className="glass-panel px-8 py-6">
        <h3 className="text-lg font-semibold text-white">Recent Ingestion Activity</h3>
        <p className="mt-2 text-sm text-white/60">
          Live feed of upstream submissions with PHI screening outcomes, retry counts, and downstream publish status.
        </p>
        <div className="mt-6 overflow-x-auto">
          <table className="min-w-full divide-y divide-white/10 text-left text-sm text-white/70">
            <thead>
              <tr>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Source</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Format</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Status</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Retries</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Updated</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/10">
              {events.map((event) => (
                <tr key={event.id} className="hover:bg-white/5">
                  <td className="px-4 py-3 font-medium text-white">{event.source}</td>
                  <td className="px-4 py-3 text-white/60">{event.format.toUpperCase()}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`rounded-full px-3 py-1 text-xs font-semibold ${
                        event.status === "published"
                          ? "bg-brand-500/15 text-brand-200"
                          : event.status === "accepted"
                            ? "bg-amber-500/15 text-amber-200"
                            : "bg-rose-500/15 text-rose-200"
                      }`}
                    >
                      {event.status.toUpperCase()}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-white/60">{event.retryCount}</td>
                  <td className="px-4 py-3 text-white/60">{new Date(event.updatedAt).toLocaleTimeString()}</td>
                  <td className="px-4 py-3 text-xs text-white/50">{event.error ?? "—"}</td>
                </tr>
              ))}
              {events.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-sm text-white/40">
                    No ingestion activity yet. Submit upstream data to populate this feed.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default function PipelinePage() {
  return (
    <Suspense fallback={<div className="text-white/60">Loading pipelines…</div>}>
      <PipelineContent />
    </Suspense>
  );
}
