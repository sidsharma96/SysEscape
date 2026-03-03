import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { TimerBar } from "../TimerBar";

describe("TimerBar", () => {
  it("shows tick/totalTicks progress text", () => {
    render(<TimerBar tick={5} totalTicks={20} />);
    expect(screen.getByText("5 / 20")).toBeInTheDocument();
  });

  it("sets progress bar width percentage via style", () => {
    render(<TimerBar tick={10} totalTicks={20} />);
    const bar = screen.getByTestId("timer-progress");
    expect(bar.style.width).toBe("50%");
  });

  it("applies signal-warn class when >75%", () => {
    render(<TimerBar tick={16} totalTicks={20} />);
    const bar = screen.getByTestId("timer-progress");
    expect(bar.className).toContain("bg-signal-warn");
  });

  it("applies signal-crit class when >=90%", () => {
    render(<TimerBar tick={18} totalTicks={20} />);
    const bar = screen.getByTestId("timer-progress");
    expect(bar.className).toContain("bg-signal-crit");
  });

  it("shows simple tick display when totalTicks undefined", () => {
    render(<TimerBar tick={7} />);
    expect(screen.getByText("Tick 7")).toBeInTheDocument();
  });
});
