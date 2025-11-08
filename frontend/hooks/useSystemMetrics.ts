"use client";

import { useQuery } from "@tanstack/react-query";
import {
  fetchPipelineStatuses,
  fetchPredictionLatency,
  fetchSystemMetrics,
  listTrainingJobs,
  fetchAlerts,
  PipelineStatus,
  PredictionLatencyPoint,
  SystemMetrics,
  TrainingJobSummary,
  AlertsResponse
} from "../lib/api";

type MetricsFallback = SystemMetrics;

const fallbackMetrics: MetricsFallback = {
  gatewayLatencyMs: 165,
  ingestionThroughput: 1850,
  kafkaLag: 4,
  piiDetectedToday: 2,
  trainingJobsActive: 3,
  predictionsPerMinute: 512
};

const fallbackPipelines: PipelineStatus[] = [
  {
    id: "ingestion",
    stage: "API Gateway ➝ Ingestion",
    status: "healthy",
    updatedAt: new Date().toISOString(),
    details: "2.3k msgs/min • backlog 3"
  },
  {
    id: "privacy",
    stage: "DLP ➝ De-ID",
    status: "healthy",
    updatedAt: new Date().toISOString(),
    details: "PII alerts under threshold"
  },
  {
    id: "ai-normalizer",
    stage: "Normalizer ➝ Linkage ➝ Storage",
    status: "degraded",
    updatedAt: new Date().toISOString(),
    details: "Kafka catch-up in progress"
  }
];

const fallbackLatency: PredictionLatencyPoint[] = Array.from({ length: 12 }, (_, idx) => ({
  timestamp: new Date(Date.now() - (11 - idx) * 5 * 60_000).toISOString(),
  latencyMs: 140 + Math.sin(idx / 2) * 15
}));

const fallbackJobs: TrainingJobSummary[] = [
  {
    id: "demo-risk",
    modelType: "risk-score",
    status: "completed",
    createdAt: new Date(Date.now() - 45 * 60_000).toISOString(),
    completedAt: new Date().toISOString(),
    accuracy: 0.87,
    loss: 0.42
  }
];

const fallbackAlerts: AlertsResponse = {
  summary: { critical: 1, warning: 3, info: 5 },
  items: [
    {
      id: "alert-demo",
      source: "hospital",
      format: "fhir",
      status: "failed",
      error: "PII validation rejected",
      payload: { resourceType: "Patient", id: "patient-001" },
      updatedAt: new Date().toISOString()
    }
  ]
};

export const useSystemMetrics = () =>
  useQuery({
    queryKey: ["system-metrics"],
    queryFn: fetchSystemMetrics,
    staleTime: 15_000,
    placeholderData: fallbackMetrics
  });

export const usePipelineStatuses = () =>
  useQuery({
    queryKey: ["pipeline-status"],
    queryFn: fetchPipelineStatuses,
    staleTime: 15_000,
    placeholderData: fallbackPipelines
  });

export const useTrainingJobs = () =>
  useQuery({
    queryKey: ["training-jobs"],
    queryFn: () => listTrainingJobs(8),
    refetchInterval: 15_000,
    placeholderData: fallbackJobs
  });

export const usePredictionLatency = () =>
  useQuery({
    queryKey: ["prediction-latency"],
    queryFn: fetchPredictionLatency,
    staleTime: 20_000,
    placeholderData: fallbackLatency
  });

export const useAlerts = () =>
  useQuery({
    queryKey: ["alerts"],
    queryFn: fetchAlerts,
    refetchInterval: 20_000,
    placeholderData: fallbackAlerts
  });
