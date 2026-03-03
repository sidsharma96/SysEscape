import { useParams, useLocation } from "react-router-dom";
import { useEngineA } from "@/hooks/use-engine-a";
import { MetricsPanel } from "@/components/engine-a/MetricsPanel";
import { ActionBar } from "@/components/engine-a/ActionBar";
import { WinOverlay } from "@/components/engine-a/WinOverlay";

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
  const { tick, metrics, won, actions, appliedActions, dispatchAction } = useEngineA({ runId, runToken });

  return (
    <div className="space-y-6">
      <div className="flex items-baseline justify-between">
        <h1 className="text-2xl font-semibold">Engine A</h1>
        <span className="font-mono text-sm text-gray-400">Tick {tick}</span>
      </div>
      <MetricsPanel metrics={metrics} />
      {actions.length > 0 && (
        <ActionBar actions={actions} appliedActions={appliedActions} onDispatch={dispatchAction} />
      )}
      <WinOverlay won={won} />
    </div>
  );
}
