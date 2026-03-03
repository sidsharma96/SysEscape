export const PROTOCOL_VERSION = 1;

export type MessageType =
  | "hello"
  | "hello_ack"
  | "snapshot"
  | "delta"
  | "apply_action"
  | "action_accepted"
  | "win_update"
  | "ping"
  | "pong";

export interface Envelope {
  protocolVersion?: number;
  type: MessageType;
  runId?: string;
  seq?: number;
  payload?: unknown;
}

export interface HelloPayload {
  runToken: string;
  resumeFromSeq?: number;
}

export interface HelloAckPayload {
  snapshotRequired: boolean;
}

export interface SnapshotPayload {
  tick: number;
  won: boolean;
  metrics: Record<string, number>;
  actions?: string[];
}

export interface ApplyActionPayload {
  actionKey: string;
  clientRequestId: string;
  expectedSeq: number;
}

export interface ActionAcceptedPayload {
  actionKey: string;
  seq: number;
}

export interface WinUpdatePayload {
  won: boolean;
}
