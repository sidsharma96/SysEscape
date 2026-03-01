import { renderHook } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { Mock } from "vitest";
import { useQuery } from "urql";
import { useViewer } from "../useViewer";

vi.mock("urql", async () => {
  const actual = await vi.importActual<typeof import("urql")>("urql");
  return { ...actual, useQuery: vi.fn() };
});

const mockUseQuery = useQuery as Mock;

describe("useViewer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns viewer and isAuthenticated true when viewer exists", () => {
    const viewer = { userId: "u1", role: "player", githubUsername: "octocat" };
    mockUseQuery.mockReturnValue([
      { fetching: false, data: { viewer }, error: undefined },
    ]);

    const { result } = renderHook(() => useViewer());

    expect(result.current.viewer).toEqual(viewer);
    expect(result.current.loading).toBe(false);
    expect(result.current.isAuthenticated).toBe(true);
  });

  it("returns null viewer and isAuthenticated false when viewer is null", () => {
    mockUseQuery.mockReturnValue([
      { fetching: false, data: { viewer: null }, error: undefined },
    ]);

    const { result } = renderHook(() => useViewer());

    expect(result.current.viewer).toBeNull();
    expect(result.current.loading).toBe(false);
    expect(result.current.isAuthenticated).toBe(false);
  });

  it("returns loading true and isAuthenticated false during fetch", () => {
    mockUseQuery.mockReturnValue([
      { fetching: true, data: undefined, error: undefined },
    ]);

    const { result } = renderHook(() => useViewer());

    expect(result.current.viewer).toBeNull();
    expect(result.current.loading).toBe(true);
    expect(result.current.isAuthenticated).toBe(false);
  });
});
