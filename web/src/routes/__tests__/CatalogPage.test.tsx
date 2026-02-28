import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { Mock } from "vitest";
import { useQuery } from "urql";
import { CatalogPage } from "../CatalogPage";

vi.mock("urql", async () => {
  const actual = await vi.importActual<typeof import("urql")>("urql");
  return { ...actual, useQuery: vi.fn() };
});

const mockUseQuery = useQuery as Mock;

describe("CatalogPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("always renders the heading", () => {
    mockUseQuery.mockReturnValue([{ fetching: true, data: undefined, error: undefined }]);
    render(
      <MemoryRouter>
        <CatalogPage />
      </MemoryRouter>,
    );
    expect(screen.getByText("Room Catalog")).toBeInTheDocument();
  });

  it("shows skeleton cards while loading", () => {
    mockUseQuery.mockReturnValue([{ fetching: true, data: undefined, error: undefined }]);
    const { container } = render(
      <MemoryRouter>
        <CatalogPage />
      </MemoryRouter>,
    );
    const skeletons = container.querySelectorAll(".animate-skeleton-delayed");
    expect(skeletons.length).toBe(6);
  });

  it("renders room cards on success", () => {
    mockUseQuery.mockReturnValue([
      {
        fetching: false,
        error: undefined,
        data: {
          rooms: [
            {
              id: "1",
              slug: "linux-breakout",
              title: "Linux Breakout",
              district: "Kernel District",
              engine: "A",
              difficulty: "L1",
              description: "Escape a locked-down Linux environment.",
              latestVersion: null,
            },
          ],
        },
      },
    ]);
    render(
      <MemoryRouter>
        <CatalogPage />
      </MemoryRouter>,
    );
    expect(screen.getByText("Linux Breakout")).toBeInTheDocument();
  });

  it("shows error message on failure", () => {
    mockUseQuery.mockReturnValue([
      {
        fetching: false,
        data: undefined,
        error: { message: "Network error" },
      },
    ]);
    render(
      <MemoryRouter>
        <CatalogPage />
      </MemoryRouter>,
    );
    expect(screen.getByText(/Failed to load rooms/)).toBeInTheDocument();
    expect(screen.getByText(/Network error/)).toBeInTheDocument();
  });
});
