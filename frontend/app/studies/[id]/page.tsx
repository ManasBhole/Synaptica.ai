'use client';

import { useMemo, useState } from "react";
import { useParams } from "next/navigation";
import {
  ArrowPathIcon,
  CheckCircleIcon,
  PlusIcon,
  AdjustmentsHorizontalIcon,
  PencilSquareIcon,
  BuildingOfficeIcon,
  ClipboardDocumentIcon,
  UserPlusIcon,
  ShieldCheckIcon,
  ClockIcon
} from "@heroicons/react/24/solid";
import {
  useConsentVersions,
  useCreateConsentVersion,
  useCreateStudyForm,
  useCreateStudySite,
  useCreateVisitTemplate,
  useEnrollSubject,
  useRecordConsent,
  useStudy,
  useStudyAuditLogs,
  useStudySubjects,
  useUpdateStudyStatus
} from "../../../hooks/useStudies";
import { Study, StudySubject, ConsentVersion, AuditLogEntry } from "../../../lib/api";

const statusOptions = ["draft", "screening", "enrolling", "active", "completed", "closed"] as const;

const statusTone: Record<string, string> = {
  draft: "bg-neutral-100 text-neutral-600",
  screening: "bg-sky-100 text-sky-600",
  enrolling: "bg-amber-100 text-amber-600",
  active: "bg-emerald-100 text-emerald-600",
  completed: "bg-purple-100 text-purple-600",
  closed: "bg-rose-100 text-rose-600"
};

const fieldTypes = [
  { value: "text", label: "Text" },
  { value: "number", label: "Number" },
  { value: "date", label: "Date" },
  { value: "select", label: "Select" }
];

type FormFieldDraft = {
  key: string;
  label: string;
  type: string;
  required: boolean;
  options?: string;
};

type SubjectDraft = {
  subjectCode: string;
  siteId?: string;
  randomizationArm?: string;
  demographics?: string;
};

type ConsentVersionDraft = {
  version: string;
  title: string;
  summary: string;
  documentUrl: string;
  effectiveAt: string;
  supersededAt?: string;
};

type ConsentDraft = {
  subjectId: string;
  consentVersionId: string;
  signedAt: string;
  signerName: string;
  method: string;
  ipAddress: string;
};

export default function StudyDetailPage() {
  const params = useParams<{ id: string }>();
  const studyId = params?.id;

  const [activeTab, setActiveTab] = useState("overview");
  const [siteDraft, setSiteDraft] = useState({ siteCode: "", name: "", country: "", principalInvestigator: "", contactEmail: "" });
  const [formDraft, setFormDraft] = useState({ name: "", slug: "", description: "", fields: [{ key: "subject_id", label: "Subject ID", type: "text", required: true } as FormFieldDraft] });
  const [visitDraft, setVisitDraft] = useState({ name: "Baseline", visitOrder: 1, required: true, windowStartDays: "", windowEndDays: "", forms: [] as string[] });
  const [subjectDraft, setSubjectDraft] = useState<SubjectDraft>({ subjectCode: "" });
  const [consentVersionDraft, setConsentVersionDraft] = useState<ConsentVersionDraft>({ version: "v1.0", title: "Informed Consent", summary: "", documentUrl: "", effectiveAt: new Date().toISOString().slice(0, 10) });
  const [consentDraft, setConsentDraft] = useState<ConsentDraft>({ subjectId: "", consentVersionId: "", signedAt: new Date().toISOString(), signerName: "", method: "electronic", ipAddress: "" });

  const studyQuery = useStudy(studyId);
  const subjectsQuery = useStudySubjects(studyId ?? "");
  const auditQuery = useStudyAuditLogs(studyId ?? "");
  const consentVersionsQuery = useConsentVersions(studyId ?? "");

  const updateStatus = useUpdateStudyStatus(studyId ?? "");
  const createSite = useCreateStudySite(studyId ?? "");
  const createForm = useCreateStudyForm(studyId ?? "");
  const createVisit = useCreateVisitTemplate(studyId ?? "");
  const enrollSubjectMutation = useEnrollSubject(studyId ?? "");
  const createConsentVersionMutation = useCreateConsentVersion(studyId ?? "");
  const recordConsentMutation = useRecordConsent(consentDraft.subjectId || "", studyId ?? "");

  const study = studyQuery.data as Study | undefined;
  const subjects = subjectsQuery.data as StudySubject[];
  const consentVersions = consentVersionsQuery.data as ConsentVersion[];
  const auditLogs = auditQuery.data as AuditLogEntry[];

  const forms = study?.forms ?? [];
  const sites = study?.sites ?? [];
  const visits = study?.visitTemplates ?? [];

  const statusDisplay = study ? statusTone[study.status] ?? "bg-neutral-100 text-neutral-600" : "bg-neutral-100 text-neutral-600";

  if (!studyId) {
    return <div className="glass-panel px-6 py-6 text-neutral-600">Missing study id.</div>;
  }

  if (!study && studyQuery.isLoading) {
    return (
      <div className="glass-panel px-6 py-12 text-center text-neutral-500">
        Loading study...
      </div>
    );
  }

  if (!study) {
    return (
      <div className="glass-panel px-6 py-12 text-center text-neutral-500">
        Study not found or access restricted.
      </div>
    );
  }

  const subjectOptions = subjects.map((subject) => ({ value: subject.id, label: `${subject.subjectCode} · ${subject.status}` }));

  const handleUpdateStatus = (next: string) => {
    if (!next || next === study.status) return;
    updateStatus.mutate(next);
  };

  const handleCreateSite = () => {
    if (!siteDraft.siteCode || !siteDraft.name) return;
    createSite.mutate(
      {
        siteCode: siteDraft.siteCode.trim(),
        name: siteDraft.name.trim(),
        country: siteDraft.country.trim() || undefined,
        principalInvestigator: siteDraft.principalInvestigator.trim() || undefined,
        contact: siteDraft.contactEmail ? { email: siteDraft.contactEmail.trim() } : undefined
      },
      {
        onSuccess: () => {
          setSiteDraft({ siteCode: "", name: "", country: "", principalInvestigator: "", contactEmail: "" });
        }
      }
    );
  };

  const handleAddField = () => {
    setFormDraft((prev) => ({ ...prev, fields: [...prev.fields, { key: "", label: "", type: "text", required: false }] }));
  };

  const handleCreateForm = () => {
    if (!formDraft.name || !formDraft.slug) return;
    const fieldsSchema = formDraft.fields
      .filter((field) => field.key && field.label)
      .map((field) => ({
        key: field.key,
        label: field.label,
        type: field.type,
        required: field.required,
        options: field.options
          ? field.options
              .split(",")
              .map((option) => option.trim())
              .filter(Boolean)
          : undefined
      }));
    if (fieldsSchema.length === 0) return;
    const schema = {
      title: formDraft.name,
      version: 1,
      fields: fieldsSchema
    } as Record<string, unknown>;
    createForm.mutate(
      {
        name: formDraft.name,
        slug: formDraft.slug,
        description: formDraft.description || undefined,
        schema,
        status: "draft"
      },
      {
        onSuccess: () => {
          setFormDraft({ name: "", slug: "", description: "", fields: [{ key: "subject_id", label: "Subject ID", type: "text", required: true }] });
        }
      }
    );
  };

  const handleCreateVisit = () => {
    if (!visitDraft.name || !visitDraft.visitOrder) return;
    createVisit.mutate(
      {
        name: visitDraft.name,
        visitOrder: Number(visitDraft.visitOrder),
        required: visitDraft.required,
        windowStartDays: visitDraft.windowStartDays ? Number(visitDraft.windowStartDays) : undefined,
        windowEndDays: visitDraft.windowEndDays ? Number(visitDraft.windowEndDays) : undefined,
        forms: visitDraft.forms
      },
      {
        onSuccess: () => {
          setVisitDraft({ name: "Baseline", visitOrder: 1, required: true, windowStartDays: "", windowEndDays: "", forms: [] });
        }
      }
    );
  };

  const handleEnrollSubject = () => {
    if (!subjectDraft.subjectCode) return;
    let demographics: Record<string, unknown> | undefined;
    if (subjectDraft.demographics) {
      try {
        demographics = JSON.parse(subjectDraft.demographics);
      } catch (error) {
        demographics = { note: subjectDraft.demographics };
      }
    }
    enrollSubjectMutation.mutate(
      {
        subjectCode: subjectDraft.subjectCode,
        siteId: subjectDraft.siteId || undefined,
        randomizationArm: subjectDraft.randomizationArm || undefined,
        demographics
      },
      {
        onSuccess: () => {
          setSubjectDraft({ subjectCode: "" });
        }
      }
    );
  };

  const handleCreateConsentVersion = () => {
    if (!consentVersionDraft.version || !consentVersionDraft.effectiveAt) return;
    createConsentVersionMutation.mutate(
      {
        version: consentVersionDraft.version,
        title: consentVersionDraft.title || undefined,
        summary: consentVersionDraft.summary || undefined,
        documentUrl: consentVersionDraft.documentUrl || undefined,
        effectiveAt: new Date(consentVersionDraft.effectiveAt).toISOString(),
        supersededAt: consentVersionDraft.supersededAt ? new Date(consentVersionDraft.supersededAt).toISOString() : undefined
      },
      {
        onSuccess: () => {
          setConsentVersionDraft({ version: "v1.0", title: "Informed Consent", summary: "", documentUrl: "", effectiveAt: new Date().toISOString().slice(0, 10) });
        }
      }
    );
  };

  const handleRecordConsent = () => {
    if (!consentDraft.subjectId || !consentDraft.consentVersionId) return;
    recordConsentMutation.mutate(
      {
        consentVersionId: consentDraft.consentVersionId,
        signedAt: consentDraft.signedAt,
        signerName: consentDraft.signerName || undefined,
        method: consentDraft.method || undefined,
        ipAddress: consentDraft.ipAddress || undefined
      },
      {
        onSuccess: () => {
          setConsentDraft({ subjectId: "", consentVersionId: "", signedAt: new Date().toISOString(), signerName: "", method: "electronic", ipAddress: "" });
        }
      }
    );
  };

  const tabButton = (key: string, label: string, icon: JSX.Element) => (
    <button
      key={key}
      type="button"
      onClick={() => setActiveTab(key)}
      className={`inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs font-semibold transition ${
        activeTab === key ? "bg-gradient-to-r from-brand-500/30 via-brand-500/10 to-transparent text-brand-700" : "text-neutral-500 hover:bg-white"
      }`}
    >
      {icon}
      {label}
    </button>
  );

  return (
    <div className="space-y-10">
      <section className="glass-panel px-6 py-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <p className="text-[11px] uppercase tracking-[0.32em] text-neutral-400">Protocol {study.code}</p>
            <h1 className="mt-2 text-2xl font-semibold text-neutral-900">{study.name}</h1>
            <p className="mt-3 max-w-2xl text-sm text-neutral-500">
              {study.protocolSummary?.abstract ?? "Define visits, capture EDC forms, monitor cohorts, and keep audit-ready compliance from one cockpit."}
            </p>
            <div className="mt-4 flex flex-wrap gap-2 text-xs text-neutral-400">
              {study.phase && <span className="rounded-full bg-neutral-50 px-3 py-1 text-neutral-600">Phase: {study.phase}</span>}
              {study.therapeuticArea && <span className="rounded-full bg-neutral-50 px-3 py-1 text-neutral-600">Area: {study.therapeuticArea}</span>}
              {study.sponsor && <span className="rounded-full bg-neutral-50 px-3 py-1 text-neutral-600">Sponsor: {study.sponsor}</span>}
              <span className="rounded-full bg-neutral-50 px-3 py-1 text-neutral-600">Sites: {sites.length}</span>
              <span className="rounded-full bg-neutral-50 px-3 py-1 text-neutral-600">Subjects: {study.totalSubjects ?? 0}</span>
            </div>
          </div>
          <div className="flex flex-col items-end gap-4">
            <div className={`inline-flex items-center gap-2 rounded-full px-4 py-1 text-xs font-semibold ${statusDisplay}`}>
              Status
              <select
                value={study.status}
                onChange={(event) => handleUpdateStatus(event.target.value)}
                className="bg-transparent text-xs font-semibold text-neutral-900 focus:outline-none"
              >
                {statusOptions.map((status) => (
                  <option key={status} value={status} className="bg-slate-900 text-slate-100">
                    {status}
                  </option>
                ))}
              </select>
            </div>
            <button
              type="button"
              onClick={() => studyQuery.refetch()}
              className="inline-flex items-center gap-2 rounded-full border border-neutral-200 bg-neutral-50 px-4 py-2 text-xs font-semibold text-neutral-600 transition hover:border-neutral-200 hover:text-neutral-900"
            >
              <ArrowPathIcon className={`h-4 w-4 ${studyQuery.isFetching ? "animate-spin" : ""}`} /> Refresh
            </button>
            {updateStatus.status === "success" && (
              <span className="inline-flex items-center gap-2 text-xs text-emerald-600">
                <CheckCircleIcon className="h-4 w-4" /> Status updated
              </span>
            )}
          </div>
        </div>
        <div className="mt-6 flex flex-wrap gap-3">
          {tabButton("overview", "Overview", <AdjustmentsHorizontalIcon className="h-4 w-4" />)}
          {tabButton("sites", "Sites", <BuildingOfficeIcon className="h-4 w-4" />)}
          {tabButton("forms", "Form Builder", <ClipboardDocumentIcon className="h-4 w-4" />)}
          {tabButton("visits", "Visit Schedule", <ClockIcon className="h-4 w-4" />)}
          {tabButton("subjects", "Subjects", <UserPlusIcon className="h-4 w-4" />)}
          {tabButton("consents", "Consent", <ShieldCheckIcon className="h-4 w-4" />)}
          {tabButton("audit", "Audit", <PencilSquareIcon className="h-4 w-4" />)}
        </div>
      </section>

      {activeTab === "overview" && (
        <section className="glass-panel px-6 py-6">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <div className="rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-neutral-600">
              <p className="text-xs uppercase tracking-[0.32em] text-neutral-400">Sites</p>
              <p className="mt-2 text-2xl font-semibold text-neutral-900">{sites.length}</p>
              <p className="mt-1 text-xs text-neutral-400">Across geographies</p>
            </div>
            <div className="rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-neutral-600">
              <p className="text-xs uppercase tracking-[0.32em] text-neutral-400">Active Subjects</p>
              <p className="mt-2 text-2xl font-semibold text-neutral-900">{study.activeSubjects ?? 0}</p>
              <p className="mt-1 text-xs text-neutral-400">On treatment or follow-up</p>
            </div>
            <div className="rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-neutral-600">
              <p className="text-xs uppercase tracking-[0.32em] text-neutral-400">Forms</p>
              <p className="mt-2 text-2xl font-semibold text-neutral-900">{forms.length}</p>
              <p className="mt-1 text-xs text-neutral-400">eCRFs available</p>
            </div>
            <div className="rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-neutral-600">
              <p className="text-xs uppercase tracking-[0.32em] text-neutral-400">Visit templates</p>
              <p className="mt-2 text-2xl font-semibold text-neutral-900">{visits.length}</p>
              <p className="mt-1 text-xs text-neutral-400">Scheduled touchpoints</p>
            </div>
          </div>
          <div className="mt-6 grid gap-4 md:grid-cols-2">
            <div className="rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-neutral-600">
              <h3 className="text-lg font-semibold text-neutral-900">Timeline</h3>
              <ul className="mt-4 space-y-2 text-sm">
                <li className="flex justify-between text-neutral-500">
                  <span>Start</span>
                  <span>{study.startDate ? new Date(study.startDate).toLocaleDateString() : "TBD"}</span>
                </li>
                <li className="flex justify-between text-neutral-500">
                  <span>End</span>
                  <span>{study.endDate ? new Date(study.endDate).toLocaleDateString() : "TBD"}</span>
                </li>
                <li className="flex justify-between text-neutral-500">
                  <span>Last updated</span>
                  <span>{new Date(study.updatedAt).toLocaleString()}</span>
                </li>
              </ul>
            </div>
            <div className="rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-neutral-600">
              <h3 className="text-lg font-semibold text-neutral-900">Recent Audit Trail</h3>
              <ul className="mt-4 space-y-3 text-xs text-neutral-500">
                {auditLogs.slice(0, 6).map((log) => (
                  <li key={log.id} className="flex flex-col rounded-2xl border border-neutral-200 bg-black/20 px-3 py-3">
                    <div className="flex items-center justify-between text-neutral-600">
                      <span className="font-semibold text-neutral-900">{log.action}</span>
                      <span>{new Date(log.createdAt).toLocaleString()}</span>
                    </div>
                    <span className="mt-1">Actor: {log.actor}</span>
                    {log.entity && <span>Entity: {log.entity}</span>}
                  </li>
                ))}
                {auditLogs.length === 0 && <li>No audit entries yet.</li>}
              </ul>
            </div>
          </div>
        </section>
      )}

      {activeTab === "sites" && (
        <section className="glass-panel px-6 py-6">
          <div className="flex flex-col gap-6 lg:flex-row">
            <div className="flex-1">
              <h3 className="text-lg font-semibold text-neutral-900">Participating Sites</h3>
              <div className="mt-4 space-y-3">
                {sites.map((site) => (
                  <div key={site.id} className="rounded-3xl border border-neutral-200 bg-neutral-50 px-4 py-4 text-sm text-neutral-600">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-neutral-900 font-semibold">{site.name}</p>
                        <p className="text-xs uppercase tracking-[0.22em] text-neutral-400">{site.siteCode}</p>
                      </div>
                      <span className="text-xs text-neutral-400">{new Date(site.createdAt).toLocaleDateString()}</span>
                    </div>
                    <p className="mt-2 text-neutral-500">Principal Investigator: {site.principalInvestigator ?? "—"}</p>
                    <p className="text-neutral-500">Country: {site.country ?? "—"}</p>
                    {site.contact?.email && <p className="text-neutral-400">Contact: {String(site.contact.email)}</p>}
                  </div>
                ))}
                {sites.length === 0 && <div className="rounded-3xl border border-neutral-200 bg-neutral-50 px-4 py-6 text-center text-neutral-400">No sites yet</div>}
              </div>
            </div>
            <div className="w-full max-w-sm rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-sm text-neutral-600">
              <h4 className="text-base font-semibold text-neutral-900">Add site</h4>
              <div className="mt-4 space-y-3">
                <input
                  value={siteDraft.siteCode}
                  onChange={(event) => setSiteDraft((prev) => ({ ...prev, siteCode: event.target.value }))}
                  placeholder="SITE-001"
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <input
                  value={siteDraft.name}
                  onChange={(event) => setSiteDraft((prev) => ({ ...prev, name: event.target.value }))}
                  placeholder="City Medical Center"
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <input
                  value={siteDraft.country}
                  onChange={(event) => setSiteDraft((prev) => ({ ...prev, country: event.target.value }))}
                  placeholder="USA"
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <input
                  value={siteDraft.principalInvestigator}
                  onChange={(event) => setSiteDraft((prev) => ({ ...prev, principalInvestigator: event.target.value }))}
                  placeholder="Dr. Jane Doe"
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <input
                  value={siteDraft.contactEmail}
                  onChange={(event) => setSiteDraft((prev) => ({ ...prev, contactEmail: event.target.value }))}
                  placeholder="pi@hospital.org"
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <button
                  type="button"
                  onClick={handleCreateSite}
                  disabled={createSite.status === "pending"}
                  className="inline-flex w-full items-center justify-center gap-2 rounded-full bg-gradient-to-r from-brand-500 via-accent-500 to-accent-400 px-5 py-2 text-sm font-semibold text-white shadow-glow transition hover:shadow-[0_12px_35px_rgba(244,63,94,0.35)] disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {createSite.status === "pending" ? <ArrowPathIcon className="h-5 w-5 animate-spin" /> : <PlusIcon className="h-5 w-5" />} Add site
                </button>
                {createSite.status === "error" && <p className="text-xs text-rose-300">Failed to create site</p>}
                {createSite.status === "success" && <p className="text-xs text-emerald-600">Site added</p>}
              </div>
            </div>
          </div>
        </section>
      )}

      {activeTab === "forms" && (
        <section className="glass-panel px-6 py-6">
          <div className="grid gap-6 lg:grid-cols-[2fr_1fr]">
            <div>
              <h3 className="text-lg font-semibold text-neutral-900">Form library</h3>
              <div className="mt-4 space-y-3">
                {forms.map((form) => (
                  <div key={form.id} className="rounded-3xl border border-neutral-200 bg-neutral-50 px-5 py-4 text-sm text-neutral-600">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-neutral-900 font-semibold">{form.name}</p>
                        <p className="text-xs uppercase tracking-[0.22em] text-neutral-400">{form.slug}</p>
                      </div>
                      <span className="text-xs text-neutral-400">v{form.version}</span>
                    </div>
                    <p className="mt-2 text-neutral-500">{form.description ?? "No description"}</p>
                    <p className="mt-2 text-xs text-neutral-400">Fields: {Array.isArray((form.schema as any)?.fields) ? (form.schema as any).fields.length : 0}</p>
                  </div>
                ))}
                {forms.length === 0 && <div className="rounded-3xl border border-neutral-200 bg-neutral-50 px-4 py-6 text-center text-neutral-400">No forms yet</div>}
              </div>
            </div>
            <div className="rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-sm text-neutral-600">
              <h4 className="text-base font-semibold text-neutral-900">Compose form</h4>
              <div className="mt-4 space-y-3">
                <input
                  value={formDraft.name}
                  onChange={(event) => setFormDraft((prev) => ({ ...prev, name: event.target.value }))}
                  placeholder="Adverse event log"
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <input
                  value={formDraft.slug}
                  onChange={(event) => setFormDraft((prev) => ({ ...prev, slug: event.target.value }))}
                  placeholder="adverse-event"
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <textarea
                  value={formDraft.description}
                  onChange={(event) => setFormDraft((prev) => ({ ...prev, description: event.target.value }))}
                  placeholder="Purpose, context, reference"
                  className="h-24 w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-3 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <div className="rounded-2xl border border-neutral-200 bg-black/20 p-3">
                  <div className="flex items-center justify-between text-xs text-neutral-500">
                    <span>Fields</span>
                    <button type="button" onClick={handleAddField} className="inline-flex items-center gap-1 text-brand-600">
                      <PlusIcon className="h-3 w-3" />
                      Add field
                    </button>
                  </div>
                  <div className="mt-3 space-y-2">
                    {formDraft.fields.map((field, index) => (
                      <div key={index} className="grid gap-2 rounded-2xl border border-neutral-200 bg-neutral-50 p-3">
                        <div className="grid gap-2 md:grid-cols-2">
                          <input
                            value={field.key}
                            onChange={(event) => setFormDraft((prev) => {
                              const next = [...prev.fields];
                              next[index].key = event.target.value;
                              return { ...prev, fields: next };
                            })}
                            placeholder="field_key"
                            className="rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                          />
                          <input
                            value={field.label}
                            onChange={(event) => setFormDraft((prev) => {
                              const next = [...prev.fields];
                              next[index].label = event.target.value;
                              return { ...prev, fields: next };
                            })}
                            placeholder="Field label"
                            className="rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                          />
                        </div>
                        <div className="grid gap-2 md:grid-cols-2">
                          <select
                            value={field.type}
                            onChange={(event) => setFormDraft((prev) => {
                              const next = [...prev.fields];
                              next[index].type = event.target.value;
                              return { ...prev, fields: next };
                            })}
                            className="rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                          >
                            {fieldTypes.map((type) => (
                              <option key={type.value} value={type.value} className="bg-surface-raised text-slate-900">
                                {type.label}
                              </option>
                            ))}
                          </select>
                          <label className="inline-flex items-center gap-2 text-xs text-neutral-400">
                            <input
                              type="checkbox"
                              checked={field.required}
                              onChange={(event) => setFormDraft((prev) => {
                                const next = [...prev.fields];
                                next[index].required = event.target.checked;
                                return { ...prev, fields: next };
                              })}
                              className="h-4 w-4 rounded border-neutral-200 bg-neutral-50 text-brand-400 focus:ring-brand-400"
                            />
                            Required
                          </label>
                        </div>
                        {field.type === "select" && (
                          <input
                            value={field.options ?? ""}
                            onChange={(event) => setFormDraft((prev) => {
                              const next = [...prev.fields];
                              next[index].options = event.target.value;
                              return { ...prev, fields: next };
                            })}
                            placeholder="Option1, Option2"
                            className="rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                          />
                        )}
                      </div>
                    ))}
                  </div>
                </div>
                <button
                  type="button"
                  onClick={handleCreateForm}
                  disabled={createForm.status === "pending"}
                  className="inline-flex w-full items-center justify-center gap-2 rounded-full bg-gradient-to-r from-brand-500 via-accent-500 to-accent-400 px-5 py-2 text-sm font-semibold text-white shadow-glow transition hover:shadow-[0_12px_35px_rgba(244,63,94,0.35)] disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {createForm.status === "pending" ? <ArrowPathIcon className="h-5 w-5 animate-spin" /> : <PlusIcon className="h-5 w-5" />} Publish form
                </button>
                {createForm.status === "error" && <p className="text-xs text-rose-300">Failed to create form</p>}
                {createForm.status === "success" && <p className="text-xs text-emerald-600">Form created</p>}
              </div>
            </div>
          </div>
        </section>
      )}

      {activeTab === "visits" && (
        <section className="glass-panel px-6 py-6">
          <div className="grid gap-6 lg:grid-cols-[2fr_1fr]">
            <div>
              <h3 className="text-lg font-semibold text-neutral-900">Visit schedule</h3>
              <div className="mt-4 space-y-3">
                {visits.map((visit) => (
                  <div key={visit.id} className="rounded-3xl border border-neutral-200 bg-neutral-50 px-5 py-4 text-sm text-neutral-600">
                    <div className="flex items-center justify-between">
                      <div className="text-neutral-900 font-semibold">{visit.name}</div>
                      <span className="rounded-full bg-white px-3 py-1 text-xs font-semibold text-neutral-600">Day {visit.visitOrder}</span>
                    </div>
                    <div className="mt-2 text-neutral-500">Window: {visit.windowStartDays ?? 0} to {visit.windowEndDays ?? 0} days</div>
                    <div className="text-neutral-500">Required: {visit.required ? "Yes" : "Optional"}</div>
                    <div className="text-neutral-400">Forms: {visit.forms?.length ? visit.forms.join(", ") : "—"}</div>
                  </div>
                ))}
                {visits.length === 0 && <div className="rounded-3xl border border-neutral-200 bg-neutral-50 px-4 py-6 text-center text-neutral-400">No visit templates yet</div>}
              </div>
            </div>
            <div className="rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-sm text-neutral-600">
              <h4 className="text-base font-semibold text-neutral-900">Add visit</h4>
              <div className="mt-4 space-y-3">
                <input
                  value={visitDraft.name}
                  onChange={(event) => setVisitDraft((prev) => ({ ...prev, name: event.target.value }))}
                  placeholder="Week 12"
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <input
                  value={visitDraft.visitOrder}
                  onChange={(event) => setVisitDraft((prev) => ({ ...prev, visitOrder: Number(event.target.value) || 1 }))}
                  placeholder="Order"
                  type="number"
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <div className="grid grid-cols-2 gap-3">
                  <input
                    value={visitDraft.windowStartDays}
                    onChange={(event) => setVisitDraft((prev) => ({ ...prev, windowStartDays: event.target.value }))}
                    placeholder="-2"
                    className="rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  />
                  <input
                    value={visitDraft.windowEndDays}
                    onChange={(event) => setVisitDraft((prev) => ({ ...prev, windowEndDays: event.target.value }))}
                    placeholder="2"
                    className="rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  />
                </div>
                <label className="inline-flex items-center gap-2 text-xs text-neutral-400">
                  <input
                    type="checkbox"
                    checked={visitDraft.required}
                    onChange={(event) => setVisitDraft((prev) => ({ ...prev, required: event.target.checked }))}
                    className="h-4 w-4 rounded border-neutral-200 bg-neutral-50 text-brand-400 focus:ring-brand-400"
                  />
                  Required visit
                </label>
                <label className="text-xs text-neutral-400">
                  Forms (comma separated slugs)
                  <input
                    value={visitDraft.forms.join(", ")}
                    onChange={(event) => setVisitDraft((prev) => ({ ...prev, forms: event.target.value.split(",").map((item) => item.trim()).filter(Boolean) }))}
                    placeholder="screening, lab-panel"
                    className="mt-1 w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  />
                </label>
                <button
                  type="button"
                  onClick={handleCreateVisit}
                  disabled={createVisit.status === "pending"}
                  className="inline-flex w-full items-center justify-center gap-2 rounded-full bg-gradient-to-r from-brand-500 via-accent-500 to-accent-400 px-5 py-2 text-sm font-semibold text-white shadow-glow transition hover:shadow-[0_12px_35px_rgba(244,63,94,0.35)] disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {createVisit.status === "pending" ? <ArrowPathIcon className="h-5 w-5 animate-spin" /> : <PlusIcon className="h-5 w-5" />} Add visit
                </button>
                {createVisit.status === "error" && <p className="text-xs text-rose-300">Failed to create visit</p>}
                {createVisit.status === "success" && <p className="text-xs text-emerald-600">Visit created</p>}
              </div>
            </div>
          </div>
        </section>
      )}

      {activeTab === "subjects" && (
        <section className="glass-panel px-6 py-6">
          <div className="grid gap-6 lg:grid-cols-[2fr_1fr]">
            <div>
              <h3 className="text-lg font-semibold text-neutral-900">Subjects</h3>
              <div className="mt-4 overflow-x-auto">
                <table className="min-w-full divide-y divide-neutral-200 text-left text-sm text-neutral-600">
                  <thead>
                    <tr>
                      <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-neutral-400">Subject</th>
                      <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-neutral-400">Site</th>
                      <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-neutral-400">Status</th>
                      <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-neutral-400">Randomization</th>
                      <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-neutral-400">Consented</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-neutral-200">
                    {subjects.map((subject) => (
                      <tr key={subject.id} className="hover:bg-neutral-50">
                        <td className="px-4 py-3 font-semibold text-neutral-900">{subject.subjectCode}</td>
                        <td className="px-4 py-3 text-neutral-500">{sites.find((site) => site.id === subject.siteId)?.name ?? "—"}</td>
                        <td className="px-4 py-3 text-neutral-500">{subject.status}</td>
                        <td className="px-4 py-3 text-neutral-500">{subject.randomizationArm ?? "—"}</td>
                        <td className="px-4 py-3 text-neutral-500">{subject.consentedAt ? new Date(subject.consentedAt).toLocaleDateString() : "Pending"}</td>
                      </tr>
                    ))}
                    {subjects.length === 0 && (
                      <tr>
                        <td colSpan={5} className="px-4 py-8 text-center text-sm text-neutral-400">
                          No subjects enrolled yet.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
            <div className="rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-sm text-neutral-600">
              <h4 className="text-base font-semibold text-neutral-900">Enroll subject</h4>
              <div className="mt-4 space-y-3">
                <input
                  value={subjectDraft.subjectCode}
                  onChange={(event) => setSubjectDraft((prev) => ({ ...prev, subjectCode: event.target.value }))}
                  placeholder="SUB-001"
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <select
                  value={subjectDraft.siteId ?? ""}
                  onChange={(event) => setSubjectDraft((prev) => ({ ...prev, siteId: event.target.value || undefined }))}
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                >
                  <option value="" className="bg-surface-raised text-slate-900">
                    Assign site (optional)
                  </option>
                  {sites.map((site) => (
                    <option key={site.id} value={site.id} className="bg-surface-raised text-slate-900">
                      {site.name}
                    </option>
                  ))}
                </select>
                <input
                  value={subjectDraft.randomizationArm ?? ""}
                  onChange={(event) => setSubjectDraft((prev) => ({ ...prev, randomizationArm: event.target.value }))}
                  placeholder="Arm A"
                  className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <textarea
                  value={subjectDraft.demographics ?? ""}
                  onChange={(event) => setSubjectDraft((prev) => ({ ...prev, demographics: event.target.value }))}
                  placeholder='Optional JSON demographics e.g. {"age": 56, "sex": "female"}'
                  className="h-24 w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-3 text-neutral-900 focus:border-brand-400 focus:outline-none"
                />
                <button
                  type="button"
                  onClick={handleEnrollSubject}
                  disabled={enrollSubjectMutation.status === "pending"}
                  className="inline-flex w-full items-center justify-center gap-2 rounded-full bg-gradient-to-r from-brand-500 via-accent-500 to-accent-400 px-5 py-2 text-sm font-semibold text-neutral-900 shadow-glow transition hover:shadow-[0_12px_35px_rgba(244,63,94,0.35)] disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {enrollSubjectMutation.status === "pending" ? <ArrowPathIcon className="h-5 w-5 animate-spin" /> : <PlusIcon className="h-5 w-5" />} Enroll
                </button>
                {enrollSubjectMutation.status === "error" && <p className="text-xs text-rose-300">Failed to enroll subject</p>}
                {enrollSubjectMutation.status === "success" && <p className="text-xs text-emerald-600">Subject enrolled</p>}
              </div>
            </div>
          </div>
        </section>
      )}

      {activeTab === "consents" && (
        <section className="glass-panel px-6 py-6">
          <div className="grid gap-6 lg:grid-cols-[2fr_1fr]">
            <div>
              <h3 className="text-lg font-semibold text-neutral-900">Consent versions</h3>
              <div className="mt-4 space-y-3">
                {consentVersions.map((version) => (
                  <div key={version.id} className="rounded-3xl border border-neutral-200 bg-neutral-50 px-5 py-4 text-sm text-neutral-600">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-neutral-900 font-semibold">{version.title ?? version.version}</p>
                        <p className="text-xs uppercase tracking-[0.22em] text-neutral-400">{version.version}</p>
                      </div>
                      <span className="text-xs text-neutral-400">Effective {new Date(version.effectiveAt).toLocaleDateString()}</span>
                    </div>
                    <p className="mt-2 text-neutral-500">{version.summary ?? "—"}</p>
                    {version.documentUrl && (
                      <a href={version.documentUrl} target="_blank" rel="noreferrer" className="text-xs text-brand-600">
                        View document
                      </a>
                    )}
                  </div>
                ))}
                {consentVersions.length === 0 && <div className="rounded-3xl border border-neutral-200 bg-neutral-50 px-4 py-6 text-center text-neutral-400">No consent versions yet</div>}
              </div>
            </div>
            <div className="space-y-6">
              <div className="rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-sm text-neutral-600">
                <h4 className="text-base font-semibold text-neutral-900">New consent version</h4>
                <div className="mt-4 space-y-3">
                  <input
                    value={consentVersionDraft.version}
                    onChange={(event) => setConsentVersionDraft((prev) => ({ ...prev, version: event.target.value }))}
                    placeholder="v1.0"
                    className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  />
                  <input
                    value={consentVersionDraft.title}
                    onChange={(event) => setConsentVersionDraft((prev) => ({ ...prev, title: event.target.value }))}
                    placeholder="Informed Consent"
                    className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  />
                  <textarea
                    value={consentVersionDraft.summary}
                    onChange={(event) => setConsentVersionDraft((prev) => ({ ...prev, summary: event.target.value }))}
                    placeholder="High-level summary"
                    className="h-24 w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-3 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  />
                  <input
                    value={consentVersionDraft.documentUrl}
                    onChange={(event) => setConsentVersionDraft((prev) => ({ ...prev, documentUrl: event.target.value }))}
                    placeholder="https://..."
                    className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  />
                  <label className="text-xs text-neutral-400">
                    Effective
                    <input
                      type="date"
                      value={consentVersionDraft.effectiveAt}
                      onChange={(event) => setConsentVersionDraft((prev) => ({ ...prev, effectiveAt: event.target.value }))}
                      className="mt-1 w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                    />
                  </label>
                  <label className="text-xs text-neutral-400">
                    Superseded
                    <input
                      type="date"
                      value={consentVersionDraft.supersededAt ?? ""}
                      onChange={(event) => setConsentVersionDraft((prev) => ({ ...prev, supersededAt: event.target.value || undefined }))}
                      className="mt-1 w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                    />
                  </label>
                  <button
                    type="button"
                    onClick={handleCreateConsentVersion}
                    disabled={createConsentVersionMutation.status === "pending"}
                    className="inline-flex w-full items-center justify-center gap-2 rounded-full bg-gradient-to-r from-brand-500 via-accent-500 to-accent-400 px-5 py-2 text-sm font-semibold text-white shadow-glow transition hover:shadow-[0_12px_35px_rgba(244,63,94,0.35)] disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {createConsentVersionMutation.status === "pending" ? <ArrowPathIcon className="h-5 w-5 animate-spin" /> : <PlusIcon className="h-5 w-5" />} Create version
                  </button>
                  {createConsentVersionMutation.status === "error" && <p className="text-xs text-rose-300">Failed to create consent version</p>}
                  {createConsentVersionMutation.status === "success" && <p className="text-xs text-emerald-600">Consent version created</p>}
                </div>
              </div>
              <div className="rounded-3xl border border-neutral-200 bg-neutral-50 p-5 text-sm text-neutral-600">
                <h4 className="text-base font-semibold text-neutral-900">Record consent</h4>
                <div className="mt-4 space-y-3">
                  <select
                    value={consentDraft.subjectId}
                    onChange={(event) => setConsentDraft((prev) => ({ ...prev, subjectId: event.target.value }))}
                    className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  >
                    <option value="" className="bg-surface-raised text-slate-900">
                      Select subject
                    </option>
                    {subjectOptions.map((option) => (
                      <option key={option.value} value={option.value} className="bg-surface-raised text-slate-900">
                        {option.label}
                      </option>
                    ))}
                  </select>
                  <select
                    value={consentDraft.consentVersionId}
                    onChange={(event) => setConsentDraft((prev) => ({ ...prev, consentVersionId: event.target.value }))}
                    className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  >
                    <option value="" className="bg-surface-raised text-slate-900">
                      Select version
                    </option>
                    {consentVersions.map((version) => (
                      <option key={version.id} value={version.id} className="bg-surface-raised text-slate-900">
                        {version.version}
                      </option>
                    ))}
                  </select>
                  <label className="text-xs text-neutral-400">
                    Signed at
                    <input
                      type="datetime-local"
                      value={consentDraft.signedAt.slice(0, 16)}
                      onChange={(event) => setConsentDraft((prev) => ({ ...prev, signedAt: new Date(event.target.value).toISOString() }))}
                      className="mt-1 w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                    />
                  </label>
                  <input
                    value={consentDraft.signerName}
                    onChange={(event) => setConsentDraft((prev) => ({ ...prev, signerName: event.target.value }))}
                    placeholder="Signer name"
                    className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  />
                  <input
                    value={consentDraft.method}
                    onChange={(event) => setConsentDraft((prev) => ({ ...prev, method: event.target.value }))}
                    placeholder="electronic"
                    className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  />
                  <input
                    value={consentDraft.ipAddress}
                    onChange={(event) => setConsentDraft((prev) => ({ ...prev, ipAddress: event.target.value }))}
                    placeholder="Signer IP"
                    className="w-full rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-2 text-neutral-900 focus:border-brand-400 focus:outline-none"
                  />
                  <button
                    type="button"
                    onClick={handleRecordConsent}
                    disabled={recordConsentMutation.status === "pending" || !consentDraft.subjectId || !consentDraft.consentVersionId}
                    className="inline-flex w-full items-center justify-center gap-2 rounded-full bg-gradient-to-r from-brand-500 via-accent-500 to-accent-400 px-5 py-2 text-sm font-semibold text-neutral-900 shadow-glow transition hover:shadow-[0_12px_35px_rgba(244,63,94,0.35)] disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {recordConsentMutation.status === "pending" ? <ArrowPathIcon className="h-5 w-5 animate-spin" /> : <PlusIcon className="h-5 w-5" />} Record consent
                  </button>
                  {recordConsentMutation.status === "error" && <p className="text-xs text-rose-300">Failed to record consent</p>}
                  {recordConsentMutation.status === "success" && <p className="text-xs text-emerald-600">Consent recorded</p>}
                </div>
              </div>
            </div>
          </div>
        </section>
      )}

      {activeTab === "audit" && (
        <section className="glass-panel px-6 py-6">
          <h3 className="text-lg font-semibold text-neutral-900">Audit log</h3>
          <div className="mt-4 space-y-3">
            {auditLogs.map((log) => (
              <div key={log.id} className="rounded-3xl border border-neutral-200 bg-neutral-50 px-5 py-4 text-sm text-neutral-600">
                <div className="flex items-center justify-between text-neutral-900">
                  <span className="font-semibold">{log.action}</span>
                  <span className="text-xs text-neutral-400">{new Date(log.createdAt).toLocaleString()}</span>
                </div>
                <p className="text-xs text-neutral-400">Actor: {log.actor}{log.role ? ` (${log.role})` : ""}</p>
                {log.entity && <p className="text-xs text-neutral-400">Entity: {log.entity} · {log.entityId}</p>}
                {log.payload && (
                  <pre className="mt-2 max-h-40 overflow-auto rounded-2xl bg-neutral-50 p-3 text-xs text-neutral-500">
                    {JSON.stringify(log.payload, null, 2)}
                  </pre>
                )}
              </div>
            ))}
            {auditLogs.length === 0 && <div className="rounded-3xl border border-neutral-200 bg-neutral-50 px-4 py-6 text-center text-neutral-400">No audit entries yet</div>}
          </div>
        </section>
      )}
    </div>
  );
}
