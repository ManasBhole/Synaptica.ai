"use client";

import { BoltIcon } from "@heroicons/react/24/solid";
import { usePathname } from "next/navigation";
import Link from "next/link";

const titles: Record<string, string> = {
  "/": "Unified Intelligence Snapshot",
  "/pipeline": "Data Pipelines & Observability",
  "/training": "AI Training Orchestration",
  "/predictions": "Realtime Predictions"
};

export const Topbar = () => {
  const pathname = usePathname();
  const title = titles[pathname] ?? "Synaptica Platform";

  return (
    <header className="flex items-center justify-between border-b border-white/10 px-8 py-6">
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
        className="inline-flex items-center gap-2 rounded-full bg-gradient-to-tr from-accent-500 via-primary-500 to-primary-600 px-5 py-2 text-sm font-medium text-white shadow-floating transition hover:translate-y-0.5"
      >
        <BoltIcon className="h-4 w-4" />
        Launch Autopilot Training
      </Link>
    </header>
  );
};
