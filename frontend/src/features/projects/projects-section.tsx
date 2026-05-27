import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import { type CurrentUser, type Issue, type Project } from "../../lib/api-types";
import { PROJECT_PERMISSION_NOTE } from "../../lib/permissions";
import { hasText } from "../../lib/validation";

type ProjectsSectionProps = {
  archivingProjectIds: string[];
  canCreateProject: boolean;
  editProjectDescription: string;
  editProjectName: string;
  editingProjectId: string;
  isActive: boolean;
  isCreatingProject: boolean;
  isLoadingProjectDetail: boolean;
  isLoadingProjects: boolean;
  onArchiveProject: (project: Project) => void;
  onCancelEditingProject: () => void;
  onCreateProject: (event: FormEvent<HTMLFormElement>) => void;
  onEditProjectDescriptionChange: (value: string) => void;
  onEditProjectNameChange: (value: string) => void;
  onOpenProjectBoard: (projectId: string) => void;
  onProjectDescriptionChange: (value: string) => void;
  onProjectKeyChange: (value: string) => void;
  onProjectNameChange: (value: string) => void;
  onSelectIssue: (issueId: string) => void;
  onSelectProjectDetail: (projectId: string) => void;
  onStartEditingProject: (project: Project) => void;
  onUpdateProject: (event: FormEvent<HTMLFormElement>, project: Project) => void;
  onViewProjectIssues: (projectId: string) => void;
  projectDescription: string;
  projectDetailError: string;
  projectFormError: string;
  projectKey: string;
  projectName: string;
  projects: Project[];
  projectsError: string;
  role: CurrentUser["workspace"]["role"];
  selectedProjectDetail: Project | null;
  selectedProjectIssues: Issue[];
  selectedProjectOpenIssues: Issue[];
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
  isLoadingProjects,
  onArchiveProject,
  onCancelEditingProject,
  onCreateProject,
  onEditProjectDescriptionChange,
  onEditProjectNameChange,
  onOpenProjectBoard,
  onProjectDescriptionChange,
  onProjectKeyChange,
  onProjectNameChange,
  onSelectIssue,
  onSelectProjectDetail,
  onStartEditingProject,
  onUpdateProject,
  onViewProjectIssues,
  projectDescription,
  projectDetailError,
  projectFormError,
  projectKey,
  projectName,
  projects,
  projectsError,
  role,
  selectedProjectDetail,
  selectedProjectIssues,
  selectedProjectOpenIssues,
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
                <div className="comments-empty">No visible issues for this project</div>
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
