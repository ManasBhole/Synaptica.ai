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
  accent: "from-accent-500 to-accent-400",
  brand: "from-brand-500 to-brand-400",
  sunset: "from-amber-300 to-orange-500"
};

export const MetricCard = ({ label, value, change, icon, accent = "accent", footer }: MetricCardProps) => {
  return (
    <div className="glass-panel card-highlight relative overflow-hidden px-6 py-5">
      <div className={`absolute inset-x-4 top-0 h-[2px] bg-gradient-to-r ${gradientByAccent[accent]} opacity-80`} />
      <div className="flex items-center justify-between">
        <div>
          <p className="text-[11px] uppercase tracking-[0.32em] text-white/50">{label}</p>
          <p className="mt-3 text-3xl font-semibold text-white">{value}</p>
          {change && <p className="mt-2 text-xs text-brand-200/90">{change}</p>}
        </div>
        {icon && <div className="text-white/70">{icon}</div>}
      </div>
      {footer && <div className="mt-4 text-xs text-white/60">{footer}</div>}
    </div>
  );
};
