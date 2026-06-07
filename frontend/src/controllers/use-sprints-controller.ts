import { useState } from "react";
import type { Issue, Sprint, SprintStatus } from "../lib/api";

export function useSprintsController() {
  const [sprints, setSprints] = useState<Sprint[]>([]);
  const [issueFilterSprints, setIssueFilterSprints] = useState<Sprint[]>([]);
  const [dashboardSprintIssues, setDashboardSprintIssues] = useState<Issue[]>([]);
  const [dashboardSprintError, setDashboardSprintError] = useState("");
  const [isLoadingDashboardSprint, setIsLoadingDashboardSprint] = useState(false);
  const [sprintsError, setSprintsError] = useState("");
  const [sprintFormError, setSprintFormError] = useState("");
  const [isLoadingSprints, setIsLoadingSprints] = useState(false);
  const [isCreatingSprint, setIsCreatingSprint] = useState(false);
  const [sprintProjectId, setSprintProjectId] = useState("");
  const [sprintName, setSprintName] = useState("");
  const [sprintGoal, setSprintGoal] = useState("");
  const [sprintStartDate, setSprintStartDate] = useState("");
  const [sprintEndDate, setSprintEndDate] = useState("");
  const [sprintFilterProjectId, setSprintFilterProjectId] = useState("");
  const [sprintFilterStatus, setSprintFilterStatus] = useState<
    SprintStatus | ""
  >("");
  const [selectedSprint, setSelectedSprint] = useState<Sprint | null>(null);
  const [selectedSprintError, setSelectedSprintError] = useState("");
  const [isLoadingSelectedSprint, setIsLoadingSelectedSprint] = useState(false);
  const [isEditingSprintDetails, setIsEditingSprintDetails] = useState(false);
  const [isUpdatingSprint, setIsUpdatingSprint] = useState(false);
  const [editSprintName, setEditSprintName] = useState("");
  const [editSprintGoal, setEditSprintGoal] = useState("");
  const [editSprintStartDate, setEditSprintStartDate] = useState("");
  const [editSprintEndDate, setEditSprintEndDate] = useState("");
  const [startingSprintIds, setStartingSprintIds] = useState<string[]>([]);
  const [completingSprintIds, setCompletingSprintIds] = useState<string[]>([]);
  const [sprintPlanningIssues, setSprintPlanningIssues] = useState<Issue[]>([]);
  const [sprintPlanningError, setSprintPlanningError] = useState("");
  const [isLoadingSprintPlanning, setIsLoadingSprintPlanning] = useState(false);
  const [addingIssueToSprintIds, setAddingIssueToSprintIds] = useState<string[]>(
    [],
  );
  const [removingIssueFromSprintIds, setRemovingIssueFromSprintIds] = useState<
    string[]
  >([]);

  return {
    sprints, setSprints, issueFilterSprints, setIssueFilterSprints,
    dashboardSprintIssues, setDashboardSprintIssues, dashboardSprintError,
    setDashboardSprintError, isLoadingDashboardSprint,
    setIsLoadingDashboardSprint, sprintsError, setSprintsError, sprintFormError,
    setSprintFormError, isLoadingSprints, setIsLoadingSprints, isCreatingSprint,
    setIsCreatingSprint, sprintProjectId, setSprintProjectId, sprintName,
    setSprintName, sprintGoal, setSprintGoal, sprintStartDate,
    setSprintStartDate, sprintEndDate, setSprintEndDate, sprintFilterProjectId,
    setSprintFilterProjectId, sprintFilterStatus, setSprintFilterStatus,
    selectedSprint, setSelectedSprint, selectedSprintError,
    setSelectedSprintError, isLoadingSelectedSprint, setIsLoadingSelectedSprint,
    isEditingSprintDetails, setIsEditingSprintDetails, isUpdatingSprint,
    setIsUpdatingSprint, editSprintName, setEditSprintName, editSprintGoal,
    setEditSprintGoal, editSprintStartDate, setEditSprintStartDate,
    editSprintEndDate, setEditSprintEndDate, startingSprintIds,
    setStartingSprintIds, completingSprintIds, setCompletingSprintIds,
    sprintPlanningIssues, setSprintPlanningIssues, sprintPlanningError,
    setSprintPlanningError, isLoadingSprintPlanning, setIsLoadingSprintPlanning,
    addingIssueToSprintIds, setAddingIssueToSprintIds, removingIssueFromSprintIds,
    setRemovingIssueFromSprintIds,
  };
}
