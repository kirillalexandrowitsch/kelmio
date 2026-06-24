import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { Button, ConfirmDialog, Field, Input, Tabs } from ".";

describe("Kelmio UI primitives", () => {
  it("renders semantic button variants without changing native behavior", async () => {
    const onClick = vi.fn();
    const user = userEvent.setup();

    render(
      <Button onClick={onClick} variant="primary">
        Create issue
      </Button>,
    );

    const button = screen.getByRole("button", { name: "Create issue" });
    expect(button.getAttribute("data-variant")).toBe("primary");
    await user.click(button);
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("keeps fields natively labelled", () => {
    render(
      <Field hint="Use a stable key" label="Project key">
        <Input name="project_key" />
      </Field>,
    );

    expect(screen.getByRole("textbox", { name: "Project key" })).not.toBeNull();
    expect(screen.getByText("Use a stable key")).not.toBeNull();
  });

  it("exposes tabs and selected state", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();

    render(
      <Tabs
        activeTab="summary"
        ariaLabel="Project detail"
        items={[
          { id: "summary", label: "Summary" },
          { id: "members", label: "Members" },
        ]}
        onChange={onChange}
      />,
    );

    expect(
      screen.getByRole("tab", { name: "Summary" }).getAttribute("aria-selected"),
    ).toBe("true");
    await user.click(screen.getByRole("tab", { name: "Members" }));
    expect(onChange).toHaveBeenCalledWith("members");
  });

  it("provides accessible confirmation actions", async () => {
    const onCancel = vi.fn();
    const onConfirm = vi.fn();
    const user = userEvent.setup();

    render(
      <ConfirmDialog
        confirmLabel="Archive"
        description="This project will no longer be visible."
        onCancel={onCancel}
        onConfirm={onConfirm}
        open
        title="Archive project?"
      />,
    );

    expect(screen.getByRole("dialog", { name: "Archive project?" })).not.toBeNull();
    await user.click(screen.getByRole("button", { name: "Archive" }));
    expect(onConfirm).toHaveBeenCalledOnce();
  });
});
