import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { Mock } from "vitest";
import { useQuery } from "urql";
import { App } from "../App.tsx";

vi.mock("urql", async () => {
  const actual = await vi.importActual<typeof import("urql")>("urql");
  return { ...actual, useQuery: vi.fn() };
});

const mockUseQuery = useQuery as Mock;

describe("App", () => {
  beforeEach(() => {
    mockUseQuery.mockReturnValue([
      { fetching: false, data: { viewer: null, rooms: [] }, error: undefined },
    ]);
  });

  it("renders the catalog page by default", () => {
    render(<App />);
    expect(screen.getByText("Room Catalog")).toBeInTheDocument();
  });

  it("renders the navigation bar", () => {
    render(<App />);
    expect(screen.getByText("Systems Escape Rooms")).toBeInTheDocument();
  });
});
