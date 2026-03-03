import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { WinOverlay } from "../WinOverlay";

describe("WinOverlay", () => {
  it("is visible when won is true", () => {
    render(<WinOverlay won={true} />);
    expect(screen.getByText(/you win/i)).toBeInTheDocument();
  });

  it("is hidden when won is false", () => {
    const { container } = render(<WinOverlay won={false} />);
    expect(container.firstChild).toBeNull();
  });
});
