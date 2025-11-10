"use client";

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore - resolved within frontend workspace dependencies
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AuditLogEntry,
  ConsentVersion,
  CreateConsentVersionPayload,
  CreateStudyFormPayload,
  CreateStudyPayload,
  CreateStudySitePayload,
  CreateVisitTemplatePayload,
  EnrollSubjectPayload,
  Study,
  StudySubject,
  createConsentVersion,
  createStudy,
  createStudyForm,
  createStudySite,
  createVisitTemplate,
  enrollSubject,
  getStudy,
  listConsentVersions,
  listStudies,
  listStudyAuditLogs,
  listStudySubjects,
  recordConsent,
  updateStudyStatus,
  RecordConsentPayload
} from "../lib/api";

const fallbackStudies: Study[] = [];
const fallbackSubjects: StudySubject[] = [];
const fallbackAudit: AuditLogEntry[] = [];
const fallbackConsents: ConsentVersion[] = [];

export function useStudies(limit = 25) {
  return useQuery({
    queryKey: ["studies", limit],
    queryFn: () => listStudies(limit),
    initialData: fallbackStudies
  });
}

export function useStudy(studyId?: string) {
  return useQuery({
    queryKey: ["study", studyId],
    queryFn: () => (studyId ? getStudy(studyId) : Promise.reject(new Error("missing study id"))),
    enabled: Boolean(studyId)
  });
}

export function useCreateStudy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateStudyPayload) => createStudy(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["studies"] });
    }
  });
}

export function useUpdateStudyStatus(studyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (status: string) => updateStudyStatus(studyId, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["study", studyId] });
      queryClient.invalidateQueries({ queryKey: ["studies"] });
    }
  });
}

export function useCreateStudySite(studyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateStudySitePayload) => createStudySite(studyId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["study", studyId] });
    }
  });
}

export function useCreateStudyForm(studyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateStudyFormPayload) => createStudyForm(studyId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["study", studyId] });
    }
  });
}

export function useCreateVisitTemplate(studyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateVisitTemplatePayload) => createVisitTemplate(studyId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["study", studyId] });
    }
  });
}

export function useEnrollSubject(studyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: EnrollSubjectPayload) => enrollSubject(studyId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["study", studyId] });
      queryClient.invalidateQueries({ queryKey: ["study-subjects", studyId] });
    }
  });
}

export function useStudySubjects(studyId?: string, limit = 50) {
  return useQuery({
    queryKey: ["study-subjects", studyId, limit],
    queryFn: () => (studyId ? listStudySubjects(studyId, limit) : Promise.resolve(fallbackSubjects)),
    initialData: fallbackSubjects,
    enabled: Boolean(studyId)
  });
}

export function useConsentVersions(studyId?: string, limit = 50) {
  return useQuery({
    queryKey: ["study-consents", studyId, limit],
    queryFn: () => (studyId ? listConsentVersions(studyId, limit) : Promise.resolve(fallbackConsents)),
    initialData: fallbackConsents,
    enabled: Boolean(studyId)
  });
}

export function useCreateConsentVersion(studyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateConsentVersionPayload) => createConsentVersion(studyId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["study-consents", studyId] });
      queryClient.invalidateQueries({ queryKey: ["study", studyId] });
    }
  });
}

export function useRecordConsent(subjectId: string, studyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: RecordConsentPayload) => recordConsent(subjectId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["study", studyId] });
      queryClient.invalidateQueries({ queryKey: ["study-subjects", studyId] });
    }
  });
}

export function useStudyAuditLogs(studyId?: string, limit = 100) {
  return useQuery({
    queryKey: ["study-audit", studyId, limit],
    queryFn: () => (studyId ? listStudyAuditLogs(studyId, limit) : Promise.resolve(fallbackAudit)),
    initialData: fallbackAudit,
    enabled: Boolean(studyId)
  });
}
