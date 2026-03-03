import { memo } from "react";

interface TimerBarProps {
  tick: number;
  totalTicks?: number;
}

export const TimerBar = memo(function TimerBar({ tick, totalTicks }: TimerBarProps) {
  if (totalTicks === undefined) {
    return (
      <div className="rounded-lg border border-panel-border bg-panel-bg px-4 py-2 font-mono text-sm text-gray-300">
        Tick {tick}
      </div>
    );
  }

  const pct = Math.min(100, (tick / totalTicks) * 100);
  const colorClass = pct >= 90 ? "bg-signal-crit" : pct > 75 ? "bg-signal-warn" : "bg-signal-ok";

  return (
    <div className="rounded-lg border border-panel-border bg-panel-bg px-4 py-2">
      <div className="mb-1 flex justify-between font-mono text-sm text-gray-300">
        <span>{tick} / {totalTicks}</span>
        <span>{Math.round(pct)}%</span>
      </div>
      <div className="h-2 w-full overflow-hidden rounded-full bg-surface-mid">
        <div
          data-testid="timer-progress"
          className={`h-full rounded-full transition-all ${colorClass}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
});
