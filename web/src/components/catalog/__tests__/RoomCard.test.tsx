import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect } from "vitest";
import { RoomCard } from "../RoomCard";
import type { Room } from "@/lib/graphql/queries";

const room: Room = {
  id: "1",
  slug: "linux-breakout",
  title: "Linux Breakout",
  district: "Kernel District",
  engine: "A",
  difficulty: "L1",
  description: "Escape a locked-down Linux environment.",
  latestVersion: null,
};

describe("RoomCard", () => {
  it("renders the room title", () => {
    render(
      <MemoryRouter>
        <RoomCard room={room} />
      </MemoryRouter>,
    );
    expect(screen.getByText("Linux Breakout")).toBeInTheDocument();
  });

  it("renders the district badge", () => {
    render(
      <MemoryRouter>
        <RoomCard room={room} />
      </MemoryRouter>,
    );
    expect(screen.getByText("Kernel District")).toBeInTheDocument();
  });

  it("renders the difficulty badge", () => {
    render(
      <MemoryRouter>
        <RoomCard room={room} />
      </MemoryRouter>,
    );
    expect(screen.getByText("L1 – Standard")).toBeInTheDocument();
  });

  it("links to the room detail page", () => {
    render(
      <MemoryRouter>
        <RoomCard room={room} />
      </MemoryRouter>,
    );
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/rooms/linux-breakout");
  });
});
