"use client";

import type { ComponentProps, ElementType } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { Route } from "next";
import { ChartBarIcon, Cog6ToothIcon, CpuChipIcon, Squares2X2Icon, UserGroupIcon, ShieldCheckIcon, ClipboardDocumentListIcon } from "@heroicons/react/24/outline";

type NavLink = {
  label: string;
  href: Route;
  icon: ElementType<ComponentProps<typeof Squares2X2Icon>>;
};

const links: NavLink[] = [
  { label: "Overview", href: "/", icon: Squares2X2Icon },
  { label: "Studies", href: "/studies", icon: ClipboardDocumentListIcon },
  { label: "Cohorts", href: "/cohort", icon: UserGroupIcon },
  { label: "Privacy", href: "/privacy", icon: ShieldCheckIcon },
  { label: "Pipelines", href: "/pipeline", icon: Cog6ToothIcon },
  { label: "Training", href: "/training", icon: CpuChipIcon },
  { label: "Predictions", href: "/predictions", icon: ChartBarIcon }
];

export const Sidebar = () => {
  const pathname = usePathname();

  return (
    <aside className="hidden w-72 flex-col border-r border-neutral-200 bg-white/95 backdrop-blur lg:flex">
      <div className="flex items-center gap-3 px-7 pb-6 pt-8">
        <div className="inline-flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-brand-500 to-brand-400 font-semibold text-white shadow-glow">
          Σ
        </div>
        <div>
          <p className="text-[11px] uppercase tracking-[0.4em] text-neutral-400">Synaptica</p>
          <p className="text-lg font-semibold text-neutral-900">Insight Labs</p>
        </div>
      </div>
      <nav className="flex-1 space-y-1 px-4">
        {links.map(({ label, href, icon: Icon }) => {
          const active = pathname === href;
          return (
            <Link
              key={href}
              href={href}
              className={`flex items-center gap-3 rounded-2xl px-4 py-3 text-sm font-medium transition
                ${
                  active
                    ? "bg-brand-50 text-brand-600 shadow-inner"
                    : "text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900"
                }`}
            >
              <Icon className={`h-5 w-5 ${active ? "text-brand-500" : ""}`} />
              <span>{label}</span>
            </Link>
          );
        })}
      </nav>
      <div className="px-7 pb-8 pt-6 text-xs text-neutral-400">
        <p className="font-semibold text-neutral-600">Live Environments</p>
        <p>• Production · us-east-1</p>
        <p>• Sandbox · eu-west-2</p>
      </div>
    </aside>
  );
};
