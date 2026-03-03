import { memo } from "react";

interface WinOverlayProps {
  won: boolean;
}

export const WinOverlay = memo(function WinOverlay({ won }: WinOverlayProps) {
  if (!won) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70">
      <div className="rounded-xl border border-signal-ok bg-panel-bg px-10 py-8 text-center">
        <h2 className="text-3xl font-bold text-signal-ok">You Win!</h2>
        <p className="mt-2 text-gray-300">All win conditions met.</p>
      </div>
    </div>
  );
});
