'use client';

import { usePipelineStatuses } from "../../hooks/useSystemMetrics";
import { Suspense } from "react";

const PipelineContent = () => {
  const { data } = usePipelineStatuses();

  return (
    <div className="space-y-8">
      <div className="glass-panel px-8 py-6">
        <h2 className="text-xl font-semibold text-white">Live Topology</h2>
        <p className="mt-2 max-w-2xl text-sm text-white/60">
          End-to-end observability of the Synaptica data mesh. Each stage streams real-time metrics, lag, and data privacy
          posture to help SREs detect drift before downstream AI workloads feel it.
        </p>
        <div className="mt-6 grid gap-4 md:grid-cols-2">
          {data.map((stage) => (
            <div key={stage.id} className="rounded-3xl border border-white/5 bg-white/5 px-6 py-5">
              <p className="text-xs uppercase tracking-[0.3em] text-white/40">{stage.stage}</p>
              <p className="mt-3 text-sm text-white/70">{stage.details}</p>
              <div className="mt-4 flex items-center justify-between text-xs text-white/50">
                <span>Updated {new Date(stage.updatedAt).toLocaleTimeString()}</span>
                <span
                  className={`rounded-full px-3 py-1 text-xs font-semibold ${
                    stage.status === "healthy"
                      ? "bg-emerald-500/10 text-emerald-300"
                      : stage.status === "degraded"
                        ? "bg-amber-500/10 text-amber-300"
                        : "bg-rose-500/10 text-rose-300"
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
              className={`h-16 rounded-2xl border border-white/5 bg-gradient-to-br ${idx % 5 === 0 ? "from-rose-500/30" : "from-primary-500/20"} to-slate-900/20 backdrop-blur`}
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
