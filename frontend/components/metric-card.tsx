import { ReactNode } from "react";

interface MetricCardProps {
  label: string;
  value: string;
  change?: string;
  icon?: ReactNode;
  accent?: "accent" | "brand" | "sunset";
  footer?: ReactNode;
}

const gradientByAccent: Record<string, string> = {
  accent: "from-accent-400 to-accent-500",
  brand: "from-brand-400 to-brand-500",
  sunset: "from-rose-300 to-orange-400",
  dawn: "from-sky-200 to-indigo-300"
};

export const MetricCard = ({ label, value, change, icon, accent = "accent", footer }: MetricCardProps) => {
  return (
    <div className="glass-panel relative overflow-hidden px-6 py-5">
      <div className={`absolute inset-x-6 top-0 h-[2px] rounded-full bg-gradient-to-r ${gradientByAccent[accent]} opacity-70`} />
      <div className="flex items-center justify-between">
        <div>
          <p className="text-[11px] uppercase tracking-[0.32em] text-neutral-400">{label}</p>
          <p className="mt-3 text-3xl font-semibold text-neutral-900">{value}</p>
          {change && <p className="mt-2 text-xs text-brand-600">{change}</p>}
        </div>
        {icon && <div className="text-neutral-300">{icon}</div>}
      </div>
      {footer && <div className="mt-4 text-xs text-neutral-500">{footer}</div>}
    </div>
  );
};
