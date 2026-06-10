import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type CurrentUser,
  type Issue,
  type Project,
  type ProjectMember,
  type ProjectRole,
  type TeamMember,
} from "../../lib/api-types";
import { PROJECT_PERMISSION_NOTE } from "../../lib/permissions";
import { hasText } from "../../lib/validation";
import { ProjectMembersPanel } from "./project-members-panel";

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
  isLoadingProjects: boolean;
  onArchiveProject: (project: Project) => void;
  onCancelEditingProject: () => void;
  onCreateProject: (event: FormEvent<HTMLFormElement>) => void;
  onEditProjectDescriptionChange: (value: string) => void;
  onEditProjectNameChange: (value: string) => void;
  onOpenProjectBoard: (projectId: string) => void;
  onAddProjectMember: (event: FormEvent<HTMLFormElement>) => void;
  onProjectDetailTabChange: (tab: "summary" | "members") => void;
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
  projectDetailTab: "summary" | "members";
  projectDetailError: string;
  projectFormError: string;
  projectKey: string;
  projectName: string;
  projects: Project[];
  projectsError: string;
  projectMembers: ProjectMember[];
  projectMembersError: string;
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
  isLoadingProjects,
  onAddProjectMember,
  onArchiveProject,
  onCancelEditingProject,
  onCreateProject,
  onEditProjectDescriptionChange,
  onEditProjectNameChange,
  onOpenProjectBoard,
  onProjectDetailTabChange,
  onProjectMemberRoleChange,
  onProjectMemberRoleSelectionChange,
  onProjectMemberUserChange,
  onRemoveProjectMember,
  onProjectDescriptionChange,
  onProjectKeyChange,
  onProjectNameChange,
  onSelectIssue,
  onSelectProjectDetail,
  onStartEditingProject,
  onUpdateProject,
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
}: ProjectsSectionProps) {
  const isAdmin = role === "admin";

  return (
    <section
      className="projects-layout"
      aria-label="Projects"
      hidden={!isActive}
    >
      <div className="projects-panel">
        <header className="section-header">
          <div>
            <p className="eyebrow">Projects</p>
            <h2>Workspace projects</h2>
          </div>
          {isLoadingProjects ? <span className="muted">Loading</span> : null}
        </header>

        <FormError message={projectsError} />

        {projects.length > 0 ? (
          <div className="project-list">
            {projects.map((project) => {
              const isEditingProject = editingProjectId === project.id;
              const isUpdatingProject = updatingProjectIds.includes(project.id);
              const isArchivingProject = archivingProjectIds.includes(project.id);

              return (
                <article className="project-row" key={project.id}>
                  <span className="project-key">{project.key}</span>
                  {isEditingProject ? (
                    <form
                      className="project-inline-form"
                      onSubmit={(event) => onUpdateProject(event, project)}
                    >
                      <label>
                        <span>Name</span>
                        <input
                          maxLength={120}
                          onChange={(event) =>
                            onEditProjectNameChange(event.target.value)
                          }
                          value={editProjectName}
                        />
                      </label>
                      <label>
                        <span>Description</span>
                        <textarea
                          onChange={(event) =>
                            onEditProjectDescriptionChange(event.target.value)
                          }
                          rows={2}
                          value={editProjectDescription}
                        />
                      </label>
                      <div className="project-row-actions">
                        <button
                          className="small-button"
                          disabled={isUpdatingProject || !hasText(editProjectName)}
                          type="submit"
                        >
                          {isUpdatingProject ? "Saving" : "Save"}
                        </button>
                        <button
                          className="ghost-button"
                          disabled={isUpdatingProject}
                          onClick={onCancelEditingProject}
                          type="button"
                        >
                          Cancel
                        </button>
                      </div>
                    </form>
                  ) : (
                    <div>
                      <h3>{project.name}</h3>
                      <p>{project.description || "No description"}</p>
                    </div>
                  )}
                  {!isEditingProject ? (
                    <div className="project-row-actions">
                      <button
                        className="small-button"
                        disabled={isLoadingProjectDetail}
                        onClick={() => onSelectProjectDetail(project.id)}
                        type="button"
                      >
                        Details
                      </button>
                      {isAdmin ? (
                        <>
                          <button
                            className="small-button"
                            disabled={isArchivingProject}
                            onClick={() => onStartEditingProject(project)}
                            type="button"
                          >
                            Edit
                          </button>
                          <button
                            className="small-button danger-button"
                            disabled={isArchivingProject}
                            onClick={() => onArchiveProject(project)}
                            type="button"
                          >
                            {isArchivingProject ? "Archiving" : "Archive"}
                          </button>
                        </>
                      ) : null}
                    </div>
                  ) : null}
                </article>
              );
            })}
          </div>
        ) : (
          <div className="project-empty">No projects yet</div>
        )}
      </div>

      <div className="project-sidebar">
        {isAdmin ? (
          <form className="project-form" onSubmit={onCreateProject}>
            <header className="section-header">
              <div>
                <p className="eyebrow">Admin</p>
                <h2>Create project</h2>
              </div>
            </header>

            <label>
              <span>Key</span>
              <input
                maxLength={10}
                onChange={(event) =>
                  onProjectKeyChange(event.target.value.toUpperCase())
                }
                placeholder="CORE"
                value={projectKey}
              />
            </label>

            <label>
              <span>Name</span>
              <input
                maxLength={120}
                onChange={(event) => onProjectNameChange(event.target.value)}
                placeholder="Core Platform"
                value={projectName}
              />
            </label>

            <label>
              <span>Description</span>
              <textarea
                onChange={(event) => onProjectDescriptionChange(event.target.value)}
                placeholder="Main product workspace"
                rows={4}
                value={projectDescription}
              />
            </label>

            <FormError message={projectFormError} />

            <button disabled={!canCreateProject} type="submit">
              {isCreatingProject ? "Creating..." : "Create project"}
            </button>
          </form>
        ) : (
          <aside className="project-form permission-note">
            <header className="section-header">
              <div>
                <p className="eyebrow">{PROJECT_PERMISSION_NOTE.eyebrow}</p>
                <h2>{PROJECT_PERMISSION_NOTE.title}</h2>
              </div>
            </header>

            <p>{PROJECT_PERMISSION_NOTE.body}</p>
          </aside>
        )}

        <aside className="project-detail-panel" aria-label="Project details">
          <header className="section-header">
            <div>
              <p className="eyebrow">Project detail</p>
              <h2>
                {selectedProjectDetail
                  ? `${selectedProjectDetail.key} · ${selectedProjectDetail.name}`
                  : "Select project"}
              </h2>
            </div>
            {isLoadingProjectDetail ? <span className="muted">Loading</span> : null}
          </header>

          <FormError message={projectDetailError} />

          {selectedProjectDetail ? (
            <>
              <p className="project-detail-description">
                {selectedProjectDetail.description || "No description"}
              </p>

              {selectedProjectDetail.can_manage ? (
                <div className="project-detail-tabs" role="tablist" aria-label="Project details">
                  <button
                    aria-selected={projectDetailTab === "summary"}
                    className={projectDetailTab === "summary" ? "active" : ""}
                    onClick={() => onProjectDetailTabChange("summary")}
                    role="tab"
                    type="button"
                  >
                    Summary
                  </button>
                  <button
                    aria-selected={projectDetailTab === "members"}
                    className={projectDetailTab === "members" ? "active" : ""}
                    onClick={() => onProjectDetailTabChange("members")}
                    role="tab"
                    type="button"
                  >
                    Members
                  </button>
                </div>
              ) : null}

              {projectDetailTab === "members" && selectedProjectDetail.can_manage ? (
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
                    <aside className="project-access-note">
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

                  <div className="project-detail-stats">
                    <article>
                      <span>Visible issues</span>
                      <strong>{selectedProjectIssues.length}</strong>
                    </article>
                    <article>
                      <span>Open</span>
                      <strong>{selectedProjectOpenIssues.length}</strong>
                    </article>
                  </div>

                  <div className="project-detail-actions">
                    <button
                      className="small-button"
                      onClick={() => onViewProjectIssues(selectedProjectDetail.id)}
                      type="button"
                    >
                      View project issues
                    </button>
                    <button
                      className="small-button"
                      onClick={() => onOpenProjectBoard(selectedProjectDetail.id)}
                      type="button"
                    >
                      Open project board
                    </button>
                  </div>

                  {selectedProjectIssues.length > 0 ? (
                    <div className="project-detail-issues">
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
                    <div className="comments-empty">
                      No visible issues for this project
                    </div>
                  )}
                </>
              )}
            </>
          ) : (
            <div className="comments-empty">No project selected</div>
          )}
        </aside>
      </div>
    </section>
  );
}
