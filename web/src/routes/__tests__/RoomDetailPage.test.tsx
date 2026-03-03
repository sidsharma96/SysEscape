import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { Mock } from "vitest";
import { useQuery, useMutation } from "urql";
import { RoomDetailPage } from "../RoomDetailPage";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock("urql", async () => {
  const actual = await vi.importActual<typeof import("urql")>("urql");
  return { ...actual, useQuery: vi.fn(), useMutation: vi.fn() };
});

vi.mock("@/lib/idempotency", () => ({
  newRequestId: () => "test-uuid-1234",
}));

const mockUseQuery = useQuery as Mock;
const mockUseMutation = useMutation as Mock;

function renderWithRouter() {
  return render(
    <MemoryRouter initialEntries={["/rooms/linux-breakout"]}>
      <Routes>
        <Route path="rooms/:slug" element={<RoomDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

const room = {
  slug: "linux-breakout",
  title: "Linux Breakout",
  district: "Kernel District",
  engine: "A" as const,
  difficulty: "L1" as const,
  description: "Escape a locked-down Linux environment.",
  latestVersion: null,
};

describe("RoomDetailPage", () => {
  const mockExecute = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseMutation.mockReturnValue([{ fetching: false }, mockExecute]);
  });

  it("renders enabled Start Run button when room loaded", () => {
    mockUseQuery.mockReturnValue([{ fetching: false, data: { roomBySlug: room }, error: undefined }]);
    renderWithRouter();
    const btn = screen.getByRole("button", { name: /start run/i });
    expect(btn).toBeEnabled();
  });

  it("calls startRun mutation with clientRequestId and roomSlug on click", async () => {
    mockUseQuery.mockReturnValue([{ fetching: false, data: { roomBySlug: room }, error: undefined }]);
    mockExecute.mockResolvedValue({ data: { startRun: { runId: "run-abc", runToken: "tok-xyz" } } });
    renderWithRouter();

    fireEvent.click(screen.getByRole("button", { name: /start run/i }));

    await waitFor(() => {
      expect(mockExecute).toHaveBeenCalledWith({
        input: { clientRequestId: "test-uuid-1234", roomSlug: "linux-breakout" },
      });
    });
  });

  it("navigates to engine-a page with runToken on success", async () => {
    mockUseQuery.mockReturnValue([{ fetching: false, data: { roomBySlug: room }, error: undefined }]);
    mockExecute.mockResolvedValue({ data: { startRun: { runId: "run-abc", runToken: "tok-xyz" } } });
    renderWithRouter();

    fireEvent.click(screen.getByRole("button", { name: /start run/i }));

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/play/run-abc/engine-a", {
        state: { runToken: "tok-xyz" },
      });
    });
  });

  it("shows error when mutation fails", async () => {
    mockUseQuery.mockReturnValue([{ fetching: false, data: { roomBySlug: room }, error: undefined }]);
    mockExecute.mockResolvedValue({ error: { message: "Run creation failed" } });
    renderWithRouter();

    fireEvent.click(screen.getByRole("button", { name: /start run/i }));

    await waitFor(() => {
      expect(screen.getByText(/failed to start run/i)).toBeInTheDocument();
    });
  });
});
