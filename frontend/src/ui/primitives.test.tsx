import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Search } from "lucide-react";
import { describe, expect, it, vi } from "vitest";
import { Badge, Button, Field, Input, Modal, Tabs } from "./index";
import { Icon } from "./icon";

describe("ui primitives", () => {
  it("renders a primary button that defaults to type=button and fires onClick", async () => {
    const onClick = vi.fn();
    render(
      <Button variant="primary" onClick={onClick}>
        Save
      </Button>,
    );

    const button = screen.getByRole("button", { name: "Save" });
    expect(button.className).toContain("kl-btn");
    expect(button.className).toContain("kl-btn--primary");
    expect(button.getAttribute("type")).toBe("button");

    await userEvent.click(button);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("associates a field label with its control", () => {
    render(
      <Field label="Title" htmlFor="title">
        <Input id="title" defaultValue="hello" />
      </Field>,
    );

    const input = screen.getByLabelText("Title") as HTMLInputElement;
    expect(input.value).toBe("hello");
  });

  it("marks the active tab and reports changes", async () => {
    const onChange = vi.fn();
    render(
      <Tabs
        ariaLabel="Sections"
        active="board"
        onChange={onChange}
        items={[
          { id: "issues", label: "Issues" },
          { id: "board", label: "Board" },
        ]}
      />,
    );

    expect(
      screen.getByRole("tab", { name: "Board" }).getAttribute("aria-selected"),
    ).toBe("true");

    await userEvent.click(screen.getByRole("tab", { name: "Issues" }));
    expect(onChange).toHaveBeenCalledWith("issues");
  });

  it("applies semantic tone classes to badges", () => {
    render(<Badge tone="critical">Critical</Badge>);
    expect(screen.getByText("Critical").className).toContain("kl-badge--critical");
  });

  it("renders decorative icons as hidden and labelled icons as images", () => {
    const { rerender } = render(<Icon icon={Search} />);
    expect(document.querySelector(".kl-icon")?.getAttribute("aria-hidden")).toBe(
      "true",
    );

    rerender(<Icon icon={Search} label="Search" />);
    expect(screen.getByRole("img", { name: "Search" })).toBeTruthy();
  });

  it("opens a modal and closes it on Escape", async () => {
    const onClose = vi.fn();
    const { rerender } = render(
      <Modal open={false} onClose={onClose} label="Example">
        <p>Body</p>
      </Modal>,
    );
    expect(screen.queryByRole("dialog")).toBeNull();

    rerender(
      <Modal open onClose={onClose} label="Example">
        <p>Body</p>
      </Modal>,
    );
    expect(screen.getByRole("dialog", { name: "Example" })).toBeTruthy();

    await userEvent.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
