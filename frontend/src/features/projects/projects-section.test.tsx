import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ProjectsSection } from "./projects-section";

const project = {
  id: "p1",
  key: "KLM",
  name: "Kelmio Aurora",
  description: "The redesign workstream",
  created_by: "u1",
  created_at: "2026-06-01",
  archived_at: null,
  project_role: "lead",
  can_write: true,
  can_manage: true,
};

function renderProjects(overrides: Record<string, unknown> = {}) {
  const props = {
    projects: [project],
    projectsError: "",
    isLoadingProjects: false,
    editingProjectId: "",
    updatingProjectIds: [],
    archivingProjectIds: [],
    isLoadingProjectDetail: false,
    role: "admin",
    canCreateProject: true,
    isCreatingProject: false,
    projectKey: "",
    projectName: "",
    projectDescription: "",
    projectFormError: "",
    selectedProjectDetail: project,
    projectDetailError: "",
    projectDetailTab: "summary",
    selectedProjectIssues: [],
    selectedProjectOpenIssues: [],
    isActive: true,
    onProjectDetailTabChange: vi.fn(),
    onSelectProjectDetail: vi.fn(),
    onProjectKeyChange: vi.fn(),
    onProjectNameChange: vi.fn(),
    onProjectDescriptionChange: vi.fn(),
    onCreateProject: vi.fn((event: { preventDefault: () => void }) =>
      event.preventDefault(),
    ),
    ...overrides,
  };
  render(<ProjectsSection {...(props as never)} />);
  return props;
}

describe("ProjectsSection", () => {
  it("preserves the settings tab and project action contract", () => {
    renderProjects();

    for (const name of ["Summary", "Members", "Workflow", "Automation"]) {
      expect(screen.getByRole("tab", { name })).toBeTruthy();
    }
    expect(screen.getByRole("button", { name: "Details" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Edit" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Archive" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Create project" })).toBeTruthy();
    expect(screen.getByLabelText("Key")).toBeTruthy();
    expect(screen.getByLabelText("Name")).toBeTruthy();
    expect(screen.getByLabelText("Description")).toBeTruthy();
  });

  it("switches settings tab", async () => {
    const props = renderProjects();
    await userEvent.click(screen.getByRole("tab", { name: "Members" }));
    expect(props.onProjectDetailTabChange).toHaveBeenCalledWith("members");
  });
});
