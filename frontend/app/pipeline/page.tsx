'use client';

import { usePipelineStatuses } from "../../hooks/useSystemMetrics";
import { Suspense } from "react";

const PipelineContent = () => {
  const { data } = usePipelineStatuses();
  const stages = data ?? [];

  return (
    <div className="space-y-8">
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
        <h3 className="text-lg font-semibold text-white">Streaming Lag Heatmap</h3>
        <p className="mt-2 text-sm text-white/60">
          Kafka partitions, batch windows, and FHIR bundle delivery are tracked with 1-minute granularity. Automated
          mitigation can be triggered when lag exceeds thresholds.
        </p>
        <div className="mt-6 grid grid-cols-7 gap-3 text-center text-xs text-white/70">
          {Array.from({ length: 28 }).map((_, idx) => (
            <div
              key={idx}
              className={`h-16 rounded-2xl border border-white/5 bg-gradient-to-br ${idx % 5 === 0 ? "from-accent-500/25" : "from-brand-500/20"} to-transparent backdrop-blur`}
            >
              <div className="pt-4 text-sm font-semibold">{Math.max(0, 6 - (idx % 6))}m</div>
              <div className="text-[10px] uppercase tracking-[0.2em]">Lag</div>
            </div>
          ))}
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
