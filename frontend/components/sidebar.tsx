"use client";

import type { ComponentProps, ElementType } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { Route } from "next";
import { ChartBarIcon, Cog6ToothIcon, CpuChipIcon, Squares2X2Icon } from "@heroicons/react/24/outline";

type NavLink = {
  label: string;
  href: Route;
  icon: ElementType<ComponentProps<typeof Squares2X2Icon>>;
};

const links: NavLink[] = [
  { label: "Overview", href: "/", icon: Squares2X2Icon },
  { label: "Pipelines", href: "/pipeline", icon: Cog6ToothIcon },
  { label: "Training", href: "/training", icon: CpuChipIcon },
  { label: "Predictions", href: "/predictions", icon: ChartBarIcon }
];

export const Sidebar = () => {
  const pathname = usePathname();

  return (
    <aside className="hidden w-72 flex-col border-r border-white/10 bg-white/5 backdrop-blur-xl lg:flex">
      <div className="flex items-center gap-3 px-8 pb-6 pt-8">
        <div className="inline-flex h-10 w-10 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-500 to-accent-500 font-semibold text-white shadow-floating">
          Σ
        </div>
        <div>
          <p className="text-sm uppercase tracking-[0.2em] text-white/40">Synaptica</p>
          <p className="text-lg font-semibold text-white">Control Center</p>
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
                ${active ? "bg-primary-500/20 text-white" : "text-white/60 hover:bg-white/10 hover:text-white"}`}
            >
              <Icon className="h-5 w-5" />
              <span>{label}</span>
            </Link>
          );
        })}
      </nav>
      <div className="px-6 pb-8 pt-6 text-xs text-white/50">
        <p className="font-semibold text-white/70">Live Environments</p>
        <p>• Production US-East</p>
        <p>• Sandbox EU-West</p>
      </div>
    </aside>
  );
};
