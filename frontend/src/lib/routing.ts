export type AppSection =
  | "dashboard"
  | "projects"
  | "issues"
  | "board"
  | "sprints"
  | "notifications"
  | "team"
  | "labels"
  | "account";

export const appSections = [
  { id: "dashboard", title: "Dashboard" },
  { id: "projects", title: "Projects" },
  { id: "issues", title: "Issues" },
  { id: "board", title: "Board" },
  { id: "sprints", title: "Sprints" },
  { id: "notifications", title: "Notifications" },
  { id: "team", title: "Team" },
  { id: "labels", title: "Labels" },
  { id: "account", title: "Account" },
] satisfies Array<{ id: AppSection; title: string }>;

const appSectionPaths: Record<AppSection, string> = {
  dashboard: "/",
  projects: "/projects",
  issues: "/issues",
  board: "/board",
  sprints: "/sprints",
  notifications: "/notifications",
  team: "/team",
  labels: "/labels",
  account: "/account",
};

const inviteAcceptPath = "/accept-invite";
const boardPathname = "/board";

export function appSectionPath(section: AppSection) {
  return appSectionPaths[section];
}

export function boardPath(projectId = "") {
  if (!projectId) {
    return boardPathname;
  }

  return `${boardPathname}?projectId=${encodeURIComponent(projectId)}`;
}

export function appSectionFromPath(pathname: string): AppSection {
  const matchingSection = appSections.find(
    (section) => appSectionPath(section.id) === pathname,
  );

  if (!matchingSection && pathname.startsWith("/sprints/")) {
    return "sprints";
  }

  return matchingSection?.id ?? "dashboard";
}

export function isInviteAcceptRoute(pathname: string) {
  return pathname === inviteAcceptPath;
}

export function inviteAcceptTokenFromLocation(
  location: Pick<Location, "pathname" | "search">,
) {
  if (!isInviteAcceptRoute(location.pathname)) {
    return null;
  }

  return new URLSearchParams(location.search).get("token")?.trim() ?? "";
}

export function boardProjectIdFromLocation(
  location: Pick<Location, "pathname" | "search">,
) {
  if (location.pathname !== boardPathname) {
    return "";
  }

  return new URLSearchParams(location.search).get("projectId")?.trim() ?? "";
}

export function currentBoardProjectIdFromLocation(
  location: Pick<Location, "pathname" | "search"> | undefined = undefined,
) {
  if (location) {
    return boardProjectIdFromLocation(location);
  }

  if (typeof window === "undefined") {
    return "";
  }

  return boardProjectIdFromLocation(window.location);
}

export function currentInviteAcceptTokenFromLocation(
  location: Pick<Location, "pathname" | "search"> | undefined = undefined,
) {
  if (location) {
    return inviteAcceptTokenFromLocation(location);
  }

  if (typeof window === "undefined") {
    return null;
  }

  return inviteAcceptTokenFromLocation(window.location);
}

export function sprintIdFromPath(pathname: string) {
  const match = pathname.match(/^\/sprints\/([^/]+)$/);
  return match ? decodeURIComponent(match[1]) : "";
}

export function currentAppSectionFromLocation(
  location: Pick<Location, "pathname"> | undefined = undefined,
) {
  if (location) {
    return appSectionFromPath(location.pathname);
  }

  if (typeof window === "undefined") {
    return "dashboard";
  }

  return appSectionFromPath(window.location.pathname);
}
