import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { TopologyMap } from "../TopologyMap";

describe("TopologyMap", () => {
  it("renders each node name", () => {
    const topology = [
      { name: "app-server", type: "service" },
      { name: "cache", type: "redis" },
      { name: "database", type: "postgres" },
    ];
    render(<TopologyMap topology={topology} />);
    expect(screen.getByText("app-server")).toBeInTheDocument();
    expect(screen.getByText("cache")).toBeInTheDocument();
    expect(screen.getByText("database")).toBeInTheDocument();
  });

  it("renders node type labels", () => {
    const topology = [
      { name: "app-server", type: "service" },
      { name: "cache", type: "redis" },
    ];
    render(<TopologyMap topology={topology} />);
    expect(screen.getByText("service")).toBeInTheDocument();
    expect(screen.getByText("redis")).toBeInTheDocument();
  });

  it("returns null when topology is empty", () => {
    const { container } = render(<TopologyMap topology={[]} />);
    expect(container.firstChild).toBeNull();
  });
});
