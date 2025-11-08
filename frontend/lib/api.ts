import axios from "axios";

type ISODateString = string;

const baseURL = process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080";

export const api = axios.create({
  baseURL,
  timeout: 8000
});

export interface SystemMetrics {
  gatewayLatencyMs: number;
  ingestionThroughput: number;
  kafkaLag: number;
  piiDetectedToday: number;
  trainingJobsActive: number;
  predictionsPerMinute: number;
}

export async function fetchSystemMetrics(): Promise<SystemMetrics> {
  const { data } = await api.get<SystemMetrics>("/api/v1/metrics/overview");
  return data;
}

export interface PipelineStatus {
  id: string;
  stage: string;
  status: "healthy" | "degraded" | "failing";
  updatedAt: ISODateString;
  details: string;
}

export async function fetchPipelineStatuses(): Promise<PipelineStatus[]> {
  const { data } = await api.get<PipelineStatus[]>("/api/v1/pipelines/status");
  return data;
}

export interface TrainingJobSummary {
  id: string;
  modelType: string;
  status: string;
  createdAt: ISODateString;
  completedAt?: ISODateString;
  accuracy?: number;
  loss?: number;
}

export async function listTrainingJobs(limit = 10): Promise<TrainingJobSummary[]> {
  const { data } = await api.get<{ jobs: TrainingJobSummary[] }>("/api/v1/training/jobs", { params: { limit } });
  return data.jobs;
}

export interface PredictionLatencyPoint {
  timestamp: ISODateString;
  latencyMs: number;
}

export async function fetchPredictionLatency(): Promise<PredictionLatencyPoint[]> {
  const { data } = await api.get<PredictionLatencyPoint[]>("/api/v1/metrics/prediction-latency");
  return data;
}

export interface AlertSummary {
  critical: number;
  warning: number;
  info: number;
}

export interface AlertItem {
  id: string;
  source: string;
  format: string;
  status: string;
  error: string;
  payload: Record<string, unknown>;
  updatedAt: ISODateString;
}

export interface AlertsResponse {
  summary: AlertSummary;
  items: AlertItem[];
}

export async function fetchAlerts(): Promise<AlertsResponse> {
  const { data } = await api.get<AlertsResponse>("/api/v1/alerts");
  return data;
}

export interface CohortQueryPayload {
  id?: string;
  dsl: string;
  description?: string;
  limit?: number;
  fields?: string[];
}

export interface CohortMetadata {
  fields?: string[];
  records?: Array<Record<string, unknown>>;
  cacheHit?: boolean;
  tenant?: string;
  slices?: unknown;
  [key: string]: unknown;
}

export interface CohortResult {
  cohortId: string;
  patientIds: string[];
  count: number;
  queryTime: string | number;
  metadata?: CohortMetadata;
}

export async function runCohortQuery(payload: CohortQueryPayload): Promise<CohortResult> {
  const { data } = await api.post<CohortResult>("/api/v1/cohort/query", payload);
  return data;
}

export async function verifyCohortDSL(dsl: string): Promise<{ status: string }> {
  const { data } = await api.post<{ status: string }>("/api/v1/cohort/verify", { dsl });
  return data;
}

export async function exportCohort(payload: CohortQueryPayload): Promise<Blob> {
  const response = await api.post<Blob>("/api/v1/cohort/export", payload, { responseType: "blob" });
  return response.data;
}
