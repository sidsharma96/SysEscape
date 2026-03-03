import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ActionBar } from "../ActionBar";

describe("ActionBar", () => {
  const actions = ["restart_nginx", "flush_cache", "scale_up"];

  it("renders a button for each action", () => {
    render(<ActionBar actions={actions} appliedActions={[]} onDispatch={vi.fn()} />);
    expect(screen.getByText("restart_nginx")).toBeInTheDocument();
    expect(screen.getByText("flush_cache")).toBeInTheDocument();
    expect(screen.getByText("scale_up")).toBeInTheDocument();
  });

  it("calls onDispatch with the action key on click", () => {
    const onDispatch = vi.fn();
    render(<ActionBar actions={actions} appliedActions={[]} onDispatch={onDispatch} />);
    fireEvent.click(screen.getByText("restart_nginx"));
    expect(onDispatch).toHaveBeenCalledWith("restart_nginx");
  });

  it("disables already-applied action buttons", () => {
    render(<ActionBar actions={actions} appliedActions={["flush_cache"]} onDispatch={vi.fn()} />);
    expect(screen.getByText("flush_cache")).toBeDisabled();
    expect(screen.getByText("restart_nginx")).toBeEnabled();
  });
});
