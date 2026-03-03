import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { MetricsPanel } from "../MetricsPanel";

describe("MetricsPanel", () => {
  it("renders metric names and values", () => {
    render(<MetricsPanel metrics={{ cpu: 72.5, mem: 48.3, disk: 90 }} />);
    expect(screen.getByText("cpu")).toBeInTheDocument();
    expect(screen.getByText("72.5")).toBeInTheDocument();
    expect(screen.getByText("mem")).toBeInTheDocument();
    expect(screen.getByText("48.3")).toBeInTheDocument();
    expect(screen.getByText("disk")).toBeInTheDocument();
    expect(screen.getByText("90")).toBeInTheDocument();
  });

  it("rounds floating point values to 4 decimal places", () => {
    render(<MetricsPanel metrics={{ error_rate: 0.12100000000000005, pi: 3.14159265 }} />);
    expect(screen.getByText("0.121")).toBeInTheDocument();
    expect(screen.getByText("3.1416")).toBeInTheDocument();
  });

  it("renders empty state when no metrics", () => {
    const { container } = render(<MetricsPanel metrics={{}} />);
    expect(container.querySelectorAll("tr")).toHaveLength(0);
  });
});
