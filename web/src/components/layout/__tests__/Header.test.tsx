import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { Header } from "../Header";

vi.mock("@/lib/auth/useViewer");

import { useViewer } from "@/lib/auth/useViewer";
import type { Mock } from "vitest";

const mockUseViewer = useViewer as Mock;

describe("Header", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("always shows 'Systems Escape Rooms'", () => {
    mockUseViewer.mockReturnValue({
      viewer: null,
      loading: false,
      isAuthenticated: false,
    });

    render(
      <MemoryRouter>
        <Header />
      </MemoryRouter>,
    );

    expect(screen.getByText("Systems Escape Rooms")).toBeInTheDocument();
  });

  it("shows 'Sign in with GitHub' when unauthenticated", () => {
    mockUseViewer.mockReturnValue({
      viewer: null,
      loading: false,
      isAuthenticated: false,
    });

    render(
      <MemoryRouter>
        <Header />
      </MemoryRouter>,
    );

    const link = screen.getByText("Sign in with GitHub");
    expect(link).toBeInTheDocument();
    expect(link.closest("a")).toHaveAttribute("href", "/auth/github/login");
  });

  it("shows username and 'Log out' when authenticated", () => {
    mockUseViewer.mockReturnValue({
      viewer: { userId: "u1", role: "player", githubUsername: "octocat" },
      loading: false,
      isAuthenticated: true,
    });

    render(
      <MemoryRouter>
        <Header />
      </MemoryRouter>,
    );

    expect(screen.getByText("octocat")).toBeInTheDocument();
    expect(screen.getByText("Log out")).toBeInTheDocument();
  });

  it("hides auth section while loading", () => {
    mockUseViewer.mockReturnValue({
      viewer: null,
      loading: true,
      isAuthenticated: false,
    });

    render(
      <MemoryRouter>
        <Header />
      </MemoryRouter>,
    );

    expect(screen.queryByText("Sign in with GitHub")).not.toBeInTheDocument();
    expect(screen.queryByText("Log out")).not.toBeInTheDocument();
  });
});
