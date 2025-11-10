"use client";

import { useQuery } from "@tanstack/react-query";
import {
  fetchPipelineStatuses,
  fetchPipelineActivity,
  fetchPredictionLatency,
  fetchSystemMetrics,
  listTrainingJobs,
  fetchAlerts,
  PipelineStatus,
  PredictionLatencyPoint,
  SystemMetrics,
  TrainingJobSummary,
  AlertsResponse,
  PipelineActivityResponse
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
    promoted: true,
    promotedAt: new Date().toISOString(),
    promotedBy: "ops@demo",
    promotionNotes: "Auto approved",
    metrics: { accuracy: 0.87, loss: 0.42 },
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

const fallbackPipelineActivity: PipelineActivityResponse = {
  summary: {
    accepted: 18,
    published: 16,
    failed: 2,
    dlq: 2,
    backlog: 4,
    throughputPerMin: 22
  },
  events: Array.from({ length: 8 }).map((_, idx) => ({
    id: `demo-${idx}`,
    source: idx % 2 === 0 ? "hospital" : "lab",
    format: idx % 3 === 0 ? "fhir" : "csv",
    status: idx % 5 === 0 ? "failed" : idx % 3 === 0 ? "accepted" : "published",
    retryCount: idx % 4 === 0 ? 1 : 0,
    createdAt: new Date(Date.now() - idx * 4 * 60_000).toISOString(),
    updatedAt: new Date(Date.now() - idx * 3 * 60_000).toISOString(),
    error: idx % 5 === 0 ? "PII detected" : undefined
  }))
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

export const usePipelineActivity = () =>
  useQuery({
    queryKey: ["pipeline-activity"],
    queryFn: fetchPipelineActivity,
    refetchInterval: 15_000,
    placeholderData: fallbackPipelineActivity
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
