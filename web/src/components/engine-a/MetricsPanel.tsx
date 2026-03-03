import { memo } from "react";

interface MetricsPanelProps {
  metrics: Record<string, number>;
}

function formatValue(v: number): string {
  if (Number.isInteger(v)) return String(v);
  return parseFloat(v.toFixed(4)).toString();
}

export const MetricsPanel = memo(function MetricsPanel({ metrics }: MetricsPanelProps) {
  const entries = Object.entries(metrics);
  return (
    <div className="rounded-lg border border-panel-border bg-panel-bg p-4 font-mono">
      <table className="w-full">
        <tbody>
          {entries.map(([name, value]) => (
            <tr key={name} className="border-b border-panel-border last:border-0">
              <td className="py-1 pr-4 text-sm text-gray-400">{name}</td>
              <td className="py-1 text-right text-sm text-gray-100">{formatValue(value)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
});
