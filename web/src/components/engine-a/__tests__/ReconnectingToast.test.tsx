import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ReconnectingToast } from "../ReconnectingToast";

describe("ReconnectingToast", () => {
  it("shows Reconnecting text when state is reconnecting", () => {
    render(<ReconnectingToast connectionState="reconnecting" />);
    expect(screen.getByText("Reconnecting...")).toBeInTheDocument();
  });

  it("shows Disconnected text when state is disconnected", () => {
    render(<ReconnectingToast connectionState="disconnected" />);
    expect(screen.getByText("Disconnected")).toBeInTheDocument();
  });

  it("returns null when state is connected", () => {
    const { container } = render(<ReconnectingToast connectionState="connected" />);
    expect(container.firstChild).toBeNull();
  });
});
