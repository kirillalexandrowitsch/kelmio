import { useState } from "react";
import type {
  Label,
  Project,
  ProjectMember,
  ProjectRole,
  TeamInvite,
  TeamMember,
} from "../lib/api";

export function useWorkspaceAdminController() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [projectsError, setProjectsError] = useState("");
  const [projectFormError, setProjectFormError] = useState("");
  const [isLoadingProjects, setIsLoadingProjects] = useState(true);
  const [isCreatingProject, setIsCreatingProject] = useState(false);
  const [archivingProjectIds, setArchivingProjectIds] = useState<string[]>([]);
  const [editingProjectId, setEditingProjectId] = useState("");
  const [editProjectName, setEditProjectName] = useState("");
  const [editProjectDescription, setEditProjectDescription] = useState("");
  const [updatingProjectIds, setUpdatingProjectIds] = useState<string[]>([]);
  const [selectedProjectDetail, setSelectedProjectDetail] =
    useState<Project | null>(null);
  const [projectDetailError, setProjectDetailError] = useState("");
  const [isLoadingProjectDetail, setIsLoadingProjectDetail] = useState(false);
  const [projectDetailTab, setProjectDetailTab] = useState<
    "summary" | "members" | "workflow"
  >("summary");
  const [projectMembers, setProjectMembers] = useState<ProjectMember[]>([]);
  const [projectMembersError, setProjectMembersError] = useState("");
  const [isLoadingProjectMembers, setIsLoadingProjectMembers] = useState(false);
  const [selectedProjectMemberUserId, setSelectedProjectMemberUserId] =
    useState("");
  const [selectedProjectMemberRole, setSelectedProjectMemberRole] =
    useState<ProjectRole>("contributor");
  const [updatingProjectMemberIds, setUpdatingProjectMemberIds] = useState<
    string[]
  >([]);
  const [removingProjectMemberIds, setRemovingProjectMemberIds] = useState<
    string[]
  >([]);
  const [projectKey, setProjectKey] = useState("");
  const [projectName, setProjectName] = useState("");
  const [projectDescription, setProjectDescription] = useState("");
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>([]);
  const [teamMembersError, setTeamMembersError] = useState("");
  const [teamMemberFormError, setTeamMemberFormError] = useState("");
  const [isLoadingTeamMembers, setIsLoadingTeamMembers] = useState(false);
  const [isCreatingTeamMember, setIsCreatingTeamMember] = useState(false);
  const [teamMemberEmail, setTeamMemberEmail] = useState("");
  const [teamMemberUsername, setTeamMemberUsername] = useState("");
  const [teamMemberDisplayName, setTeamMemberDisplayName] = useState("");
  const [teamMemberPassword, setTeamMemberPassword] = useState("");
  const [teamMemberRole, setTeamMemberRole] =
    useState<TeamMember["role"]>("member");
  const [teamInvites, setTeamInvites] = useState<TeamInvite[]>([]);
  const [teamInvitesError, setTeamInvitesError] = useState("");
  const [teamInviteFormError, setTeamInviteFormError] = useState("");
  const [isLoadingTeamInvites, setIsLoadingTeamInvites] = useState(false);
  const [isCreatingTeamInvite, setIsCreatingTeamInvite] = useState(false);
  const [teamInviteEmail, setTeamInviteEmail] = useState("");
  const [teamInviteRole, setTeamInviteRole] =
    useState<TeamMember["role"]>("member");
  const [teamInviteLinksById, setTeamInviteLinksById] = useState<
    Record<string, string>
  >({});
  const [copiedTeamInviteId, setCopiedTeamInviteId] = useState("");
  const [revokingTeamInviteIds, setRevokingTeamInviteIds] = useState<string[]>([]);
  const [updatingTeamMemberIds, setUpdatingTeamMemberIds] = useState<string[]>([]);
  const [passwordResetMemberId, setPasswordResetMemberId] = useState("");
  const [teamMemberResetPassword, setTeamMemberResetPassword] = useState("");
  const [resettingTeamMemberPasswordIds, setResettingTeamMemberPasswordIds] =
    useState<string[]>([]);
  const [labels, setLabels] = useState<Label[]>([]);
  const [labelsError, setLabelsError] = useState("");
  const [isLoadingLabels, setIsLoadingLabels] = useState(false);
  const [labelName, setLabelName] = useState("");
  const [labelColor, setLabelColor] = useState("#4e795d");
  const [isCreatingLabel, setIsCreatingLabel] = useState(false);
  const [deletingLabelIds, setDeletingLabelIds] = useState<string[]>([]);

  return {
    projects,
    setProjects,
    projectsError,
    setProjectsError,
    projectFormError,
    setProjectFormError,
    isLoadingProjects,
    setIsLoadingProjects,
    isCreatingProject,
    setIsCreatingProject,
    archivingProjectIds,
    setArchivingProjectIds,
    editingProjectId,
    setEditingProjectId,
    editProjectName,
    setEditProjectName,
    editProjectDescription,
    setEditProjectDescription,
    updatingProjectIds,
    setUpdatingProjectIds,
    selectedProjectDetail,
    setSelectedProjectDetail,
    projectDetailError,
    setProjectDetailError,
    isLoadingProjectDetail,
    setIsLoadingProjectDetail,
    projectDetailTab,
    setProjectDetailTab,
    projectMembers,
    setProjectMembers,
    projectMembersError,
    setProjectMembersError,
    isLoadingProjectMembers,
    setIsLoadingProjectMembers,
    selectedProjectMemberUserId,
    setSelectedProjectMemberUserId,
    selectedProjectMemberRole,
    setSelectedProjectMemberRole,
    updatingProjectMemberIds,
    setUpdatingProjectMemberIds,
    removingProjectMemberIds,
    setRemovingProjectMemberIds,
    projectKey,
    setProjectKey,
    projectName,
    setProjectName,
    projectDescription,
    setProjectDescription,
    teamMembers,
    setTeamMembers,
    teamMembersError,
    setTeamMembersError,
    teamMemberFormError,
    setTeamMemberFormError,
    isLoadingTeamMembers,
    setIsLoadingTeamMembers,
    isCreatingTeamMember,
    setIsCreatingTeamMember,
    teamMemberEmail,
    setTeamMemberEmail,
    teamMemberUsername,
    setTeamMemberUsername,
    teamMemberDisplayName,
    setTeamMemberDisplayName,
    teamMemberPassword,
    setTeamMemberPassword,
    teamMemberRole,
    setTeamMemberRole,
    teamInvites,
    setTeamInvites,
    teamInvitesError,
    setTeamInvitesError,
    teamInviteFormError,
    setTeamInviteFormError,
    isLoadingTeamInvites,
    setIsLoadingTeamInvites,
    isCreatingTeamInvite,
    setIsCreatingTeamInvite,
    teamInviteEmail,
    setTeamInviteEmail,
    teamInviteRole,
    setTeamInviteRole,
    teamInviteLinksById,
    setTeamInviteLinksById,
    copiedTeamInviteId,
    setCopiedTeamInviteId,
    revokingTeamInviteIds,
    setRevokingTeamInviteIds,
    updatingTeamMemberIds,
    setUpdatingTeamMemberIds,
    passwordResetMemberId,
    setPasswordResetMemberId,
    teamMemberResetPassword,
    setTeamMemberResetPassword,
    resettingTeamMemberPasswordIds,
    setResettingTeamMemberPasswordIds,
    labels,
    setLabels,
    labelsError,
    setLabelsError,
    isLoadingLabels,
    setIsLoadingLabels,
    labelName,
    setLabelName,
    labelColor,
    setLabelColor,
    isCreatingLabel,
    setIsCreatingLabel,
    deletingLabelIds,
    setDeletingLabelIds,
  };
}
