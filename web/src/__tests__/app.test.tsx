import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { App } from "../App.tsx";

describe("App", () => {
  it("renders the catalog page by default", () => {
    render(<App />);
    expect(screen.getByText("Room Catalog")).toBeInTheDocument();
  });

  it("renders the navigation bar", () => {
    render(<App />);
    expect(screen.getByText("Systems Escape Rooms")).toBeInTheDocument();
  });
});
