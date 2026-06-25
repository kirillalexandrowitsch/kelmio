import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { CommandPalette } from "./command-palette";

const recentIssues = [
  { id: "i1", issue_key: "KLM-2481", title: "Aurora command palette" },
  { id: "i2", issue_key: "KLM-2477", title: "Peek panel slide-over" },
];

function renderPalette(overrides: Record<string, unknown> = {}) {
  const props = {
    open: true,
    onClose: vi.fn(),
    onNavigate: vi.fn(),
    onOpenIssue: vi.fn(),
    recentIssues,
    ...overrides,
  };
  render(<CommandPalette {...(props as never)} />);
  return props;
}

describe("CommandPalette", () => {
  it("is not rendered while closed", () => {
    renderPalette({ open: false });
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("lists create, go-to and recent issue commands", () => {
    renderPalette();
    expect(screen.getByRole("dialog", { name: "Command palette" })).toBeTruthy();
    expect(screen.getByRole("option", { name: /New issue/ })).toBeTruthy();
    expect(screen.getByRole("option", { name: /Board/ })).toBeTruthy();
    expect(
      screen.getByRole("option", { name: /Aurora command palette/ }),
    ).toBeTruthy();
  });

  it("navigates and closes when a go-to command is chosen", async () => {
    const props = renderPalette();
    await userEvent.click(screen.getByRole("option", { name: /Board/ }));
    expect(props.onNavigate).toHaveBeenCalledWith("board");
    expect(props.onClose).toHaveBeenCalledTimes(1);
  });

  it("filters commands by query and opens a recent issue", async () => {
    const props = renderPalette();
    const input = screen.getByLabelText("Search or run a command");

    await userEvent.type(input, "KLM-2477");
    const listbox = screen.getByRole("listbox", { name: "Commands" });
    expect(within(listbox).queryByText("New issue")).toBeNull();

    await userEvent.click(
      screen.getByRole("option", { name: /Peek panel slide-over/ }),
    );
    expect(props.onOpenIssue).toHaveBeenCalledWith("i2");
    expect(props.onClose).toHaveBeenCalledTimes(1);
  });

  it("runs the first command on Enter", async () => {
    const props = renderPalette();
    await userEvent.type(
      screen.getByLabelText("Search or run a command"),
      "new project{Enter}",
    );
    expect(props.onNavigate).toHaveBeenCalledWith("projects");
  });
});
