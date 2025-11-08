interface EventItem {
  id: string;
  title: string;
  description: string;
  timestamp: string;
  status: "success" | "warning" | "error" | "info";
}

const statusStyles: Record<EventItem["status"], string> = {
  success: "bg-emerald-500",
  warning: "bg-amber-400",
  error: "bg-rose-500",
  info: "bg-brand-500"
};

export const EventTimeline = ({ events }: { events: EventItem[] }) => {
  return (
    <div className="glass-panel space-y-6 px-6 py-6">
      <div>
        <p className="text-[11px] uppercase tracking-[0.32em] text-white/50">Latest activity</p>
        <h2 className="mt-2 text-lg font-semibold text-white">Event Stream</h2>
      </div>
      <ol className="relative space-y-5 before:absolute before:left-2 before:top-3 before:h-[calc(100%-12px)] before:w-px before:bg-brand-500/20">
        {events.map((event) => (
          <li key={event.id} className="relative pl-8">
            <span className={`absolute left-0 top-1.5 h-3.5 w-3.5 rounded-full ${statusStyles[event.status]} shadow-glow`} />
            <p className="text-sm font-medium text-white">{event.title}</p>
            <p className="text-xs text-white/50">{event.description}</p>
            <p className="mt-1 text-xs text-white/40">{event.timestamp}</p>
          </li>
        ))}
      </ol>
    </div>
  );
};
