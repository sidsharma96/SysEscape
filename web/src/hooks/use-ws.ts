import { useState, useEffect, useRef, useCallback } from "react";
import type { Envelope, HelloPayload } from "@/lib/ws/protocol";
import { PROTOCOL_VERSION } from "@/lib/ws/protocol";

export type ConnectionState = "connected" | "reconnecting" | "disconnected";

const BACKOFF_MS = [500, 1000, 2000, 4000, 8000];

export interface UseWsOptions {
  url: string;
  runId: string;
  runToken: string;
  onMessage: (msg: Envelope) => void;
}

export interface UseWsResult {
  send: (envelope: Envelope) => void;
  reconnect: (forceSnapshot?: boolean) => void;
  setLastSeq: (seq: number) => void;
  connectionState: ConnectionState;
}

export function useWs({ url, runId, runToken, onMessage }: UseWsOptions): UseWsResult {
  const [connectionState, setConnectionState] = useState<ConnectionState>("disconnected");
  const wsRef = useRef<WebSocket | null>(null);
  const backoffIdx = useRef(0);
  const lastSeqRef = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const intentionalClose = useRef(false);
  const onMessageRef = useRef(onMessage);
  const connectRef = useRef<(resumeFromSeq?: number) => void>(() => {});

  // Sync refs in an effect rather than during render
  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  const clearReconnectTimer = useCallback(() => {
    if (reconnectTimer.current !== null) {
      clearTimeout(reconnectTimer.current);
      reconnectTimer.current = null;
    }
  }, []);

  const send = useCallback((envelope: Envelope) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(envelope));
    }
  }, []);

  const setLastSeq = useCallback((seq: number) => {
    lastSeqRef.current = seq;
  }, []);

  const connect = useCallback(
    (resumeFromSeq?: number) => {
      clearReconnectTimer();
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.close();
      }

      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        const helloPayload: HelloPayload = { runToken };
        if (resumeFromSeq !== undefined && resumeFromSeq > 0) {
          helloPayload.resumeFromSeq = resumeFromSeq;
        }
        const hello: Envelope = {
          protocolVersion: PROTOCOL_VERSION,
          type: "hello",
          runId,
          payload: helloPayload,
        };
        ws.send(JSON.stringify(hello));
      };

      ws.onmessage = (ev: MessageEvent) => {
        const msg: Envelope = JSON.parse(ev.data as string);
        if (msg.type === "ping") {
          const pong: Envelope = { type: "pong" };
          ws.send(JSON.stringify(pong));
          return;
        }
        if (msg.type === "hello_ack") {
          backoffIdx.current = 0;
          setConnectionState("connected");
        }
        onMessageRef.current(msg);
      };

      ws.onclose = () => {
        if (intentionalClose.current) return;
        setConnectionState("reconnecting");
        const delay = BACKOFF_MS[Math.min(backoffIdx.current, BACKOFF_MS.length - 1)];
        backoffIdx.current++;
        reconnectTimer.current = setTimeout(() => {
          connectRef.current(lastSeqRef.current > 0 ? lastSeqRef.current : undefined);
        }, delay);
      };
    },
    [url, runId, runToken, clearReconnectTimer],
  );

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  const reconnect = useCallback(
    (forceSnapshot?: boolean) => {
      if (forceSnapshot) {
        lastSeqRef.current = 0;
        connectRef.current();
      } else {
        connectRef.current(lastSeqRef.current > 0 ? lastSeqRef.current : undefined);
      }
    },
    [],
  );

  useEffect(() => {
    intentionalClose.current = false;
    connectRef.current();
    return () => {
      intentionalClose.current = true;
      clearReconnectTimer();
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect, clearReconnectTimer]);

  return { send, reconnect, setLastSeq, connectionState };
}
