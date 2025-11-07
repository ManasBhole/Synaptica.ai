'use client';

import { MetricCard } from "../components/metric-card";
import { Sparkline } from "../components/sparkline";
import { EventTimeline } from "../components/event-timeline";
import { usePredictionLatency, usePipelineStatuses, useSystemMetrics, useTrainingJobs } from "../hooks/useSystemMetrics";
import { Suspense } from "react";

const OverviewContent = () => {
  const { data: metrics } = useSystemMetrics();
  const { data: latency } = usePredictionLatency();
  const { data: pipelines } = usePipelineStatuses();
  const { data: jobs } = useTrainingJobs();

  return (
    <div className="space-y-8">
      <section className="metric-grid">
        <MetricCard
          label="Gateway Latency"
          value={`${metrics.gatewayLatencyMs.toFixed(0)} ms`}
          change="-12% vs last hour"
          footer={<Sparkline points={latency.map((item) => item.latencyMs)} />}
        />
        <MetricCard
          label="Ingestion Throughput"
          value={`${metrics.ingestionThroughput.toLocaleString()} events/min`}
          change="+8% vs yesterday"
        />
        <MetricCard label="PII Alerts Today" value={`${metrics.piiDetectedToday}`} change="Dual review complete" accent="primary" />
        <MetricCard label="Active Training Jobs" value={`${metrics.trainingJobsActive}`} change="3 running / 8 queued" accent="emerald" />
      </section>

      <section className="grid gap-8 lg:grid-cols-3">
        <div className="glass-panel space-y-4 px-6 py-6 lg:col-span-2">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-white/50">Pipeline Status</p>
              <h2 className="text-lg font-semibold text-white">Flow Health</h2>
            </div>
            <span className="rounded-full bg-emerald-500/20 px-3 py-1 text-xs text-emerald-300">SLO • 99.3%</span>
          </div>
          <div className="divide-y divide-white/10">
            {pipelines.map((stage) => (
              <div key={stage.id} className="flex items-center justify-between py-4">
                <div>
                  <p className="text-sm font-medium text-white/80">{stage.stage}</p>
                  <p className="text-xs text-white/50">{stage.details}</p>
                </div>
                <span
                  className={`rounded-full px-3 py-1 text-xs font-medium ${
                    stage.status === "healthy"
                      ? "bg-emerald-500/10 text-emerald-300"
                      : stage.status === "degraded"
                        ? "bg-amber-500/10 text-amber-300"
                        : "bg-rose-500/10 text-rose-300"
                  }`}
                >
                  {stage.status.toUpperCase()}
                </span>
              </div>
            ))}
          </div>
        </div>
        <EventTimeline
          events={jobs.slice(0, 4).map((job) => ({
            id: job.id,
            title: `${job.modelType} • ${job.status}`,
            description: job.completedAt ? `Accuracy ${(job.accuracy ?? 0.0 * 100).toFixed(1)}%` : "Training in progress",
            timestamp: new Date(job.createdAt).toLocaleString(),
            status: job.status === "completed" ? "success" : job.status === "running" ? "info" : "warning"
          }))}
        />
      </section>
    </div>
  );
};

export default function OverviewPage() {
  return (
    <Suspense fallback={<div className="text-white/60">Loading analytics…</div>}>
      <OverviewContent />
    </Suspense>
  );
}
