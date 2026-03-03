import { useParams, useLocation } from "react-router-dom";
import { useEngineA } from "@/hooks/use-engine-a";
import { MetricsPanel } from "@/components/engine-a/MetricsPanel";
import { ActionBar } from "@/components/engine-a/ActionBar";
import { WinOverlay } from "@/components/engine-a/WinOverlay";
import { TopologyMap } from "@/components/engine-a/TopologyMap";
import { LogsPanel } from "@/components/engine-a/LogsPanel";
import { TimerBar } from "@/components/engine-a/TimerBar";
import { ReconnectingToast } from "@/components/engine-a/ReconnectingToast";

export function EngineAPage() {
  const { runId } = useParams<{ runId: string }>();
  const location = useLocation();
  const runToken = (location.state as { runToken?: string } | null)?.runToken;

  if (!runId || !runToken) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <p className="text-signal-crit">Session expired. Please start a new run from the room page.</p>
      </div>
    );
  }

  return <EngineAGameplay runId={runId} runToken={runToken} />;
}

function EngineAGameplay({ runId, runToken }: { runId: string; runToken: string }) {
  const { tick, metrics, won, actions, appliedActions, topology, logs, totalTicks, connectionState, dispatchAction } =
    useEngineA({ runId, runToken });

  return (
    <div className="space-y-4">
      <TimerBar tick={tick} totalTicks={totalTicks} />
      <TopologyMap topology={topology} />
      <div className="grid gap-4 lg:grid-cols-2">
        <MetricsPanel metrics={metrics} />
        <LogsPanel logs={logs} />
      </div>
      {actions.length > 0 && (
        <ActionBar actions={actions} appliedActions={appliedActions} onDispatch={dispatchAction} />
      )}
      <WinOverlay won={won} />
      <ReconnectingToast connectionState={connectionState} />
    </div>
  );
}
