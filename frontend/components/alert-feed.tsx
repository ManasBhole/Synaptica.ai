import type { AlertsResponse } from "../lib/api";

const statusColor: Record<string, string> = {
  failed: "bg-rose-500/20 text-rose-200",
  accepted: "bg-amber-500/20 text-amber-200",
  published: "bg-brand-500/20 text-brand-200"
};

export const AlertFeed = ({ alerts }: { alerts: AlertsResponse }) => {
  const { summary, items } = alerts;

  return (
    <div className="glass-panel space-y-5 px-6 py-6">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-[11px] uppercase tracking-[0.32em] text-white/50">Security posture</p>
          <h3 className="mt-2 text-lg font-semibold text-white">Alerts & De-Identification</h3>
        </div>
        <div className="flex items-center gap-3 text-xs text-white/60">
          <span className="rounded-full bg-rose-500/15 px-3 py-1 font-medium text-rose-200">Critical {summary.critical}</span>
          <span className="rounded-full bg-amber-500/15 px-3 py-1 font-medium text-amber-200">Warning {summary.warning}</span>
          <span className="rounded-full bg-brand-500/15 px-3 py-1 font-medium text-brand-200">Info {summary.info}</span>
        </div>
      </div>
      <div className="space-y-4">
        {items.map((alert) => (
          <div key={alert.id} className="rounded-3xl border border-white/5 bg-surface-raised/70 px-5 py-4 shadow-[rgba(244,63,94,0.25)_0px_12px_30px_-25px]">
            <div className="flex items-center justify-between">
              <div className="text-sm font-semibold text-white">
                {alert.source} • {alert.format}
              </div>
              <span className={`rounded-full px-3 py-1 text-xs font-medium ${statusColor[alert.status] ?? "bg-brand-500/20 text-brand-200"}`}>
                {alert.status}
              </span>
            </div>
            {alert.error && <p className="mt-2 text-xs text-rose-200/80">{alert.error}</p>}
            <p className="mt-2 text-xs text-white/45">{new Date(alert.updatedAt).toLocaleString()}</p>
          </div>
        ))}
      </div>
    </div>
  );
};
