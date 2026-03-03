import { memo, useRef, useEffect, useCallback, useState } from "react";
import type { LogEntry } from "@/lib/ws/protocol";

interface LogsPanelProps {
  logs: LogEntry[];
}

export const LogsPanel = memo(function LogsPanel({ logs }: LogsPanelProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  const handleScroll = useCallback(() => {
    const el = containerRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 8;
    setAutoScroll(atBottom);
  }, []);

  useEffect(() => {
    if (autoScroll && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  if (logs.length === 0) {
    return (
      <div className="rounded-lg border border-panel-border bg-panel-bg p-4 font-mono text-sm text-gray-400">
        No log entries.
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      onScroll={handleScroll}
      className="h-48 overflow-y-auto rounded-lg border border-panel-border bg-panel-bg p-4 font-mono text-sm"
    >
      {logs.map((entry, i) => (
        <div key={i} className="py-0.5">
          <span className="mr-2 text-gray-500">t{entry.tick}</span>
          <span className="text-gray-100">{entry.message}</span>
        </div>
      ))}
    </div>
  );
});
