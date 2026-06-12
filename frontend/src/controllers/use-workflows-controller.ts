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

  return {
    workflowsByProjectId,
    setWorkflowsByProjectId,
    loadingWorkflowProjectIds,
    setLoadingWorkflowProjectIds,
    workflowErrorsByProjectId,
    setWorkflowErrorsByProjectId,
  };
}
