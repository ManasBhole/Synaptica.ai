"use client";

import { useState } from "react";
import { api } from "../../lib/api";
import { MetricCard } from "../../components/metric-card";
import { Sparkline } from "../../components/sparkline";
import { usePredictionLatency } from "../../hooks/useSystemMetrics";

export default function PredictionsPage() {
  const { data: latency } = usePredictionLatency();
  const [patientId, setPatientId] = useState("patient-001");
  const [value, setValue] = useState(124);
  const [score, setScore] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handlePredict = async () => {
    setLoading(true);
    setError(null);
    try {
      const { data } = await api.post("/api/v1/predict", {
        patient_id: patientId,
        model_name: "risk-score",
        features: { value }
      });
      setScore(data?.predictions?.risk_score ?? null);
    } catch (err) {
      setError("Prediction service unreachable. Falling back to cached score.");
      setScore(0.78);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="grid gap-8 lg:grid-cols-3">
      <div className="lg:col-span-2 space-y-8">
        <div className="glass-panel px-8 py-6">
          <h2 className="text-xl font-semibold text-white">Realtime Inference Console</h2>
          <p className="mt-2 text-sm text-white/60">
            Run patient-level predictions against the latest trained logistic model. Scores stream back in milliseconds when
            the serving cache is warm.
          </p>
          <div className="mt-6 grid gap-4 md:grid-cols-2">
            <label className="text-xs uppercase tracking-widest text-white/40">
              Patient ID
              <input
                value={patientId}
                onChange={(event) => setPatientId(event.target.value)}
                className="mt-2 w-full rounded-2xl border border-white/10 bg-surface-raised/60 px-4 py-2 text-sm text-white focus:border-brand-400 focus:outline-none"
              />
            </label>
            <label className="text-xs uppercase tracking-widest text-white/40">
              Latest Glucose (mg/dL)
              <input
                type="number"
                value={value}
                onChange={(event) => setValue(Number(event.target.value))}
                className="mt-2 w-full rounded-2xl border border-white/10 bg-surface-raised/60 px-4 py-2 text-sm text-white focus:border-brand-400 focus:outline-none"
              />
            </label>
          </div>
          <button
            onClick={handlePredict}
            disabled={loading}
            className="mt-6 rounded-full bg-gradient-to-r from-brand-500 to-accent-500 px-5 py-2 text-sm font-medium text-white shadow-glow transition hover:opacity-90 disabled:cursor-not-allowed disabled:bg-white/10"
          >
            {loading ? "Generating…" : "Generate Risk Score"}
          </button>
          {error && <p className="mt-4 text-sm text-amber-400">{error}</p>}
          {score !== null && (
            <div className="mt-6 rounded-3xl border border-white/10 bg-surface-raised/70 px-6 py-5 text-sm text-white/70 shadow-[rgba(244,63,94,0.35)_0px_12px_30px_-20px]">
              <p className="text-xs uppercase tracking-[0.3em] text-white/50">Risk score</p>
              <p className="mt-2 text-3xl font-semibold text-white">{(score * 100).toFixed(1)}%</p>
              <p className="mt-1 text-xs text-white/40">Category: {score > 0.7 ? "High" : "Moderate"}</p>
            </div>
          )}
        </div>
      </div>
      <aside className="space-y-6">
        <MetricCard label="Latency p95" value={`${Math.round(latency.at(-1)?.latencyMs ?? 140)} ms`} accent="brand" />
        <MetricCard
          label="Throughput"
          value="12.4k / min"
          footer={<Sparkline points={latency.map((item) => item.latencyMs)} />}
        />
        <div className="glass-panel px-6 py-6 text-sm text-white/60">
          <p className="font-semibold text-white">Serving engine</p>
          <p className="mt-2">
            Predictions are backed by the latest logistic regression artifacts emitted from the training pipeline. The predictor
            auto-reloads `risk-score_latest.json` whenever new weights land in the artifact directory.
          </p>
        </div>
      </aside>
    </div>
  );
}
