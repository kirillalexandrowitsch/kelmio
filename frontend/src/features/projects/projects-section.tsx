import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type CurrentUser,
  type AutomationRule,
  type CreateAutomationRuleInput,
  type Issue,
  type Label,
  type Project,
  type ProjectMember,
  type ProjectRole,
  type ProjectWorkflow,
  type ProjectWorkflowStatus,
  type TeamMember,
  type CreateWorkflowStatusInput,
  type UpdateWorkflowStatusInput,
  type WorkflowTransitionInput,
} from "../../lib/api-types";
import { PROJECT_PERMISSION_NOTE } from "../../lib/permissions";
import { hasText } from "../../lib/validation";
import { Badge, Button, Field, Input, TextArea, Tabs } from "../../ui";
import { ProjectMembersPanel } from "./project-members-panel";
import { WorkflowSettingsPanel } from "./workflow-settings-panel";
import { AutomationSettingsPanel } from "./automation-settings-panel";

export type ProjectDetailTab = "summary" | "members" | "workflow" | "automation";

const ROLE_LABEL: Record<"lead" | "contributor" | "viewer", string> = {
  lead: "Lead",
  contributor: "Contributor",
  viewer: "Viewer",
};

const DETAIL_TABS = [
  { id: "summary", label: "Summary" },
  { id: "members", label: "Members" },
  { id: "workflow", label: "Workflow" },
  { id: "automation", label: "Automation" },
] as const;

type ProjectsSectionProps = {
  archivingProjectIds: string[];
  canCreateProject: boolean;
  editProjectDescription: string;
  editProjectName: string;
  editingProjectId: string;
  isActive: boolean;
  isCreatingProject: boolean;
  isLoadingProjectDetail: boolean;
  isLoadingProjectMembers: boolean;
  isLoadingProjectWorkflow: boolean;
  isLoadingAutomationRules: boolean;
  isCreatingAutomationRule: boolean;
  isReorderingAutomationRules: boolean;
  isLoadingProjects: boolean;
  isReorderingWorkflowStatuses: boolean;
  isSavingWorkflowTransitions: boolean;
  onArchiveProject: (project: Project) => void;
  onCancelEditingProject: () => void;
  onCreateProject: (event: FormEvent<HTMLFormElement>) => void;
  onEditProjectDescriptionChange: (value: string) => void;
  onEditProjectNameChange: (value: string) => void;
  onOpenProjectBoard: (projectId: string) => void;
  onAddProjectMember: (event: FormEvent<HTMLFormElement>) => void;
  onProjectDetailTabChange: (tab: ProjectDetailTab) => void;
  onArchiveWorkflowStatus: (
    status: ProjectWorkflowStatus,
    replacementStatusId: string,
  ) => Promise<boolean>;
  onCreateAutomationRule: (input: CreateAutomationRuleInput) => Promise<boolean>;
  onDeleteAutomationRule: (rule: AutomationRule) => Promise<boolean>;
  onReorderAutomationRules: (ruleIds: string[]) => Promise<boolean>;
  onUpdateAutomationRule: (
    rule: AutomationRule,
    input: CreateAutomationRuleInput | { is_enabled: boolean },
  ) => Promise<boolean>;
  onCreateWorkflowStatus: (input: CreateWorkflowStatusInput) => Promise<boolean>;
  onReorderWorkflowStatuses: (statusIds: string[]) => Promise<boolean>;
  onReplaceWorkflowTransitions: (
    transitions: WorkflowTransitionInput[],
  ) => Promise<boolean>;
  onUpdateWorkflowStatus: (
    status: ProjectWorkflowStatus,
    input: UpdateWorkflowStatusInput,
  ) => Promise<boolean>;
  onProjectMemberRoleChange: (member: ProjectMember, role: ProjectRole) => void;
  onProjectMemberRoleSelectionChange: (role: ProjectRole) => void;
  onProjectMemberUserChange: (userId: string) => void;
  onRemoveProjectMember: (member: ProjectMember) => void;
  onProjectDescriptionChange: (value: string) => void;
  onProjectKeyChange: (value: string) => void;
  onProjectNameChange: (value: string) => void;
  onSelectIssue: (issueId: string) => void;
  onSelectProjectDetail: (projectId: string) => void;
  onStartEditingProject: (project: Project) => void;
  onUpdateProject: (event: FormEvent<HTMLFormElement>, project: Project) => void;
  onViewProjectIssues: (projectId: string) => void;
  projectDescription: string;
  projectDetailTab: ProjectDetailTab;
  projectDetailError: string;
  projectFormError: string;
  projectKey: string;
  projectName: string;
  projects: Project[];
  projectsError: string;
  projectMembers: ProjectMember[];
  projectMembersError: string;
  projectWorkflow?: ProjectWorkflow;
  projectWorkflowError: string;
  automationRules: AutomationRule[];
  automationRulesError: string;
  deletingAutomationRuleIds: string[];
  archivingWorkflowStatusIds: string[];
  creatingWorkflowStatus: boolean;
  removingProjectMemberIds: string[];
  role: CurrentUser["workspace"]["role"];
  selectedProjectMemberRole: ProjectRole;
  selectedProjectMemberUserId: string;
  selectedProjectDetail: Project | null;
  selectedProjectIssues: Issue[];
  selectedProjectOpenIssues: Issue[];
  teamMembers: TeamMember[];
  updatingProjectMemberIds: string[];
  updatingProjectIds: string[];
  updatingWorkflowStatusIds: string[];
  updatingAutomationRuleIds: string[];
  labels: Label[];
};

export function ProjectsSection({
  archivingProjectIds,
  canCreateProject,
  editProjectDescription,
  editProjectName,
  editingProjectId,
  isActive,
  isCreatingProject,
  isLoadingProjectDetail,
  isLoadingProjectMembers,
  isLoadingProjectWorkflow,
  isLoadingAutomationRules,
  isCreatingAutomationRule,
  isReorderingAutomationRules,
  isLoadingProjects,
  isReorderingWorkflowStatuses,
  isSavingWorkflowTransitions,
  onAddProjectMember,
  onArchiveProject,
  onCancelEditingProject,
  onCreateProject,
  onEditProjectDescriptionChange,
  onEditProjectNameChange,
  onOpenProjectBoard,
  onProjectDetailTabChange,
  onArchiveWorkflowStatus,
  onCreateAutomationRule,
  onDeleteAutomationRule,
  onReorderAutomationRules,
  onUpdateAutomationRule,
  onCreateWorkflowStatus,
  onProjectMemberRoleChange,
  onProjectMemberRoleSelectionChange,
  onProjectMemberUserChange,
  onReorderWorkflowStatuses,
  onRemoveProjectMember,
  onReplaceWorkflowTransitions,
  onProjectDescriptionChange,
  onProjectKeyChange,
  onProjectNameChange,
  onSelectIssue,
  onSelectProjectDetail,
  onStartEditingProject,
  onUpdateProject,
  onUpdateWorkflowStatus,
  onViewProjectIssues,
  projectDescription,
  projectDetailTab,
  projectDetailError,
  projectFormError,
  projectKey,
  projectName,
  projects,
  projectsError,
  projectMembers,
  projectMembersError,
  projectWorkflow,
  projectWorkflowError,
  automationRules,
  automationRulesError,
  deletingAutomationRuleIds,
  archivingWorkflowStatusIds,
  creatingWorkflowStatus,
  removingProjectMemberIds,
  role,
  selectedProjectMemberRole,
  selectedProjectMemberUserId,
  selectedProjectDetail,
  selectedProjectIssues,
  selectedProjectOpenIssues,
  teamMembers,
  updatingProjectMemberIds,
  updatingProjectIds,
  updatingWorkflowStatusIds,
  updatingAutomationRuleIds,
  labels,
}: ProjectsSectionProps) {
  const isAdmin = role === "admin";

  return (
    <section className="kl-projects" aria-label="Projects" hidden={!isActive}>
      <header className="kl-section-head">
        <div>
          <p className="kl-eyebrow">Workspace</p>
          <h2>Projects</h2>
        </div>
        {isLoadingProjects ? <span className="kl-muted">Loading</span> : null}
      </header>

      <FormError message={projectsError} />

      {projects.length > 0 ? (
        <div className="kl-projects__grid">
          {projects.map((project) => {
            const isEditingProject = editingProjectId === project.id;
            const isUpdatingProject = updatingProjectIds.includes(project.id);
            const isArchivingProject = archivingProjectIds.includes(project.id);

            return (
              <article className="project-row kl-project-card" key={project.id}>
                <div className="kl-project-card__top">
                  <span className="kl-project-card__key">{project.key}</span>
                  {project.project_role ? (
                    <Badge tone="accent">{ROLE_LABEL[project.project_role]}</Badge>
                  ) : null}
                  {project.archived_at ? <Badge>Archived</Badge> : null}
                </div>

                {isEditingProject ? (
                  <form
                    className="kl-project-card__form"
                    onSubmit={(event) => onUpdateProject(event, project)}
                  >
                    <Field label="Name" htmlFor={`edit-name-${project.id}`}>
                      <Input
                        id={`edit-name-${project.id}`}
                        maxLength={120}
                        onChange={(event) =>
                          onEditProjectNameChange(event.target.value)
                        }
                        value={editProjectName}
                      />
                    </Field>
                    <Field
                      label="Description"
                      htmlFor={`edit-desc-${project.id}`}
                    >
                      <TextArea
                        id={`edit-desc-${project.id}`}
                        onChange={(event) =>
                          onEditProjectDescriptionChange(event.target.value)
                        }
                        rows={2}
                        value={editProjectDescription}
                      />
                    </Field>
                    <div className="kl-project-card__actions">
                      <Button
                        variant="primary"
                        size="sm"
                        disabled={isUpdatingProject || !hasText(editProjectName)}
                        type="submit"
                      >
                        {isUpdatingProject ? "Saving" : "Save"}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={isUpdatingProject}
                        onClick={onCancelEditingProject}
                      >
                        Cancel
                      </Button>
                    </div>
                  </form>
                ) : (
                  <div className="kl-project-card__body">
                    <h3>{project.name}</h3>
                    <p>{project.description || "No description"}</p>
                  </div>
                )}

                {!isEditingProject ? (
                  <div className="kl-project-card__actions">
                    <Button
                      variant="secondary"
                      size="sm"
                      disabled={isLoadingProjectDetail}
                      onClick={() => onSelectProjectDetail(project.id)}
                    >
                      Details
                    </Button>
                    {isAdmin ? (
                      <>
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={isArchivingProject}
                          onClick={() => onStartEditingProject(project)}
                        >
                          Edit
                        </Button>
                        <Button
                          variant="danger"
                          size="sm"
                          disabled={isArchivingProject}
                          onClick={() => onArchiveProject(project)}
                        >
                          {isArchivingProject ? "Archiving" : "Archive"}
                        </Button>
                      </>
                    ) : null}
                  </div>
                ) : null}
              </article>
            );
          })}
        </div>
      ) : (
        <div className="kl-empty-block">No projects yet</div>
      )}

      {isAdmin ? (
        <form className="project-form kl-card kl-project-create" onSubmit={onCreateProject}>
          <header className="kl-section-head">
            <div>
              <p className="kl-eyebrow">Admin</p>
              <h2>Create project</h2>
            </div>
          </header>

          <div className="kl-project-create__fields">
            <Field label="Key" htmlFor="create-key">
              <Input
                id="create-key"
                maxLength={10}
                onChange={(event) =>
                  onProjectKeyChange(event.target.value.toUpperCase())
                }
                placeholder="CORE"
                value={projectKey}
              />
            </Field>
            <Field label="Name" htmlFor="create-name">
              <Input
                id="create-name"
                maxLength={120}
                onChange={(event) => onProjectNameChange(event.target.value)}
                placeholder="Core Platform"
                value={projectName}
              />
            </Field>
          </div>

          <Field label="Description" htmlFor="create-desc">
            <TextArea
              id="create-desc"
              onChange={(event) => onProjectDescriptionChange(event.target.value)}
              placeholder="Main product workspace"
              rows={3}
              value={projectDescription}
            />
          </Field>

          <FormError message={projectFormError} />

          <Button variant="primary" disabled={!canCreateProject} type="submit">
            {isCreatingProject ? "Creating..." : "Create project"}
          </Button>
        </form>
      ) : (
        <aside className="kl-card kl-permission-note">
          <p className="kl-eyebrow">{PROJECT_PERMISSION_NOTE.eyebrow}</p>
          <h2>{PROJECT_PERMISSION_NOTE.title}</h2>
          <p>{PROJECT_PERMISSION_NOTE.body}</p>
        </aside>
      )}

      <aside className="kl-card kl-project-detail" aria-label="Project details">
        <header className="kl-section-head">
          <div>
            <p className="kl-eyebrow">Project detail</p>
            <h2>
              {selectedProjectDetail
                ? `${selectedProjectDetail.key} · ${selectedProjectDetail.name}`
                : "Select project"}
            </h2>
          </div>
          {isLoadingProjectDetail ? <span className="kl-muted">Loading</span> : null}
        </header>

        <FormError message={projectDetailError} />

        {selectedProjectDetail ? (
          <>
            <p className="kl-muted">
              {selectedProjectDetail.description || "No description"}
            </p>

            {selectedProjectDetail.can_manage ? (
              <Tabs<ProjectDetailTab>
                ariaLabel="Project settings"
                active={projectDetailTab}
                items={DETAIL_TABS}
                onChange={onProjectDetailTabChange}
              />
            ) : null}

            {projectDetailTab === "automation" && selectedProjectDetail.can_manage ? (
              <AutomationSettingsPanel
                creatingRule={isCreatingAutomationRule}
                deletingRuleIds={deletingAutomationRuleIds}
                error={automationRulesError}
                isLoading={isLoadingAutomationRules}
                isReordering={isReorderingAutomationRules}
                labels={labels}
                members={projectMembers}
                onCreateRule={onCreateAutomationRule}
                onDeleteRule={onDeleteAutomationRule}
                onReorderRules={onReorderAutomationRules}
                onUpdateRule={onUpdateAutomationRule}
                rules={automationRules}
                teamMembers={teamMembers}
                updatingRuleIds={updatingAutomationRuleIds}
                workflow={projectWorkflow}
              />
            ) : projectDetailTab === "workflow" && selectedProjectDetail.can_manage ? (
              <WorkflowSettingsPanel
                archivingStatusIds={archivingWorkflowStatusIds}
                creatingStatus={creatingWorkflowStatus}
                error={projectWorkflowError}
                isLoading={isLoadingProjectWorkflow}
                isReordering={isReorderingWorkflowStatuses}
                isSavingTransitions={isSavingWorkflowTransitions}
                onArchiveStatus={onArchiveWorkflowStatus}
                onCreateStatus={onCreateWorkflowStatus}
                onReorderStatuses={onReorderWorkflowStatuses}
                onReplaceTransitions={onReplaceWorkflowTransitions}
                onUpdateStatus={onUpdateWorkflowStatus}
                updatingStatusIds={updatingWorkflowStatusIds}
                workflow={projectWorkflow}
              />
            ) : projectDetailTab === "members" && selectedProjectDetail.can_manage ? (
              <ProjectMembersPanel
                error={projectMembersError}
                isLoading={isLoadingProjectMembers}
                members={projectMembers}
                onAddMember={onAddProjectMember}
                onMemberRoleChange={onProjectMemberRoleChange}
                onRemoveMember={onRemoveProjectMember}
                onRoleChange={onProjectMemberRoleSelectionChange}
                onUserChange={onProjectMemberUserChange}
                removingMemberIds={removingProjectMemberIds}
                role={selectedProjectMemberRole}
                selectedUserId={selectedProjectMemberUserId}
                teamMembers={teamMembers}
                updatingMemberIds={updatingProjectMemberIds}
              />
            ) : (
              <>
                {!selectedProjectDetail.can_manage ? (
                  <aside className="kl-project-access">
                    <strong>
                      {selectedProjectDetail.project_role === "viewer"
                        ? "Viewer access"
                        : "Project access"}
                    </strong>
                    <span>
                      {selectedProjectDetail.can_write
                        ? "You can work with project issues, comments, and sprints. Project access is managed by a lead or workspace admin."
                        : "This project is read-only. Ask a project lead or workspace admin for access changes."}
                    </span>
                  </aside>
                ) : null}

                <div className="kl-project-stats">
                  <article>
                    <span>Visible issues</span>
                    <strong>{selectedProjectIssues.length}</strong>
                  </article>
                  <article>
                    <span>Open</span>
                    <strong>{selectedProjectOpenIssues.length}</strong>
                  </article>
                </div>

                <div className="kl-project-detail__actions">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => onViewProjectIssues(selectedProjectDetail.id)}
                  >
                    View project issues
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => onOpenProjectBoard(selectedProjectDetail.id)}
                  >
                    Open project board
                  </Button>
                </div>

                {selectedProjectIssues.length > 0 ? (
                  <div className="kl-project-detail__issues">
                    {selectedProjectIssues.slice(0, 4).map((issue) => (
                      <button
                        key={issue.id}
                        onClick={() => onSelectIssue(issue.id)}
                        type="button"
                      >
                        <span>{issue.issue_key}</span>
                        <strong>{issue.title}</strong>
                      </button>
                    ))}
                  </div>
                ) : (
                  <div className="kl-empty-block">
                    No visible issues for this project
                  </div>
                )}
              </>
            )}
          </>
        ) : (
          <div className="kl-empty-block">No project selected</div>
        )}
      </aside>
    </section>
  );
}
