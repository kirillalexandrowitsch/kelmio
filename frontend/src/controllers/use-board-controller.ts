import { useState } from "react";

import type { Issue } from "../lib/api";

export function useBoardController(initialProjectId: string) {
  const [boardProjectId, setBoardProjectId] = useState(initialProjectId);
  const [boardIssues, setBoardIssues] = useState<Issue[]>([]);
  const [boardError, setBoardError] = useState("");
  const [isLoadingBoard, setIsLoadingBoard] = useState(false);

  return {
    boardProjectId,
    setBoardProjectId,
    boardIssues,
    setBoardIssues,
    boardError,
    setBoardError,
    isLoadingBoard,
    setIsLoadingBoard,
  };
}
