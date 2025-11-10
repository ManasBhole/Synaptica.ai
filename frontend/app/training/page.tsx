"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Suspense, useState } from "react";
import { api, deprecateTrainingJob, promoteTrainingJob } from "../../lib/api";
import { useTrainingJobs } from "../../hooks/useSystemMetrics";
import { MetricCard } from "../../components/metric-card";

const TrainingContent = () => {
  const queryClient = useQueryClient();
  const { data: jobsData } = useTrainingJobs();
  const jobs = jobsData ?? [];
  const [threshold, setThreshold] = useState(115);
  const [learningRate, setLearningRate] = useState(0.05);
  const [epochs, setEpochs] = useState(300);

  const mutation = useMutation({
    mutationFn: async () => {
      await api.post("/api/v1/training/jobs", {
        model_type: "risk-score",
        config: { learning_rate: learningRate, epochs, threshold },
        filters: { cohort: "diabetes-risk" }
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["training-jobs"] });
    }
  });

  const promoteMutation = useMutation({
    mutationFn: async ({ id }: { id: string }) => {
      await promoteTrainingJob(id, {
        promoted_by: "ml-ops@console",
        deployment_target: "production",
        notes: "Promoted from control center"
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["training-jobs"] });
    }
  });

  const deprecateMutation = useMutation({
    mutationFn: async ({ id }: { id: string }) => {
      await deprecateTrainingJob(id, { notes: "Deprecated from console" });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["training-jobs"] });
    }
  });

  const promotedCount = jobs.filter((job) => job.promoted).length;
  const runningCount = jobs.filter((job) => job.status === "running").length;
  const avgAccuracy = (() => {
    const acc = jobs.map((job) => (typeof job.accuracy === "number" ? job.accuracy : undefined)).filter((val): val is number => typeof val === "number");
    if (!acc.length) return 0;
    return acc.reduce((sum, val) => sum + val, 0) / acc.length;
  })();

  return (
    <div className="grid gap-8 lg:grid-cols-3">
      <section className="lg:col-span-2 space-y-8">
        <div className="glass-panel px-8 py-6">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <p className="text-[11px] uppercase tracking-[0.32em] text-white/50">Model Factory</p>
              <h2 className="mt-2 text-xl font-semibold text-white">Queued & historical jobs</h2>
            </div>
            <button
              onClick={() => mutation.mutate()}
              disabled={mutation.status === "pending"}
              className="rounded-full bg-gradient-to-r from-brand-500 to-accent-500 px-5 py-2 text-sm font-medium text-white shadow-glow transition hover:opacity-90 disabled:cursor-not-allowed disabled:bg-white/10"
            >
              {mutation.status === "pending" ? "Scheduling…" : "Schedule new training"}
            </button>
          </div>
          <table className="mt-6 w-full text-sm text-white/70">
            <thead className="text-xs uppercase tracking-widest text-white/40">
              <tr>
                <th className="pb-3 text-left">Job</th>
                <th className="pb-3 text-left">Status</th>
                <th className="pb-3 text-left">Accuracy</th>
                <th className="pb-3 text-left">Loss</th>
                <th className="pb-3 text-left">Promotion</th>
                <th className="pb-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {jobs.map((job) => (
                <tr key={job.id} className="hover:bg-white/5">
                  <td className="py-3 font-medium text-white">
                    <div className="flex flex-col">
                      <span>{job.modelType}</span>
                      <span className="text-xs text-white/50">{new Date(job.createdAt).toLocaleString()}</span>
                    </div>
                  </td>
                  <td className="py-3 capitalize text-white/60">{job.status}</td>
                  <td className="py-3">{typeof job.accuracy === "number" ? `${(job.accuracy * 100).toFixed(1)}%` : "—"}</td>
                  <td className="py-3">{typeof job.loss === "number" ? job.loss.toFixed(2) : "—"}</td>
                  <td className="py-3">
                    {job.promoted ? (
                      <span className="rounded-full bg-brand-500/15 px-3 py-1 text-xs text-brand-200">Promoted</span>
                    ) : (
                      <span className="rounded-full bg-white/10 px-3 py-1 text-xs text-white/60">Staging</span>
                    )}
                  </td>
                  <td className="py-3 text-right">
                    <div className="flex justify-end gap-2">
                      {job.promoted ? (
                        <button
                          onClick={() => deprecateMutation.mutate({ id: job.id })}
                          disabled={deprecateMutation.status === "pending"}
                          className="rounded-full border border-white/15 px-3 py-1 text-xs text-white/70 transition hover:border-amber-400 hover:text-amber-200 disabled:cursor-not-allowed"
                        >
                          Deprecate
                        </button>
                      ) : (
                        <button
                          onClick={() => promoteMutation.mutate({ id: job.id })}
                          disabled={promoteMutation.status === "pending"}
                          className="rounded-full bg-gradient-to-r from-brand-500 to-accent-500 px-3 py-1 text-xs font-semibold text-white shadow-glow transition hover:opacity-90 disabled:cursor-not-allowed"
                        >
                          Promote
                        </button>
                      )}
                      {job.artifactPath && (
                        <a
                          href={`/api/v1/training/jobs/${job.id}/artifact`}
                          className="rounded-full border border-white/15 px-3 py-1 text-xs text-white/70 transition hover:border-brand-400 hover:text-brand-200"
                        >
                          Artifact
                        </a>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
              {jobs.length === 0 && (
                <tr>
                  <td colSpan={6} className="py-6 text-center text-sm text-white/40">
                    No jobs scheduled yet. Kick off a training run above.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>
      <aside className="space-y-6">
        <MetricCard label="Promoted Models" value={promotedCount.toString()} change="Deployment ready" accent="brand" />
        <MetricCard label="Jobs Running" value={runningCount.toString()} change="Training workers" accent="accent" />
        <MetricCard label="Avg Accuracy" value={`${(avgAccuracy * 100).toFixed(1)}%`} change="Across completed" accent="sunset" />
        <div className="glass-panel space-y-4 px-6 py-6">
          <div>
            <p className="text-[11px] uppercase tracking-[0.32em] text-white/45">Hyperparameters</p>
            <h3 className="text-lg font-semibold text-white">NEO Risk Model</h3>
          </div>
          <label className="block text-xs uppercase tracking-widest text-white/40">
            Threshold (mg/dL)
            <input
              type="number"
              value={threshold}
              onChange={(event) => setThreshold(Number(event.target.value))}
              className="mt-2 w-full rounded-2xl border border-white/10 bg-surface-raised/60 px-4 py-2 text-sm text-white focus:border-brand-400 focus:outline-none"
            />
          </label>
          <label className="block text-xs uppercase tracking-widest text-white/40">
            Learning Rate
            <input
              type="number"
              step="0.01"
              value={learningRate}
              onChange={(event) => setLearningRate(Number(event.target.value))}
              className="mt-2 w-full rounded-2xl border border-white/10 bg-white/10 px-4 py-2 text-sm text-white"
            />
          </label>
          <label className="block text-xs uppercase tracking-widest text-white/40">
            Epochs
            <input
              type="number"
              value={epochs}
              onChange={(event) => setEpochs(Number(event.target.value))}
              className="mt-2 w-full rounded-2xl border border-white/10 bg-white/10 px-4 py-2 text-sm text-white"
            />
          </label>
          <p className="text-xs text-white/40">
            Simulated training runs logistic regression over canonical lab value thresholds. Artifacts become available via the
            Serving service seconds after completion.
          </p>
        </div>
      </aside>
    </div>
  );
};

export default function TrainingPage() {
  return (
    <Suspense fallback={<div className="text-white/60">Loading training data…</div>}>
      <TrainingContent />
    </Suspense>
  );
}
