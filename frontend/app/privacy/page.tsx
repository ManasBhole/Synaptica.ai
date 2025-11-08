'use client';

import { Suspense, useMemo } from "react";
import { useDLPStats } from "../../hooks/usePrivacy";
import { MetricCard } from "../../components/metric-card";

const PrivacyContent = () => {
  const { data } = useDLPStats();
  const stats = data ?? {
    todayFailed: 0,
    todayAccepted: 0,
    tokenVaultSize: 0,
    topReasons: [],
    recentIncidents: []
  };

  const totalToday = stats.todayFailed + stats.todayAccepted;
  const failureRate = totalToday === 0 ? 0 : (stats.todayFailed / totalToday) * 100;

  const reasons = useMemo(() => stats.topReasons ?? [], [stats.topReasons]);
  const incidents = stats.recentIncidents ?? [];

  return (
    <div className="space-y-8">
      <section className="grid gap-6 md:grid-cols-4">
        <MetricCard
          label="PHI Blocks Today"
          value={stats.todayFailed.toLocaleString()}
          change={`${failureRate.toFixed(1)}% rejection rate`}
          accent="accent"
        />
        <MetricCard
          label="Accepted Records"
          value={stats.todayAccepted.toLocaleString()}
          change="Post-sanitization"
          accent="brand"
        />
        <MetricCard
          label="Token Vault Size"
          value={stats.tokenVaultSize.toLocaleString()}
          change="Active reversible tokens"
          accent="sunset"
        />
        <MetricCard
          label="Incidents (24h)"
          value={incidents.length.toLocaleString()}
          change="Triggered reviews"
        />
      </section>

      <section className="glass-panel px-8 py-6">
        <h2 className="text-xl font-semibold text-white">Top Rejection Reasons</h2>
        <p className="mt-2 text-sm text-white/60">
          Insight into why upstream payloads are failing DLP screening. Engineering can tune regexes or request source fixes.
        </p>
        <div className="mt-6 grid gap-4 md:grid-cols-2">
          {reasons.length === 0 && (
            <div className="rounded-2xl border border-white/10 bg-white/5 px-6 py-6 text-sm text-white/50">
              No DLP rejections recorded yet today.
            </div>
          )}
          {reasons.map((reason) => (
            <div
              key={reason.reason}
              className="rounded-2xl border border-white/10 bg-white/5 px-6 py-6 text-sm text-white/70 shadow-[rgba(244,63,94,0.25)_0px_16px_35px_-30px]"
            >
              <p className="text-xs uppercase tracking-[0.28em] text-rose-200/70">{reason.count} incidents</p>
              <p className="mt-2 text-base font-semibold text-white">{reason.reason}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="glass-panel px-8 py-6">
        <h3 className="text-lg font-semibold text-white">Recent PHI Incidents</h3>
        <p className="mt-2 text-sm text-white/60">
          Detailed ledger of rejected payloads. Use this to coordinate source remediation or adjust anonymization rules.
        </p>
        <div className="mt-6 overflow-x-auto">
          <table className="min-w-full divide-y divide-white/10 text-left text-sm text-white/70">
            <thead>
              <tr>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Source</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Format</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Status</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Error</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Retries</th>
                <th className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-white/50">Updated</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/10">
              {incidents.map((incident) => (
                <tr key={incident.id} className="hover:bg-white/5">
                  <td className="px-4 py-3 font-medium text-white">{incident.source}</td>
                  <td className="px-4 py-3 text-white/60">{incident.format.toUpperCase()}</td>
                  <td className="px-4 py-3 text-white/60">{incident.status}</td>
                  <td className="px-4 py-3 text-xs text-white/60">{incident.error || "Unknown"}</td>
                  <td className="px-4 py-3 text-white/60">{incident.retryCount}</td>
                  <td className="px-4 py-3 text-white/60">{new Date(incident.updatedAt).toLocaleTimeString()}</td>
                </tr>
              ))}
              {incidents.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-sm text-white/40">
                    Zero PHI incidents logged recently. Keep streaming!
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
};

export default function PrivacyPage() {
  return (
    <Suspense fallback={<div className="text-white/60">Loading privacy telemetry…</div>}>
      <PrivacyContent />
    </Suspense>
  );
}
