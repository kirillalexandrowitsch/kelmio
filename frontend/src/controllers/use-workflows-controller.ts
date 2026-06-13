import { useState } from "react";

import type { ProjectWorkflow } from "../lib/api";

export function useWorkflowsController() {
  const [workflowsByProjectId, setWorkflowsByProjectId] = useState<
    Record<string, ProjectWorkflow>
  >({});
  const [loadingWorkflowProjectIds, setLoadingWorkflowProjectIds] = useState<
    string[]
  >([]);
  const [workflowErrorsByProjectId, setWorkflowErrorsByProjectId] = useState<
    Record<string, string>
  >({});
  const [workflowMutationError, setWorkflowMutationError] = useState("");
  const [creatingWorkflowStatus, setCreatingWorkflowStatus] = useState(false);
  const [updatingWorkflowStatusIds, setUpdatingWorkflowStatusIds] = useState<
    string[]
  >([]);
  const [archivingWorkflowStatusIds, setArchivingWorkflowStatusIds] = useState<
    string[]
  >([]);
  const [isReorderingWorkflowStatuses, setIsReorderingWorkflowStatuses] =
    useState(false);
  const [isSavingWorkflowTransitions, setIsSavingWorkflowTransitions] =
    useState(false);

  return {
    workflowsByProjectId,
    setWorkflowsByProjectId,
    loadingWorkflowProjectIds,
    setLoadingWorkflowProjectIds,
    workflowErrorsByProjectId,
    setWorkflowErrorsByProjectId,
    workflowMutationError,
    setWorkflowMutationError,
    creatingWorkflowStatus,
    setCreatingWorkflowStatus,
    updatingWorkflowStatusIds,
    setUpdatingWorkflowStatusIds,
    archivingWorkflowStatusIds,
    setArchivingWorkflowStatusIds,
    isReorderingWorkflowStatuses,
    setIsReorderingWorkflowStatuses,
    isSavingWorkflowTransitions,
    setIsSavingWorkflowTransitions,
  };
}
