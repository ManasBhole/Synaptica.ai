import { ReactNode } from "react";

interface MetricCardProps {
  label: string;
  value: string;
  change?: string;
  icon?: ReactNode;
  accent?: "accent" | "primary" | "emerald";
  footer?: ReactNode;
}

const gradientByAccent: Record<string, string> = {
  accent: "from-accent-400 to-primary-500",
  primary: "from-primary-500 to-primary-600",
  emerald: "from-emerald-400 to-teal-500"
};

export const MetricCard = ({ label, value, change, icon, accent = "accent", footer }: MetricCardProps) => {
  return (
    <div className="glass-panel relative overflow-hidden px-6 py-5">
      <div className={`absolute inset-x-4 top-0 h-px bg-gradient-to-r ${gradientByAccent[accent]} opacity-60`} />
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-white/50">{label}</p>
          <p className="mt-2 text-2xl font-semibold text-white">{value}</p>
          {change && <p className="mt-1 text-xs text-emerald-400">{change}</p>}
        </div>
        {icon && <div className="text-white/70">{icon}</div>}
      </div>
      {footer && <div className="mt-4 text-xs text-white/60">{footer}</div>}
    </div>
  );
};
