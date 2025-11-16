import type { AlertsResponse } from "../lib/api";

const statusColor: Record<string, string> = {
  failed: "bg-rose-100 text-rose-600",
  accepted: "bg-amber-100 text-amber-600",
  published: "bg-brand-100 text-brand-600"
};

export const AlertFeed = ({ alerts }: { alerts: AlertsResponse }) => {
  const { summary, items } = alerts;

  return (
    <div className="glass-panel space-y-5 px-6 py-6">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-[11px] uppercase tracking-[0.32em] text-neutral-400">Security posture</p>
          <h3 className="mt-2 text-lg font-semibold text-neutral-900">Alerts & De-Identification</h3>
        </div>
        <div className="flex items-center gap-3 text-xs text-neutral-500">
          <span className="rounded-full bg-rose-100 px-3 py-1 font-medium text-rose-600">Critical {summary.critical}</span>
          <span className="rounded-full bg-amber-100 px-3 py-1 font-medium text-amber-600">Warning {summary.warning}</span>
          <span className="rounded-full bg-brand-100 px-3 py-1 font-medium text-brand-600">Info {summary.info}</span>
        </div>
      </div>
      <div className="space-y-4">
        {items.map((alert) => (
          <div key={alert.id} className="rounded-3xl border border-neutral-200 bg-white px-5 py-4 shadow-card">
            <div className="flex items-center justify-between">
              <div className="text-sm font-semibold text-neutral-900">
                {alert.source} • {alert.format}
              </div>
              <span className={`rounded-full px-3 py-1 text-xs font-medium ${statusColor[alert.status] ?? "bg-neutral-100 text-neutral-600"}`}>
                {alert.status}
              </span>
            </div>
            {alert.error && <p className="mt-2 text-xs text-rose-600/80">{alert.error}</p>}
            <p className="mt-2 text-xs text-neutral-400">{new Date(alert.updatedAt).toLocaleString()}</p>
          </div>
        ))}
      </div>
    </div>
  );
};
