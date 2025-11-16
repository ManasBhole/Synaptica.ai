'use client';

import { MetricCard } from "../components/metric-card";
import { Sparkline } from "../components/sparkline";
import { EventTimeline } from "../components/event-timeline";
import { AlertFeed } from "../components/alert-feed";
import {
  usePredictionLatency,
  usePipelineStatuses,
  useSystemMetrics,
  useTrainingJobs,
  useAlerts
} from "../hooks/useSystemMetrics";
import { Suspense } from "react";

const OverviewContent = () => {
  const { data: metricsData } = useSystemMetrics();
  const { data: latencyData } = usePredictionLatency();
  const { data: pipelinesData } = usePipelineStatuses();
  const { data: jobsData } = useTrainingJobs();
  const { data: alerts } = useAlerts();

  const metrics = metricsData ?? {
    gatewayLatencyMs: 0,
    ingestionThroughput: 0,
    kafkaLag: 0,
    piiDetectedToday: 0,
    trainingJobsActive: 0,
    predictionsPerMinute: 0
  };
  const latency = latencyData ?? [];
  const pipelines = pipelinesData ?? [];
  const jobs = jobsData ?? [];

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
          accent="brand"
        />
        <MetricCard label="PII Alerts Today" value={`${metrics.piiDetectedToday}`} change="Dual review complete" accent="accent" />
        <MetricCard label="Active Training Jobs" value={`${metrics.trainingJobsActive}`} change="3 running / 8 queued" accent="sunset" />
      </section>

      <section className="grid gap-8 lg:grid-cols-3">
        <div className="glass-panel space-y-4 px-6 py-6 lg:col-span-2">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-[11px] uppercase tracking-[0.32em] text-neutral-400">Pipeline Status</p>
              <h2 className="mt-2 text-xl font-semibold text-neutral-900">Flow Health</h2>
            </div>
            <span className="rounded-full bg-brand-100 px-3 py-1 text-xs font-medium text-brand-700">SLO • 99.3%</span>
          </div>
          <div className="divide-y divide-neutral-200">
            {pipelines.map((stage) => (
              <div key={stage.id} className="flex items-center justify-between py-4">
                <div>
                  <p className="text-sm font-semibold text-neutral-900">{stage.stage}</p>
                  <p className="text-xs text-neutral-500">{stage.details}</p>
                </div>
                <span
                  className={`rounded-full px-3 py-1 text-xs font-medium ${
                    stage.status === "healthy"
                      ? "bg-brand-100 text-brand-700"
                      : stage.status === "degraded"
                        ? "bg-amber-100 text-amber-700"
                        : "bg-rose-100 text-rose-700"
                  }`}
                >
                  {stage.status.toUpperCase()}
                </span>
              </div>
            ))}
          </div>
        </div>
        <div className="space-y-6">
          <EventTimeline
            events={jobs.slice(0, 4).map((job) => ({
              id: job.id,
              title: `${job.modelType} • ${job.status}`,
              description: job.completedAt ? `Accuracy ${((job.accuracy ?? 0) * 100).toFixed(1)}%` : "Training in progress",
              timestamp: new Date(job.createdAt).toLocaleString(),
              status: job.status === "completed" ? "success" : job.status === "running" ? "info" : "warning"
            }))}
          />
          {alerts && <AlertFeed alerts={alerts} />}
        </div>
      </section>
    </div>
  );
};

export default function OverviewPage() {
  return (
    <Suspense fallback={<div className="text-neutral-500">Loading analytics…</div>}>
      <OverviewContent />
    </Suspense>
  );
}
