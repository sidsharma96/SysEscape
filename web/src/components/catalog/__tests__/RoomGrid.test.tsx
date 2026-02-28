import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect } from "vitest";
import { RoomGrid } from "../RoomGrid";
import type { Room } from "@/lib/graphql/queries";

const rooms: Room[] = [
  {
    id: "1",
    slug: "linux-breakout",
    title: "Linux Breakout",
    district: "Kernel District",
    engine: "A",
    difficulty: "L0",
    description: "Escape a locked-down Linux environment.",
    latestVersion: null,
  },
  {
    id: "2",
    slug: "container-escape",
    title: "Container Escape",
    district: "Docker Bay",
    engine: "B",
    difficulty: "L2",
    description: "Break out of a container sandbox.",
    latestVersion: null,
  },
];

describe("RoomGrid", () => {
  it("renders cards for each room", () => {
    render(
      <MemoryRouter>
        <RoomGrid rooms={rooms} />
      </MemoryRouter>,
    );
    expect(screen.getByText("Linux Breakout")).toBeInTheDocument();
    expect(screen.getByText("Container Escape")).toBeInTheDocument();
  });

  it("shows empty state for no rooms", () => {
    render(
      <MemoryRouter>
        <RoomGrid rooms={[]} />
      </MemoryRouter>,
    );
    expect(screen.getByText("No rooms found.")).toBeInTheDocument();
  });
});
