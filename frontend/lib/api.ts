import axios from "axios";

type ISODateString = string;

const rawBase = process.env.NEXT_PUBLIC_API_BASE;
const baseURL = rawBase && rawBase.trim().length > 0 ? rawBase : "";

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

export interface PipelineActivitySummary {
  accepted: number;
  published: number;
  failed: number;
  dlq: number;
  backlog: number;
  throughputPerMin: number;
}

export interface PipelineEvent {
  id: string;
  source: string;
  format: string;
  status: string;
  error?: string;
  retryCount: number;
  lastAttempt?: ISODateString;
  createdAt: ISODateString;
  updatedAt: ISODateString;
}

export interface PipelineActivityResponse {
  summary: PipelineActivitySummary;
  events: PipelineEvent[];
}

export async function fetchPipelineActivity(): Promise<PipelineActivityResponse> {
  const { data } = await api.get<PipelineActivityResponse>("/api/v1/pipelines/activity");
  return data;
}

export interface TrainingJobSummary {
  id: string;
  modelType: string;
  status: string;
  createdAt: ISODateString;
  completedAt?: ISODateString;
  metrics?: Record<string, unknown>;
  artifactPath?: string;
  errorMessage?: string;
  promoted: boolean;
  promotedAt?: ISODateString;
  promotedBy?: string;
  promotionNotes?: string;
  deploymentTarget?: string;
  accuracy?: number;
  loss?: number;
}

export async function listTrainingJobs(limit = 10): Promise<TrainingJobSummary[]> {
  const { data } = await api.get<{ jobs: TrainingJobSummary[] }>("/api/v1/training/jobs", { params: { limit } });
  return data.jobs.map((job) => {
    const metrics = typeof job.metrics === "object" && job.metrics !== null ? (job.metrics as Record<string, unknown>) : undefined;
    const accuracy = typeof metrics?.accuracy === "number" ? (metrics?.accuracy as number) : job.accuracy;
    const loss = typeof metrics?.loss === "number" ? (metrics?.loss as number) : job.loss;
    return {
      ...job,
      accuracy,
      loss
    };
  });
}

export async function promoteTrainingJob(
  id: string,
  payload: { promoted_by?: string; notes?: string; deployment_target?: string }
): Promise<void> {
  await api.post(`/api/v1/training/jobs/${id}/promote`, payload);
}

export async function deprecateTrainingJob(id: string, payload: { notes?: string }): Promise<void> {
  await api.post(`/api/v1/training/jobs/${id}/deprecate`, payload);
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

export interface DLPReasonCount {
  reason: string;
  count: number;
}

export interface DLPIncident {
  id: string;
  source: string;
  format: string;
  status: string;
  error: string;
  updatedAt: ISODateString;
  createdAt: ISODateString;
  retryCount: number;
}

export interface DLPStatsResponse {
  todayFailed: number;
  todayAccepted: number;
  tokenVaultSize: number;
  topReasons: DLPReasonCount[];
  recentIncidents: DLPIncident[];
}

export async function fetchDLPStats(): Promise<DLPStatsResponse> {
  const { data } = await api.get<DLPStatsResponse>("/api/v1/metrics/dlp");
  return data;
}

export interface PredictionMetricsSummary {
  total: number;
  windowSeconds: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  averageLatencyMs: number;
  averageConfidence: number;
}

export interface PredictionEvent {
  id: string;
  patientId: string;
  modelName: string;
  latencyMs: number;
  confidence: number;
  createdAt: ISODateString;
}

export interface PredictionMetricsResponse {
  summary: PredictionMetricsSummary;
  events: PredictionEvent[];
}

export async function fetchPredictionMetrics(): Promise<PredictionMetricsResponse> {
  const { data } = await api.get<PredictionMetricsResponse>("/api/v1/metrics/predictions");
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

export interface CohortMaterialization {
  id: string;
  cohortId: string;
  tenantId?: string;
  dsl: string;
  fields?: string[];
  status: string;
  resultCount: number;
  errorMessage?: string;
  requestedBy?: string;
  createdAt: ISODateString;
  startedAt?: ISODateString;
  completedAt?: ISODateString;
}

export async function materializeCohort(payload: {
  cohortId?: string;
  dsl: string;
  fields?: string[];
  limit?: number;
  filters?: Record<string, unknown>;
}): Promise<CohortMaterialization> {
  const { data } = await api.post<{ job: CohortMaterialization }>("/api/v1/cohort/materialize", payload);
  return data.job;
}

export async function listCohortMaterializations(limit = 25): Promise<CohortMaterialization[]> {
  const { data } = await api.get<{ jobs: CohortMaterialization[] }>("/api/v1/cohort/materialize", {
    params: { limit }
  });
  return data.jobs;
}

export async function verifyCohortDSL(dsl: string): Promise<{ status: string }> {
  const { data } = await api.post<{ status: string }>("/api/v1/cohort/verify", { dsl });
  return data;
}

export async function exportCohort(payload: CohortQueryPayload): Promise<Blob> {
  const response = await api.post<Blob>("/api/v1/cohort/export", payload, { responseType: "blob" });
  return response.data;
}

export interface StudySite {
  id: string;
  studyId: string;
  siteCode: string;
  name: string;
  country?: string;
  principalInvestigator?: string;
  status: string;
  contact?: Record<string, unknown>;
  createdAt: ISODateString;
}

export interface StudyFormSchemaField {
  key: string;
  label: string;
  type: string;
  required?: boolean;
  options?: Array<{ label: string; value: string }>;
}

export interface StudyForm {
  id: string;
  studyId: string;
  name: string;
  slug: string;
  version: number;
  description?: string;
  schema: Record<string, unknown>;
  status: string;
  createdAt: ISODateString;
  updatedAt: ISODateString;
}

export interface VisitTemplate {
  id: string;
  studyId: string;
  name: string;
  visitOrder: number;
  windowStartDays?: number;
  windowEndDays?: number;
  required: boolean;
  forms?: string[];
  createdAt: ISODateString;
  updatedAt: ISODateString;
}

export interface Study {
  id: string;
  code: string;
  name: string;
  phase?: string;
  therapeuticArea?: string;
  status: string;
  sponsor?: string;
  protocolSummary?: Record<string, unknown>;
  startDate?: ISODateString;
  endDate?: ISODateString;
  createdAt: ISODateString;
  updatedAt: ISODateString;
  sites?: StudySite[];
  forms?: StudyForm[];
  visitTemplates?: VisitTemplate[];
  activeSubjects?: number;
  totalSubjects?: number;
}

export interface StudySubject {
  id: string;
  studyId: string;
  siteId?: string | null;
  subjectCode: string;
  status: string;
  randomizationArm?: string;
  consentedAt?: ISODateString;
  demographics?: Record<string, unknown>;
  createdAt: ISODateString;
}

export interface ConsentVersion {
  id: string;
  studyId: string;
  version: string;
  title?: string;
  summary?: string;
  documentUrl?: string;
  effectiveAt: ISODateString;
  supersededAt?: ISODateString;
  createdAt: ISODateString;
}

export interface ConsentSignature {
  id: string;
  subjectId: string;
  consentVersionId: string;
  signedAt: ISODateString;
  signerName?: string;
  method?: string;
  ipAddress?: string;
  metadata?: Record<string, unknown>;
  createdAt: ISODateString;
}

export interface AuditLogEntry {
  id: number;
  studyId: string;
  subjectId?: string | null;
  actor: string;
  role?: string;
  action: string;
  entity?: string;
  entityId?: string;
  payload?: Record<string, unknown>;
  createdAt: ISODateString;
}

export interface CreateStudyPayload {
  code: string;
  name: string;
  phase?: string;
  therapeuticArea?: string;
  sponsor?: string;
  protocolSummary?: Record<string, unknown>;
  startDate?: ISODateString;
  endDate?: ISODateString;
}

export interface CreateStudySitePayload {
  siteCode: string;
  name: string;
  country?: string;
  principalInvestigator?: string;
  contact?: Record<string, unknown>;
}

export interface CreateStudyFormPayload {
  name: string;
  slug: string;
  description?: string;
  schema: Record<string, unknown>;
  status?: string;
}

export interface CreateVisitTemplatePayload {
  name: string;
  visitOrder: number;
  windowStartDays?: number;
  windowEndDays?: number;
  required?: boolean;
  forms?: string[];
}

export interface EnrollSubjectPayload {
  siteId?: string;
  subjectCode: string;
  randomizationArm?: string;
  demographics?: Record<string, unknown>;
}

export interface CreateConsentVersionPayload {
  version: string;
  title?: string;
  summary?: string;
  documentUrl?: string;
  effectiveAt: ISODateString;
  supersededAt?: ISODateString;
}

export interface RecordConsentPayload {
  consentVersionId: string;
  signedAt: ISODateString;
  signerName?: string;
  method?: string;
  ipAddress?: string;
  metadata?: Record<string, unknown>;
}

export async function listStudies(limit = 25): Promise<Study[]> {
  const { data } = await api.get<{ items: Study[] }>("/api/v1/edc/studies", { params: { limit } });
  return data.items ?? [];
}

export async function getStudy(id: string): Promise<Study> {
  const { data } = await api.get<{ study: Study }>(`/api/v1/edc/studies/${id}`);
  return data.study;
}

export async function createStudy(payload: CreateStudyPayload): Promise<Study> {
  const { data } = await api.post<{ study: Study }>("/api/v1/edc/studies", payload);
  return data.study;
}

export async function updateStudyStatus(id: string, status: string): Promise<void> {
  await api.patch(`/api/v1/edc/studies/${id}/status`, { status });
}

export async function createStudySite(studyId: string, payload: CreateStudySitePayload): Promise<StudySite> {
  const { data } = await api.post<{ site: StudySite }>(`/api/v1/edc/studies/${studyId}/sites`, payload);
  return data.site;
}

export async function createStudyForm(studyId: string, payload: CreateStudyFormPayload): Promise<StudyForm> {
  const { data } = await api.post<{ form: StudyForm }>(`/api/v1/edc/studies/${studyId}/forms`, payload);
  return data.form;
}

export async function createVisitTemplate(studyId: string, payload: CreateVisitTemplatePayload): Promise<VisitTemplate> {
  const { data } = await api.post<{ visit: VisitTemplate }>(`/api/v1/edc/studies/${studyId}/visits`, payload);
  return data.visit;
}

export async function enrollSubject(studyId: string, payload: EnrollSubjectPayload): Promise<StudySubject> {
  const { data } = await api.post<{ subject: StudySubject }>(`/api/v1/edc/studies/${studyId}/subjects`, payload);
  return data.subject;
}

export async function listStudySubjects(studyId: string, limit = 50): Promise<StudySubject[]> {
  const { data } = await api.get<{ items: StudySubject[] }>(`/api/v1/edc/studies/${studyId}/subjects`, {
    params: { limit }
  });
  return data.items ?? [];
}

export async function createConsentVersion(studyId: string, payload: CreateConsentVersionPayload): Promise<ConsentVersion> {
  const { data } = await api.post<{ version: ConsentVersion }>(`/api/v1/edc/studies/${studyId}/consents`, payload);
  return data.version;
}

export async function listConsentVersions(studyId: string, limit = 50): Promise<ConsentVersion[]> {
  const { data } = await api.get<{ items: ConsentVersion[] }>(`/api/v1/edc/studies/${studyId}/consents`, {
    params: { limit }
  });
  return data.items ?? [];
}

export async function recordConsent(subjectId: string, payload: RecordConsentPayload): Promise<ConsentSignature> {
  const { data } = await api.post<{ signature: ConsentSignature }>(`/api/v1/edc/subjects/${subjectId}/consents`, payload);
  return data.signature;
}

export async function listStudyAuditLogs(studyId: string, limit = 100): Promise<AuditLogEntry[]> {
  const { data } = await api.get<{ items: AuditLogEntry[] }>(`/api/v1/edc/studies/${studyId}/audit`, {
    params: { limit }
  });
  return data.items ?? [];
}
