import { memo } from "react";
import type { TopologyNode } from "@/lib/ws/protocol";

interface TopologyMapProps {
  topology: TopologyNode[];
}

export const TopologyMap = memo(function TopologyMap({ topology }: TopologyMapProps) {
  if (topology.length === 0) return null;

  return (
    <div className="flex items-center gap-2 overflow-x-auto rounded-lg border border-panel-border bg-panel-bg p-4">
      {topology.map((node, i) => (
        <div key={node.name} className="flex items-center gap-2">
          {i > 0 && <span className="text-gray-500">&rarr;</span>}
          <div data-testid="topology-node" className="rounded border border-panel-border bg-surface-mid px-3 py-2 text-center">
            <div className="text-sm font-medium text-gray-100">{node.name}</div>
            <div className="text-xs text-gray-400">{node.type}</div>
          </div>
        </div>
      ))}
    </div>
  );
});
