import { useState } from "react";

import type { AutomationRule } from "../lib/api";

export function useAutomationsController() {
  const [automationRules, setAutomationRules] = useState<AutomationRule[]>([]);
  const [automationRulesError, setAutomationRulesError] = useState("");
  const [isLoadingAutomationRules, setIsLoadingAutomationRules] = useState(false);
  const [isCreatingAutomationRule, setIsCreatingAutomationRule] = useState(false);
  const [updatingAutomationRuleIds, setUpdatingAutomationRuleIds] = useState<
    string[]
  >([]);
  const [deletingAutomationRuleIds, setDeletingAutomationRuleIds] = useState<
    string[]
  >([]);
  const [isReorderingAutomationRules, setIsReorderingAutomationRules] =
    useState(false);

  return {
    automationRules,
    setAutomationRules,
    automationRulesError,
    setAutomationRulesError,
    isLoadingAutomationRules,
    setIsLoadingAutomationRules,
    isCreatingAutomationRule,
    setIsCreatingAutomationRule,
    updatingAutomationRuleIds,
    setUpdatingAutomationRuleIds,
    deletingAutomationRuleIds,
    setDeletingAutomationRuleIds,
    isReorderingAutomationRules,
    setIsReorderingAutomationRules,
  };
}
