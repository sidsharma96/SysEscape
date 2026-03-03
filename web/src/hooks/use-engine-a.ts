import { useState, useCallback, useRef, useMemo } from "react";
import { useWs } from "@/hooks/use-ws";
import type { Envelope, SnapshotPayload, WinUpdatePayload, ActionAcceptedPayload, TopologyNode, LogEntry } from "@/lib/ws/protocol";
import type { ConnectionState } from "@/hooks/use-ws";
import { PROTOCOL_VERSION } from "@/lib/ws/protocol";
import { newRequestId } from "@/lib/idempotency";

export interface UseEngineAOptions {
  runId: string;
  runToken: string;
}

export interface EngineAState {
  tick: number;
  metrics: Record<string, number>;
  won: boolean;
  actions: string[];
  appliedActions: string[];
  topology: TopologyNode[];
  logs: LogEntry[];
  totalTicks: number | undefined;
  connectionState: ConnectionState;
  dispatchAction: (actionKey: string) => void;
}

export function useEngineA({ runId, runToken }: UseEngineAOptions): EngineAState {
  const [tick, setTick] = useState(0);
  const [metrics, setMetrics] = useState<Record<string, number>>({});
  const [won, setWon] = useState(false);
  const [actions, setActions] = useState<string[]>([]);
  const [appliedActions, setAppliedActions] = useState<string[]>([]);
  const [topology, setTopology] = useState<TopologyNode[]>([]);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [totalTicks, setTotalTicks] = useState<number | undefined>(undefined);
  const lastSeqRef = useRef(0);

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${protocol}//${window.location.host}/ws/engineA/${runId}`;

  const onMessage = useCallback(
    (msg: Envelope) => {
      switch (msg.type) {
        case "hello_ack":
          break;
        case "snapshot":
        case "delta": {
          const seq = msg.seq;
          if (seq === undefined) break;
          if (msg.type === "delta" && seq !== lastSeqRef.current + 1) {
            wsResult.reconnect(true);
            return;
          }
          const payload = msg.payload as SnapshotPayload;
          lastSeqRef.current = seq;
          wsResult.setLastSeq(seq);
          setTick(payload.tick);
          setMetrics(payload.metrics);
          if (payload.actions) setActions(payload.actions);
          if (payload.won) setWon(true);
          if (payload.topology) setTopology(payload.topology);
          if (payload.totalTicks !== undefined) setTotalTicks(payload.totalTicks);
          if (payload.logs) setLogs((prev) => [...prev, ...payload.logs!]);
          break;
        }
        case "win_update": {
          const payload = msg.payload as WinUpdatePayload;
          if (payload.won) setWon(true);
          break;
        }
        case "action_accepted": {
          const payload = msg.payload as ActionAcceptedPayload;
          setAppliedActions((prev) =>
            prev.includes(payload.actionKey) ? prev : [...prev, payload.actionKey],
          );
          break;
        }
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- wsResult used via ref pattern
    [],
  );

  const wsResult = useWs({ url: wsUrl, runId, runToken, onMessage });

  const dispatchAction = useCallback(
    (actionKey: string) => {
      wsResult.send({
        protocolVersion: PROTOCOL_VERSION,
        type: "apply_action",
        runId,
        payload: {
          actionKey,
          clientRequestId: newRequestId(),
          expectedSeq: lastSeqRef.current,
        },
      });
    },
    [wsResult, runId],
  );

  const { connectionState } = wsResult;

  return useMemo(
    () => ({ tick, metrics, won, actions, appliedActions, topology, logs, totalTicks, connectionState, dispatchAction }),
    [tick, metrics, won, actions, appliedActions, topology, logs, totalTicks, connectionState, dispatchAction],
  );
}
