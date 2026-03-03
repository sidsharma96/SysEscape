import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { Mock } from "vitest";

vi.mock("@/hooks/use-ws", () => ({
  useWs: vi.fn(),
}));

import { useWs } from "@/hooks/use-ws";
import { useEngineA } from "../use-engine-a";

const mockUseWs = useWs as Mock;

describe("useEngineA", () => {
  let capturedOnMessage: (msg: Record<string, unknown>) => void;
  const mockSend = vi.fn();
  const mockReconnect = vi.fn();
  const mockSetLastSeq = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseWs.mockImplementation((opts: { onMessage: (msg: Record<string, unknown>) => void }) => {
      capturedOnMessage = opts.onMessage;
      return { send: mockSend, reconnect: mockReconnect, setLastSeq: mockSetLastSeq, connectionState: "connected" };
    });
  });

  const defaultProps = {
    runId: "run-1",
    runToken: "tok-1",
  };

  it("applies snapshot: replaces full state including actions", () => {
    const { result } = renderHook(() => useEngineA(defaultProps));

    act(() => {
      capturedOnMessage({
        type: "snapshot",
        seq: 3,
        payload: { tick: 10, won: false, metrics: { cpu: 75, mem: 50 }, actions: ["cool_down", "increase_ttl"] },
      });
    });

    expect(result.current.tick).toBe(10);
    expect(result.current.won).toBe(false);
    expect(result.current.metrics).toEqual({ cpu: 75, mem: 50 });
    expect(result.current.actions).toEqual(["cool_down", "increase_ttl"]);
    expect(mockSetLastSeq).toHaveBeenCalledWith(3);
  });

  it("applies delta: replaces state and increments seq", () => {
    const { result } = renderHook(() => useEngineA(defaultProps));

    // First apply a snapshot at seq 1
    act(() => {
      capturedOnMessage({
        type: "snapshot",
        seq: 1,
        payload: { tick: 0, won: false, metrics: { cpu: 50 } },
      });
    });

    // Then a delta at seq 2
    act(() => {
      capturedOnMessage({
        type: "delta",
        seq: 2,
        payload: { tick: 1, won: false, metrics: { cpu: 60 } },
      });
    });

    expect(result.current.tick).toBe(1);
    expect(result.current.metrics).toEqual({ cpu: 60 });
    expect(mockSetLastSeq).toHaveBeenCalledWith(2);
  });

  it("detects seq gap and forces snapshot reconnect", () => {
    renderHook(() => useEngineA(defaultProps));

    act(() => {
      capturedOnMessage({
        type: "snapshot",
        seq: 1,
        payload: { tick: 0, won: false, metrics: {} },
      });
    });

    // Delta with gap: seq 1 -> seq 5 (expected 2)
    act(() => {
      capturedOnMessage({
        type: "delta",
        seq: 5,
        payload: { tick: 4, won: false, metrics: {} },
      });
    });

    expect(mockReconnect).toHaveBeenCalledWith(true);
  });

  it("win_update with won=true sets won state", () => {
    const { result } = renderHook(() => useEngineA(defaultProps));

    act(() => {
      capturedOnMessage({
        type: "win_update",
        payload: { won: true },
      });
    });

    expect(result.current.won).toBe(true);
  });

  it("tracks applied actions from action_accepted", () => {
    const { result } = renderHook(() => useEngineA(defaultProps));

    act(() => {
      capturedOnMessage({
        type: "snapshot",
        seq: 1,
        payload: { tick: 0, won: false, metrics: {} },
      });
    });

    act(() => {
      capturedOnMessage({
        type: "action_accepted",
        payload: { actionKey: "restart_nginx", seq: 2 },
      });
    });

    expect(result.current.appliedActions).toContain("restart_nginx");
  });

  it("dispatchAction sends apply_action with clientRequestId", () => {
    const { result } = renderHook(() => useEngineA(defaultProps));

    act(() => {
      capturedOnMessage({
        type: "snapshot",
        seq: 3,
        payload: { tick: 5, won: false, metrics: {} },
      });
    });

    act(() => {
      result.current.dispatchAction("restart_nginx");
    });

    expect(mockSend).toHaveBeenCalledTimes(1);
    const sent = mockSend.mock.calls[0][0];
    expect(sent.type).toBe("apply_action");
    expect(sent.payload.actionKey).toBe("restart_nginx");
    expect(sent.payload.expectedSeq).toBe(3);
    expect(sent.payload.clientRequestId).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i,
    );
  });
});
