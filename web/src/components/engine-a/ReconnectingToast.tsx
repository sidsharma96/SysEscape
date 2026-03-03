import { memo } from "react";
import type { ConnectionState } from "@/hooks/use-ws";

interface ReconnectingToastProps {
  connectionState: ConnectionState;
}

export const ReconnectingToast = memo(function ReconnectingToast({ connectionState }: ReconnectingToastProps) {
  if (connectionState === "connected") return null;

  const text = connectionState === "reconnecting" ? "Reconnecting..." : "Disconnected";

  return (
    <div className="fixed bottom-4 right-4 z-40 rounded-lg border border-signal-warn bg-panel-bg px-4 py-2 text-sm text-gray-100 shadow-lg">
      {text}
    </div>
  );
});
