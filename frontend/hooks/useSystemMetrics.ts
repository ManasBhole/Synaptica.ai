"use client";

import { useQuery } from "@tanstack/react-query";
import { fetchPipelineStatuses, fetchPredictionLatency, fetchSystemMetrics, listTrainingJobs } from "../lib/api";

const fallbackMetrics = {
  gatewayLatencyMs: 124,
  ingestionThroughput: 1850,
  kafkaLag: 3,
  piiDetectedToday: 2,
  trainingJobsActive: 3,
  predictionsPerMinute: 492
};

export const useSystemMetrics = () =>
  useQuery({
    queryKey: ["system-metrics"],
    queryFn: fetchSystemMetrics,
    staleTime: 10_000,
    retry: 1,
    placeholderData: fallbackMetrics
  });

export const usePipelineStatuses = () =>
  useQuery({
    queryKey: ["pipeline-status"],
    queryFn: fetchPipelineStatuses,
    staleTime: 15_000,
    retry: 1,
    placeholderData: [
      {
        id: "ingestion",
        stage: "API Gateway ➝ Ingestion",
        status: "healthy" as const,
        updatedAt: new Date().toISOString(),
        details: "2.3k events/min"
      },
      {
        id: "privacy",
        stage: "DLP ➝ De-ID",
        status: "healthy" as const,
        updatedAt: new Date().toISOString(),
        details: "< 250ms median"
      },
      {
        id: "normalizer",
        stage: "Normalizer ➝ Linkage",
        status: "degraded" as const,
        updatedAt: new Date().toISOString(),
        details: "Kafka catch-up (lag 3)"
      }
    ]
  });

export const useTrainingJobs = () =>
  useQuery({
    queryKey: ["training-jobs"],
    queryFn: () => listTrainingJobs(8),
    refetchInterval: 10_000,
    placeholderData: [
      {
        id: "demo-risk",
        modelType: "risk-score",
        status: "completed",
        createdAt: new Date(Date.now() - 3600_000).toISOString(),
        completedAt: new Date().toISOString(),
        accuracy: 0.87,
        loss: 0.42
      }
    ]
  });

export const usePredictionLatency = () =>
  useQuery({
    queryKey: ["prediction-latency"],
    queryFn: fetchPredictionLatency,
    staleTime: 20_000,
    placeholderData: Array.from({ length: 12 }, (_, idx) => ({
      timestamp: new Date(Date.now() - (11 - idx) * 5 * 60_000).toISOString(),
      latencyMs: 150 + Math.sin(idx / 2) * 20
    }))
  });
