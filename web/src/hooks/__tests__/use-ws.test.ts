import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { useWs } from "../use-ws";

let mockInstances: MockWebSocket[];

class MockWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;
  readonly CONNECTING = 0;
  readonly OPEN = 1;
  readonly CLOSING = 2;
  readonly CLOSED = 3;

  url: string;
  readyState = MockWebSocket.CONNECTING;
  onopen: ((ev: Event) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  sent: string[] = [];

  constructor(url: string) {
    this.url = url;
    mockInstances.push(this);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
  }

  // Test helpers
  simulateOpen() {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.(new Event("open"));
  }

  simulateMessage(data: unknown) {
    this.onmessage?.(new MessageEvent("message", { data: JSON.stringify(data) }));
  }

  simulateClose(code = 1000) {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent("close", { code }));
  }
}

describe("useWs", () => {
  beforeEach(() => {
    mockInstances = [];
    vi.useFakeTimers();
    vi.stubGlobal("WebSocket", MockWebSocket);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  const defaultProps = {
    url: "ws://localhost:8081/ws/engineA/run-1",
    runId: "run-1",
    runToken: "tok-1",
    onMessage: vi.fn(),
  };

  it("sends hello with runToken on open", () => {
    renderHook(() => useWs(defaultProps));
    const ws = mockInstances[0];
    ws.simulateOpen();

    expect(ws.sent).toHaveLength(1);
    const hello = JSON.parse(ws.sent[0]);
    expect(hello.type).toBe("hello");
    expect(hello.protocolVersion).toBe(1);
    expect(hello.runId).toBe("run-1");
    expect(hello.payload.runToken).toBe("tok-1");
  });

  it("replies pong to server ping", () => {
    renderHook(() => useWs(defaultProps));
    const ws = mockInstances[0];
    ws.simulateOpen();
    ws.simulateMessage({ type: "ping" });

    expect(ws.sent).toHaveLength(2);
    const pong = JSON.parse(ws.sent[1]);
    expect(pong.type).toBe("pong");
  });

  it("forwards non-ping messages to onMessage", () => {
    const onMessage = vi.fn();
    renderHook(() => useWs({ ...defaultProps, onMessage }));
    const ws = mockInstances[0];
    ws.simulateOpen();

    const snapshot = { type: "snapshot", seq: 1, payload: { tick: 0, won: false, metrics: {} } };
    ws.simulateMessage(snapshot);

    expect(onMessage).toHaveBeenCalledWith(snapshot);
  });

  it("reconnects with exponential backoff", () => {
    renderHook(() => useWs(defaultProps));
    const ws0 = mockInstances[0];
    ws0.simulateOpen();
    ws0.simulateClose(1006);

    // First reconnect after 500ms
    expect(mockInstances).toHaveLength(1);
    act(() => { vi.advanceTimersByTime(500); });
    expect(mockInstances).toHaveLength(2);

    // Second reconnect after 1000ms
    mockInstances[1].simulateOpen();
    mockInstances[1].simulateClose(1006);
    act(() => { vi.advanceTimersByTime(1000); });
    expect(mockInstances).toHaveLength(3);

    // Third reconnect after 2000ms
    mockInstances[2].simulateOpen();
    mockInstances[2].simulateClose(1006);
    act(() => { vi.advanceTimersByTime(2000); });
    expect(mockInstances).toHaveLength(4);
  });

  it("resets backoff on hello_ack", () => {
    renderHook(() => useWs(defaultProps));
    const ws0 = mockInstances[0];
    ws0.simulateOpen();
    ws0.simulateClose(1006);

    act(() => { vi.advanceTimersByTime(500); });
    const ws1 = mockInstances[1];
    ws1.simulateOpen();
    ws1.simulateMessage({ type: "hello_ack", payload: { snapshotRequired: true } });
    ws1.simulateClose(1006);

    // After hello_ack, backoff resets — next reconnect at 500ms not 1000ms
    act(() => { vi.advanceTimersByTime(500); });
    expect(mockInstances).toHaveLength(3);
  });

  it("sends resumeFromSeq on reconnect when lastSeq provided", () => {
    const onMessage = vi.fn();
    const { result } = renderHook(() => useWs({ ...defaultProps, onMessage }));
    const ws0 = mockInstances[0];
    ws0.simulateOpen();

    // Simulate receiving hello_ack and a snapshot with seq
    ws0.simulateMessage({ type: "hello_ack", payload: { snapshotRequired: true } });
    ws0.simulateMessage({ type: "snapshot", seq: 5, payload: { tick: 5, won: false, metrics: {} } });

    // Update lastSeq via the hook
    act(() => { result.current.setLastSeq(5); });

    ws0.simulateClose(1006);
    act(() => { vi.advanceTimersByTime(500); });

    const ws1 = mockInstances[1];
    ws1.simulateOpen();
    const hello = JSON.parse(ws1.sent[0]);
    expect(hello.payload.resumeFromSeq).toBe(5);
  });
});
