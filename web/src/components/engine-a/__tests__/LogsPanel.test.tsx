import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { LogsPanel } from "../LogsPanel";

describe("LogsPanel", () => {
  it("renders log messages", () => {
    const logs = [
      { tick: 1, message: "Server started" },
      { tick: 2, message: "Cache miss detected" },
    ];
    render(<LogsPanel logs={logs} />);
    expect(screen.getByText("Server started")).toBeInTheDocument();
    expect(screen.getByText("Cache miss detected")).toBeInTheDocument();
  });

  it("shows tick numbers alongside messages", () => {
    const logs = [{ tick: 5, message: "Request timeout" }];
    render(<LogsPanel logs={logs} />);
    expect(screen.getByText("t5")).toBeInTheDocument();
  });

  it("shows empty state text when no logs", () => {
    render(<LogsPanel logs={[]} />);
    expect(screen.getByText("No log entries.")).toBeInTheDocument();
  });
});
