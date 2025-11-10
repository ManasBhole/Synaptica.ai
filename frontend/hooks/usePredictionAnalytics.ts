'use client';

import { useQuery } from "@tanstack/react-query";
import { fetchPredictionMetrics, PredictionMetricsResponse } from "../lib/api";

const fallbackPredictionMetrics: PredictionMetricsResponse = {
  summary: {
    total: 12,
    windowSeconds: 600,
    p50LatencyMs: 120,
    p95LatencyMs: 210,
    averageLatencyMs: 150,
    averageConfidence: 0.82
  },
  events: Array.from({ length: 6 }).map((_, idx) => ({
    id: `demo-${idx}`,
    patientId: `patient-${(idx + 1).toString().padStart(3, "0")}`,
    modelName: "risk-score",
    latencyMs: 110 + idx * 15,
    confidence: 0.75 + idx * 0.02,
    createdAt: new Date(Date.now() - idx * 5 * 60_000).toISOString()
  }))
};

export const usePredictionAnalytics = () =>
  useQuery({
    queryKey: ["prediction-metrics"],
    queryFn: fetchPredictionMetrics,
    refetchInterval: 20_000,
    placeholderData: fallbackPredictionMetrics
  });
