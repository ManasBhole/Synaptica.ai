'use client';

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowPathIcon, ArrowRightIcon, CheckCircleIcon, PlusIcon } from "@heroicons/react/24/solid";
import { MetricCard } from "../../components/metric-card";
import { Study, CreateStudyPayload } from "../../lib/api";
import { useCreateStudy, useStudies } from "../../hooks/useStudies";

const defaultDraft: CreateStudyPayload & { summary?: string } = {
  code: "",
  name: "",
  phase: "",
  therapeuticArea: "",
  sponsor: "",
  startDate: "",
  endDate: "",
  summary: ""
};

const statusColors: Record<string, string> = {
  draft: "bg-white/10 text-white",
  screening: "bg-sky-500/20 text-sky-200",
  enrolling: "bg-amber-500/20 text-amber-200",
  active: "bg-emerald-500/20 text-emerald-200",
  completed: "bg-purple-500/20 text-purple-200",
  closed: "bg-rose-500/20 text-rose-200"
};

const phases = ["Phase I", "Phase II", "Phase III", "Phase IV", "Observational"];

export default function StudiesPage() {
  const router = useRouter();
  const [showCreate, setShowCreate] = useState(false);
  const [draft, setDraft] = useState(defaultDraft);
  const studiesQuery = useStudies(50);
  const createStudy = useCreateStudy();

  const studies = (studiesQuery.data ?? []) as Study[];

  const stats = useMemo(() => {
    const total = studies.length;
    const active = studies.filter((study) => study.status === "active" || study.status === "enrolling").length;
    const totalSubjects = studies.reduce((sum, study) => sum + (study.totalSubjects ?? 0), 0);
    const activeSubjects = studies.reduce((sum, study) => sum + (study.activeSubjects ?? 0), 0);
    return { total, active, totalSubjects, activeSubjects };
  }, [studies]);

  const handleCreate = () => {
    if (!draft.code || !draft.name) {
      return;
    }
    const payload: CreateStudyPayload = {
      code: draft.code.trim(),
      name: draft.name.trim(),
      phase: draft.phase?.trim() || undefined,
      therapeuticArea: draft.therapeuticArea?.trim() || undefined,
      sponsor: draft.sponsor?.trim() || undefined,
      protocolSummary: draft.summary ? { abstract: draft.summary } : undefined,
      startDate: draft.startDate ? new Date(draft.startDate).toISOString() : undefined,
      endDate: draft.endDate ? new Date(draft.endDate).toISOString() : undefined
    };
    createStudy.mutate(payload, {
      onSuccess: (created) => {
        setShowCreate(false);
        setDraft(defaultDraft);
        router.push(`/studies/${created.id}`);
      }
    });
  };

  const isCreating = createStudy.status === "pending";

  return (
    <div className="space-y-10">
      <section className="glass-panel px-6 py-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="text-[11px] uppercase tracking-[0.32em] text-white/50">Protocol Operations</p>
            <h1 className="mt-2 text-2xl font-semibold text-white">Study Orchestration & EDC Command</h1>
            <p className="mt-3 max-w-2xl text-sm text-white/60">
              Spin up multi-site trials, orchestrate visits, and keep audit-grade traceability in one unified workspace.
            </p>
          </div>
          <div className="flex gap-3">
            <button
              type="button"
              onClick={() => setShowCreate(true)}
              className="inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-brand-500 via-accent-500 to-accent-400 px-5 py-2 text-sm font-semibold text-white shadow-glow transition hover:shadow-[0_12px_35px_rgba(244,63,94,0.35)]"
            >
              <PlusIcon className="h-5 w-5" /> New Study
            </button>
            <button
              type="button"
              onClick={() => studiesQuery.refetch()}
              className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-xs font-medium text-white/70 transition hover:border-white/20 hover:text-white"
            >
              <ArrowPathIcon className={`h-4 w-4 ${studiesQuery.isFetching ? "animate-spin" : ""}`} /> Refresh
            </button>
          </div>
        </div>
        <div className="mt-6 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <MetricCard label="Studies" value={stats.total.toString()} change="Portfolio footprint" accent="brand" />
          <MetricCard label="Active" value={stats.active.toString()} change="Enrolling or running" accent="accent" />
          <MetricCard label="Subjects" value={stats.totalSubjects.toString()} change="All time" accent="sunset" />
          <MetricCard label="Active Subjects" value={stats.activeSubjects.toString()} change="Currently on protocol" accent="dawn" />
        </div>
      </section>

      <section className="glass-panel px-6 py-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-[11px] uppercase tracking-[0.32em] text-white/50">Studies</p>
            <h2 className="mt-2 text-xl font-semibold text-white">Portfolio Overview</h2>
          </div>
          <span className="text-xs text-white/40">{studiesQuery.isFetching ? "Updating…" : `${studies.length} total`}</span>
        </div>
        <div className="mt-6 overflow-x-auto">
          <table className="min-w-full divide-y divide-white/10 text-left text-sm text-white/70">
            <thead>
              <tr>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/40">Study</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/40">Phase</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/40">Status</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/40">Sites</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/40">Subjects</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/40">Updated</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/40" aria-label="actions" />
              </tr>
            </thead>
            <tbody className="divide-y divide-white/10">
              {studies.map((study) => (
                <tr key={study.id} className="hover:bg-white/5">
                  <td className="px-4 py-3">
                    <div className="font-semibold text-white">{study.name}</div>
                    <div className="text-xs uppercase tracking-[0.22em] text-white/40">{study.code}</div>
                  </td>
                  <td className="px-4 py-3 text-white/60">{study.phase ?? "—"}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold ${
                        statusColors[study.status] ?? "bg-white/10 text-white"
                      }`}
                    >
                      {study.status}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-white/60">{study.sites?.length ?? 0}</span>
                  </td>
                  <td className="px-4 py-3">
                    <div className="text-white/70">{study.activeSubjects ?? 0}<span className="text-white/30"> / {study.totalSubjects ?? 0}</span></div>
                  </td>
                  <td className="px-4 py-3 text-white/50">{new Date(study.updatedAt).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-right">
                    <button
                      type="button"
                      onClick={() => router.push(`/studies/${study.id}`)}
                      className="inline-flex items-center gap-1 rounded-full border border-white/10 px-3 py-1 text-xs font-semibold text-white/70 transition hover:border-brand-400 hover:text-white"
                    >
                      Open
                      <ArrowRightIcon className="h-3 w-3" />
                    </button>
                  </td>
                </tr>
              ))}
              {studies.length === 0 && (
                <tr>
                  <td colSpan={7} className="px-4 py-12 text-center text-sm text-white/40">
                    {studiesQuery.isFetching ? "Loading studies…" : "No studies yet. Launch your first protocol to get started."}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>

      {showCreate && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/60 px-4">
          <div className="w-full max-w-xl rounded-3xl border border-white/10 bg-surface-raised/90 p-6 shadow-2xl">
            <h3 className="text-lg font-semibold text-white">Create Study</h3>
            <p className="mt-1 text-sm text-white/50">Define the protocol shell, you can refine details later.</p>
            <div className="mt-5 grid gap-4">
              <div className="grid gap-2">
                <label className="text-xs uppercase tracking-[0.32em] text-white/40">Code *</label>
                <input
                  value={draft.code}
                  onChange={(event) => setDraft((prev) => ({ ...prev, code: event.target.value }))}
                  placeholder="SYN-001"
                  className="rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm text-white focus:border-brand-400 focus:outline-none"
                />
              </div>
              <div className="grid gap-2">
                <label className="text-xs uppercase tracking-[0.32em] text-white/40">Name *</label>
                <input
                  value={draft.name}
                  onChange={(event) => setDraft((prev) => ({ ...prev, name: event.target.value }))}
                  placeholder="Synaptica Heart Health"
                  className="rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm text-white focus:border-brand-400 focus:outline-none"
                />
              </div>
              <div className="grid gap-2">
                <label className="text-xs uppercase tracking-[0.32em] text-white/40">Phase</label>
                <select
                  value={draft.phase}
                  onChange={(event) => setDraft((prev) => ({ ...prev, phase: event.target.value }))}
                  className="rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm text-white focus:border-brand-400 focus:outline-none"
                >
                  <option value="" className="bg-surface-raised text-slate-900">Select phase</option>
                  {phases.map((phase) => (
                    <option key={phase} value={phase} className="bg-surface-raised text-slate-900">
                      {phase}
                    </option>
                  ))}
                </select>
              </div>
              <div className="grid gap-2">
                <label className="text-xs uppercase tracking-[0.32em] text-white/40">Therapeutic Area</label>
                <input
                  value={draft.therapeuticArea}
                  onChange={(event) => setDraft((prev) => ({ ...prev, therapeuticArea: event.target.value }))}
                  placeholder="Cardiology"
                  className="rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm text-white focus:border-brand-400 focus:outline-none"
                />
              </div>
              <div className="grid gap-2">
                <label className="text-xs uppercase tracking-[0.32em] text-white/40">Sponsor</label>
                <input
                  value={draft.sponsor}
                  onChange={(event) => setDraft((prev) => ({ ...prev, sponsor: event.target.value }))}
                  placeholder="Synaptica Labs"
                  className="rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm text-white focus:border-brand-400 focus:outline-none"
                />
              </div>
              <div className="grid gap-2 md:grid-cols-2 md:gap-4">
                <div className="grid gap-2">
                  <label className="text-xs uppercase tracking-[0.32em] text-white/40">Start</label>
                  <input
                    type="date"
                    value={draft.startDate ?? ""}
                    onChange={(event) => setDraft((prev) => ({ ...prev, startDate: event.target.value }))}
                    className="rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm text-white focus:border-brand-400 focus:outline-none"
                  />
                </div>
                <div className="grid gap-2">
                  <label className="text-xs uppercase tracking-[0.32em] text-white/40">End</label>
                  <input
                    type="date"
                    value={draft.endDate ?? ""}
                    onChange={(event) => setDraft((prev) => ({ ...prev, endDate: event.target.value }))}
                    className="rounded-2xl border border-white/10 bg-white/5 px-4 py-2 text-sm text-white focus:border-brand-400 focus:outline-none"
                  />
                </div>
              </div>
              <div className="grid gap-2">
                <label className="text-xs uppercase tracking-[0.32em] text-white/40">Synopsis</label>
                <textarea
                  value={draft.summary}
                  onChange={(event) => setDraft((prev) => ({ ...prev, summary: event.target.value }))}
                  placeholder="Primary objective, comparator, endpoints..."
                  className="h-32 w-full rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-white focus:border-brand-400 focus:outline-none"
                />
              </div>
            </div>
            <div className="mt-6 flex items-center justify-between">
              {createStudy.status === "success" && (
                <span className="inline-flex items-center gap-2 text-xs font-medium text-emerald-200">
                  <CheckCircleIcon className="h-4 w-4" /> Study created
                </span>
              )}
              {createStudy.status === "error" && (
                <span className="text-xs text-rose-300">{createStudy.error?.message ?? "Failed to create study"}</span>
              )}
              <div className="ml-auto flex gap-3">
                <button
                  type="button"
                  onClick={() => {
                    setShowCreate(false);
                    setDraft(defaultDraft);
                  }}
                  className="rounded-full border border-white/10 px-4 py-2 text-xs font-semibold text-white/70 transition hover:border-white/20 hover:text-white"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleCreate}
                  disabled={isCreating || !draft.code || !draft.name}
                  className="inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-brand-500 via-accent-500 to-accent-400 px-5 py-2 text-sm font-semibold text-white shadow-glow transition hover:shadow-[0_12px_35px_rgba(244,63,94,0.35)] disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {isCreating ? <ArrowPathIcon className="h-5 w-5 animate-spin" /> : <CheckCircleIcon className="h-5 w-5" />}
                  {isCreating ? "Creating" : "Launch Study"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
