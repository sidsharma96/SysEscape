import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { LoginPage } from "../LoginPage";

vi.mock("@/lib/auth/useViewer");

import { useViewer } from "@/lib/auth/useViewer";
import type { Mock } from "vitest";

const mockUseViewer = useViewer as Mock;

describe("LoginPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows 'Sign in with GitHub' when unauthenticated", () => {
    mockUseViewer.mockReturnValue({
      viewer: null,
      loading: false,
      isAuthenticated: false,
    });

    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    );

    const link = screen.getByText("Sign in with GitHub");
    expect(link).toBeInTheDocument();
    expect(link.closest("a")).toHaveAttribute("href", "/auth/github/login");
  });

  it("redirects to / when already authenticated", () => {
    mockUseViewer.mockReturnValue({
      viewer: { userId: "u1", role: "player", githubUsername: "octocat" },
      loading: false,
      isAuthenticated: true,
    });

    render(
      <MemoryRouter initialEntries={["/login"]}>
        <LoginPage />
      </MemoryRouter>,
    );

    expect(screen.queryByText("Sign in with GitHub")).not.toBeInTheDocument();
  });
});
