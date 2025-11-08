"use client";

import { BoltIcon } from "@heroicons/react/24/solid";
import { usePathname } from "next/navigation";
import Link from "next/link";

const titles: Record<string, string> = {
  "/": "Unified Intelligence Snapshot",
  "/cohort": "Cohort Analytics Workbench",
  "/privacy": "Privacy & DLP Posture",
  "/pipeline": "Data Pipelines & Observability",
  "/training": "AI Training Orchestration",
  "/predictions": "Realtime Predictions"
};

export const Topbar = () => {
  const pathname = usePathname();
  const title = titles[pathname] ?? "Synaptica Platform";

  return (
    <header className="flex items-center justify-between border-b border-white/5 bg-surface-raised/60 px-8 py-6 backdrop-blur-xl shadow-[rgba(15,23,42,0.35)_0px_10px_35px_-25px]">
      <div>
        <nav aria-label="Breadcrumb" className="mb-1 text-xs uppercase tracking-[0.35em] text-white/40">
          <Link href="/" className="hover:text-white/70">Synaptica</Link>
          <span className="mx-2">/</span>
          <span>{title}</span>
        </nav>
        <h1 className="text-2xl font-semibold text-white">{title}</h1>
      </div>
      <Link
        href="/training"
        className="inline-flex items-center gap-2 rounded-full bg-gradient-to-tr from-brand-500 via-accent-500 to-accent-400 px-5 py-2 text-sm font-medium text-white shadow-glow transition hover:shadow-[0_12px_35px_rgba(244,63,94,0.35)]"
      >
        <BoltIcon className="h-4 w-4" />
        Launch Autopilot Training
      </Link>
    </header>
  );
};
