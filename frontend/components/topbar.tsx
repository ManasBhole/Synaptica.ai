"use client";

import { BoltIcon } from "@heroicons/react/24/solid";
import { usePathname } from "next/navigation";
import Link from "next/link";

const titles: Record<string, string> = {
  "/": "Unified Intelligence Snapshot",
  "/studies": "Study Orchestration & EDC",
  "/cohort": "Cohort Analytics Workbench",
  "/privacy": "Privacy & DLP Posture",
  "/pipeline": "Data Pipelines & Observability",
  "/training": "AI Training Orchestration",
  "/predictions": "Realtime Predictions"
};

export const Topbar = () => {
  const pathname = usePathname();
  const normalized = pathname.startsWith("/studies/") ? "/studies" : pathname;
  const title = titles[normalized] ?? "Synaptica Platform";

  return (
    <header className="flex items-center justify-between border-b border-neutral-200 bg-white/90 px-10 py-6 backdrop-blur">
      <div>
        <nav aria-label="Breadcrumb" className="mb-1 text-xs uppercase tracking-[0.35em] text-neutral-400">
          <Link href="/" className="hover:text-neutral-700">
            Synaptica
          </Link>
          <span className="mx-2">/</span>
          <span>{title}</span>
        </nav>
        <h1 className="text-2xl font-semibold text-neutral-900">{title}</h1>
      </div>
      <Link
        href="/training"
        className="inline-flex items-center gap-2 rounded-full bg-gradient-to-tr from-brand-500 via-brand-400 to-accent-400 px-5 py-2 text-sm font-medium text-white shadow-glow transition hover:opacity-95"
      >
        <BoltIcon className="h-4 w-4" />
        Launch Autopilot Training
      </Link>
    </header>
  );
};
