"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Suspense, useState } from "react";
import { api } from "../../lib/api";
import { useTrainingJobs } from "../../hooks/useSystemMetrics";
import { MetricCard } from "../../components/metric-card";

const TrainingContent = () => {
  const queryClient = useQueryClient();
  const { data: jobs } = useTrainingJobs();
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

  return (
    <div className="grid gap-8 lg:grid-cols-3">
      <section className="lg:col-span-2 space-y-8">
        <div className="glass-panel px-8 py-6">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-white/50">Model Factory</p>
              <h2 className="text-xl font-semibold text-white">Queued & historical jobs</h2>
            </div>
            <button
              onClick={() => mutation.mutate()}
              disabled={mutation.isLoading}
              className="rounded-full bg-primary-500 px-5 py-2 text-sm font-medium text-white transition hover:bg-primary-600 disabled:cursor-not-allowed disabled:bg-white/10"
            >
              {mutation.isLoading ? "Scheduling…" : "Schedule new training"}
            </button>
          </div>
          <table className="mt-6 w-full text-sm text-white/70">
            <thead className="text-xs uppercase tracking-widest text-white/40">
              <tr>
                <th className="pb-3 text-left">Job</th>
                <th className="pb-3 text-left">Status</th>
                <th className="pb-3 text-left">Accuracy</th>
                <th className="pb-3 text-left">Loss</th>
                <th className="pb-3 text-right">Created</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {jobs.map((job) => (
                <tr key={job.id} className="hover:bg-white/5">
                  <td className="py-3 font-medium text-white">{job.modelType}</td>
                  <td className="py-3 capitalize text-white/60">{job.status}</td>
                  <td className="py-3">{job.accuracy ? `${(job.accuracy * 100).toFixed(1)}%` : "—"}</td>
                  <td className="py-3">{job.loss?.toFixed(2) ?? "—"}</td>
                  <td className="py-3 text-right text-white/50">{new Date(job.createdAt).toLocaleTimeString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
      <aside className="space-y-6">
        <MetricCard label="Average Accuracy" value="87.4%" change="Last 7 runs" />
        <div className="glass-panel space-y-4 px-6 py-6">
          <div>
            <p className="text-xs uppercase tracking-[0.3em] text-white/50">Hyperparameters</p>
            <h3 className="text-lg font-semibold text-white">NEO Risk Model</h3>
          </div>
          <label className="block text-xs uppercase tracking-widest text-white/40">
            Threshold (mg/dL)
            <input
              type="number"
              value={threshold}
              onChange={(event) => setThreshold(Number(event.target.value))}
              className="mt-2 w-full rounded-2xl border border-white/10 bg-white/10 px-4 py-2 text-sm text-white focus:border-accent-400 focus:outline-none"
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
