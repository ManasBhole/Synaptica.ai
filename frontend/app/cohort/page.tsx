'use client';

import { useCallback, useMemo, useState } from "react";
import { CheckCircleIcon, ExclamationTriangleIcon, ArrowPathIcon } from "@heroicons/react/24/solid";
import { MetricCard } from "../../components/metric-card";
import { useCohortQuery, useCohortVerify } from "../../hooks/useCohort";
import type { CohortQueryPayload } from "../../lib/api";

const defaultDSL = `select patient_id, resource_type, concept, value, timestamp
from lakehouse
where resource_type = 'observation' and concept = 'blood_pressure_systolic' and value > 140
limit 200`;

const availableFields = [
  { key: "patient_id", label: "Patient ID" },
  { key: "master_id", label: "Master ID" },
  { key: "resource_type", label: "Resource Type" },
  { key: "concept", label: "Concept" },
  { key: "unit", label: "Unit" },
  { key: "value", label: "Value" },
  { key: "timestamp", label: "Timestamp" },
  { key: "code_loinc", label: "LOINC" },
  { key: "code_snomed", label: "SNOMED" }
];

const templates: Array<{ label: string; dsl: string }> = [
  {
    label: "High BMI (>= 30)",
    dsl: `select patient_id, concept, value, timestamp
from lakehouse
where concept = 'body_mass_index' and value >= 30
limit 150`
  },
  {
    label: "Recent HbA1c",
    dsl: `select patient_id, value, timestamp
from lakehouse
where concept = 'hba1c' and timestamp >= '2024-01-01'
limit 120`
  },
  {
    label: "Cardiology Encounters",
    dsl: `select patient_id, resource_type, timestamp
from lakehouse
where resource_type = 'encounter' and concept = 'cardiology'
limit 200`
  }
];

const formatValue = (value: unknown) => {
  if (value === null || value === undefined) return "—";
  if (typeof value === "string") return value;
  if (typeof value === "number") return Number.isInteger(value) ? value.toString() : value.toFixed(3);
  if (value instanceof Date) return value.toISOString();
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
};

const escapeCsvValue = (value: unknown) => {
  if (value === null || value === undefined) return "";
  const stringValue = typeof value === "string" ? value : formatValue(value);
  if (/[",\n]/.test(stringValue)) {
    return `"${stringValue.replace(/"/g, '""')}"`;
  }
  return stringValue;
};

const formatDuration = (value: unknown) => {
  if (typeof value === "number") {
    const ms = value / 1_000_000;
    if (ms >= 1000) {
      return `${(ms / 1000).toFixed(2)} s`;
    }
    return `${ms.toFixed(1)} ms`;
  }
  if (typeof value === "string" && value.trim() !== "") {
    return value;
  }
  return "—";
};

export default function CohortPage() {
  const [dsl, setDsl] = useState(defaultDSL);
  const [description, setDescription] = useState("");
  const [limit, setLimit] = useState(200);
  const [fields, setFields] = useState<string[]>(["patient_id", "resource_type", "concept", "value", "timestamp"]);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(25);

  const verify = useCohortVerify();
  const query = useCohortQuery();

  const records = useMemo(() => {
    const raw = query.data?.metadata?.records;
    if (Array.isArray(raw)) {
      return raw as Array<Record<string, unknown>>;
    }
    return [];
  }, [query.data]);

  const columns = useMemo(() => {
    const ordered = [...fields];
    const seen = new Set(ordered);
    records.forEach((row) => {
      Object.keys(row).forEach((key) => {
        if (!seen.has(key)) {
          ordered.push(key);
          seen.add(key);
        }
      });
    });
    return ordered;
  }, [fields, records]);

  const totalRecords = records.length;
  const pageSizeSafe = Math.max(pageSize, 1);
  const pageCount = Math.max(1, Math.ceil(totalRecords / pageSizeSafe));
  const safePage = Math.min(page, pageCount - 1);
  const pageStart = safePage * pageSizeSafe;
  const pageEnd = pageStart + pageSizeSafe;
  const pagedRecords = records.slice(pageStart, pageEnd);
  const showingStart = totalRecords === 0 ? 0 : pageStart + 1;
  const showingEnd = Math.min(totalRecords, pageEnd);
  const exportDisabled = totalRecords === 0;
  const cacheHit = query.data?.metadata?.cacheHit === true;
  const tenant = typeof query.data?.metadata?.tenant === "string" ? (query.data?.metadata?.tenant as string) : undefined;

  const handleToggleField = (key: string) => {
    setFields((prev) => {
      if (prev.includes(key)) {
        return prev.filter((item) => item !== key);
      }
      return [...prev, key];
    });
  };

  const handleRunQuery = () => {
    const payload: CohortQueryPayload = {
      dsl,
      description: description || undefined,
      limit,
      fields: fields.length > 0 ? fields : undefined
    };
    query.mutate(payload);
    setPage(0);
  };

  const handleVerify = () => {
    verify.mutate(dsl);
  };

  const handleExport = useCallback(() => {
    if (records.length === 0) {
      return;
    }
    const header = columns;
    const csvRows = [header.join(",")];
    records.forEach((row) => {
      const line = header.map((column) => escapeCsvValue(row[column])).join(",");
      csvRows.push(line);
    });
    const blob = new Blob([csvRows.join("\n")], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `${query.data?.cohortId ?? "cohort"}-${Date.now()}.csv`;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(url);
  }, [records, columns, query.data?.cohortId]);

  const handlePrevPage = useCallback(() => {
    setPage((prev) => Math.max(0, prev - 1));
  }, []);

  const handleNextPage = useCallback(() => {
    setPage((prev) => Math.min(pageCount - 1, prev + 1));
  }, [pageCount]);

  const handlePageSizeChange = (value: number) => {
    setPageSize(value);
    setPage(0);
  };

  const queryTimeDisplay = formatDuration(query.data?.queryTime);
  const cohortSize = query.data?.count ?? 0;
  const uniquePatients = query.data?.patientIds.length ?? 0;

  const patientPreview = query.data?.patientIds.slice(0, 12) ?? [];
  const sliceCount = Array.isArray(query.data?.metadata?.slices) ? (query.data?.metadata?.slices as unknown[]).length : undefined;

  const isVerifying = verify.status === "pending";
  const verifyState = verify.status;

  return (
    <div className="space-y-10">
      <section className="glass-panel px-6 py-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="text-[11px] uppercase tracking-[0.32em] text-white/50">Cohort Intelligence</p>
            <h1 className="mt-2 text-2xl font-semibold text-white">Design, validate, and materialize cohorts in seconds</h1>
            <p className="mt-3 max-w-2xl text-sm text-white/60">
              Use Synaptica&apos;s DSL to slice the longitudinal lakehouse, verify privacy-safe filters, and export curated patient populations for AI training
              or payer reporting.
            </p>
          </div>
          <div className="flex gap-3">
            <button
              type="button"
              onClick={handleVerify}
              disabled={isVerifying}
              className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-xs font-medium text-white/80 transition hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {isVerifying ? <ArrowPathIcon className="h-4 w-4 animate-spin" /> : <CheckCircleIcon className="h-4 w-4 text-brand-300" />}
              {isVerifying ? "Verifying" : "Verify DSL"}
            </button>
            <button
              type="button"
              onClick={handleRunQuery}
              disabled={query.status === "pending"}
              className="inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-brand-500 via-accent-500 to-accent-400 px-5 py-2 text-sm font-semibold text-white shadow-glow transition hover:shadow-[0_12px_35px_rgba(244,63,94,0.35)] disabled:cursor-not-allowed disabled:opacity-70"
            >
              {query.status === "pending" ? <ArrowPathIcon className="h-5 w-5 animate-spin" /> : <CheckCircleIcon className="h-5 w-5" />}
              {query.status === "pending" ? "Running" : "Run Cohort"}
            </button>
          </div>
        </div>
        {verifyState === "success" && (
          <div className="mt-4 inline-flex items-center gap-2 rounded-full bg-brand-500/15 px-4 py-1 text-xs font-medium text-brand-200">
            <CheckCircleIcon className="h-4 w-4" /> Cohort DSL verified
          </div>
        )}
        {verifyState === "error" && (
          <div className="mt-4 inline-flex items-center gap-2 rounded-full bg-rose-500/20 px-4 py-1 text-xs font-medium text-rose-200">
            <ExclamationTriangleIcon className="h-4 w-4" /> {verify.error?.message ?? "Verification failed"}
          </div>
        )}
      </section>

      <section className="grid gap-6 lg:grid-cols-[1.4fr_1fr]">
        <div className="glass-panel px-6 py-6">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-white">Cohort DSL</h2>
              <p className="text-xs text-white/50">Tune select fields, predicates, and limit guards. All queries auto-enforce RLS and tokenization policies.</p>
            </div>
            <label className="flex items-center gap-2 text-xs text-white/60">
              Limit
              <input
                type="number"
                min={25}
                max={5000}
                value={limit}
                onChange={(event) => setLimit(Number.parseInt(event.target.value, 10) || 0)}
                className="w-24 rounded-xl border border-white/10 bg-white/5 px-3 py-1 text-right text-sm text-white focus:border-brand-400 focus:outline-none"
              />
            </label>
          </div>
          <textarea
            value={dsl}
            onChange={(event) => setDsl(event.target.value)}
            spellCheck={false}
            className="mt-4 h-64 w-full resize-none rounded-2xl border border-white/10 bg-black/40 px-4 py-4 font-mono text-sm text-white/80 shadow-inner focus:border-brand-400 focus:outline-none"
          />
          <div className="mt-4 flex flex-wrap gap-3 text-xs text-white/60">
            {availableFields.map((field) => {
              const active = fields.includes(field.key);
              return (
                <button
                  type="button"
                  key={field.key}
                  onClick={() => handleToggleField(field.key)}
                  className={`rounded-full border px-3 py-1 transition ${
                    active ? "border-brand-400/60 bg-brand-500/20 text-white" : "border-white/10 bg-white/5 hover:border-white/20"
                  }`}
                >
                  {field.label}
                </button>
              );
            })}
          </div>
          <div className="mt-6">
            <p className="text-[11px] uppercase tracking-[0.32em] text-white/40">Templates</p>
            <div className="mt-3 grid gap-3 md:grid-cols-3">
              {templates.map((template) => (
                <button
                  type="button"
                  key={template.label}
                  onClick={() => setDsl(template.dsl)}
                  className="rounded-2xl border border-white/10 bg-white/5 p-4 text-left text-xs text-white/70 transition hover:border-brand-400/60 hover:bg-brand-500/10"
                >
                  <p className="text-sm font-semibold text-white">{template.label}</p>
                  <p className="mt-2 line-clamp-4 whitespace-pre-line text-[11px] text-white/50">{template.dsl}</p>
                </button>
              ))}
            </div>
          </div>
        </div>
        <div className="space-y-4">
          <div className="glass-panel px-6 py-6">
            <label className="text-xs uppercase tracking-[0.32em] text-white/40">Description</label>
            <textarea
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="Investor-ready narrative for this cohort run"
              className="mt-2 h-28 w-full resize-none rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-white/70 focus:border-brand-400 focus:outline-none"
            />
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <MetricCard label="Cohort Size" value={cohortSize ? cohortSize.toLocaleString() : "—"} change="Distinct facts returned" accent="brand" />
            <MetricCard label="Unique Patients" value={uniquePatients ? uniquePatients.toLocaleString() : "—"} change="Master IDs deduped" accent="accent" />
            <MetricCard
              label="Query Time"
              value={queryTimeDisplay}
              change={cacheHit ? "Served from cache" : "Live execution"}
              accent="sunset"
            />
            <MetricCard
              label="Preview IDs"
              value={patientPreview.length ? patientPreview.slice(0, 3).join(", ") : "—"}
              change={sliceCount ? `${sliceCount} slices analysed` : tenant ? `Tenant • ${tenant}` : "Live sample"}
            />
          </div>
        </div>
      </section>

      <section className="glass-panel px-6 py-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <p className="text-[11px] uppercase tracking-[0.32em] text-white/50">Record Sample</p>
            <h2 className="mt-2 text-xl font-semibold text-white">Materialized rows (top 200)</h2>
            <p className="mt-1 text-xs text-white/40">
              {cacheHit ? "Cached" : "Fresh"} • Showing {showingStart ? `${showingStart}–${showingEnd}` : "0"} of {totalRecords}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-3 text-xs text-white/60">
            <label className="flex items-center gap-2">
              Rows/page
              <select
                value={pageSize}
                onChange={(event) => handlePageSizeChange(Number(event.target.value))}
                className="rounded-xl border border-white/10 bg-white/5 px-3 py-1 text-white focus:border-brand-400 focus:outline-none"
              >
                {[10, 25, 50, 100].map((size) => (
                  <option key={size} value={size} className="bg-surface-raised text-slate-900">
                    {size}
                  </option>
                ))}
              </select>
            </label>
            <div className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-white/70">
              <button
                type="button"
                onClick={handlePrevPage}
                disabled={safePage === 0}
                className="rounded-full px-2 text-white/70 transition hover:text-white disabled:cursor-not-allowed disabled:opacity-40"
              >
                Prev
              </button>
              <span>
                Page {Math.min(safePage + 1, pageCount)} / {pageCount}
              </span>
              <button
                type="button"
                onClick={handleNextPage}
                disabled={safePage >= pageCount - 1 || totalRecords === 0}
                className="rounded-full px-2 text-white/70 transition hover:text-white disabled:cursor-not-allowed disabled:opacity-40"
              >
                Next
              </button>
            </div>
            <button
              type="button"
              onClick={handleExport}
              disabled={exportDisabled}
              className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-1 font-medium text-white/70 transition hover:border-brand-400 hover:text-white disabled:cursor-not-allowed disabled:opacity-40"
            >
              Export CSV
            </button>
            {query.status === "pending" && <ArrowPathIcon className="h-5 w-5 animate-spin text-brand-300" />}
          </div>
        </div>
        <div className="mt-4 overflow-x-auto">
          <table className="min-w-full divide-y divide-white/10 text-left text-sm text-white/70">
            <thead>
              <tr>
                {columns.map((column) => (
                  <th key={column} className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">
                    {column}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-white/10">
              {pagedRecords.map((row, rowIndex) => {
                const candidateId = row["patient_id"];
                const rowKey = typeof candidateId === "string" ? `${candidateId}-${pageStart + rowIndex}` : `row-${pageStart + rowIndex}`;
                return (
                  <tr key={rowKey} className="hover:bg-white/5">
                    {columns.map((column) => (
                      <td key={column} className="px-4 py-3 font-mono text-[13px] text-white/70">
                        {formatValue(row[column])}
                      </td>
                    ))}
                  </tr>
                );
              })}
              {totalRecords === 0 && (
                <tr>
                  <td colSpan={Math.max(columns.length, 1)} className="px-4 py-12 text-center text-sm text-white/40">
                    {query.status === "success"
                      ? "No records returned for this cohort. Adjust filters or expand the limit."
                      : "Run the cohort to preview records."}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
