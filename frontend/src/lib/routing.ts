export type AppSection =
  | "dashboard"
  | "projects"
  | "issues"
  | "board"
  | "sprints"
  | "notifications"
  | "team"
  | "labels"
  | "administration"
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
  { id: "administration", title: "Administration" },
  { id: "account", title: "Account" },
] satisfies Array<{ id: AppSection; title: string }>;

// Sections that are only available to site administrators. The navigation,
// command palette and section rendering hide these for everyone else.
const siteAdminSections = new Set<AppSection>(["administration"]);

export function isSiteAdminSection(section: AppSection) {
  return siteAdminSections.has(section);
}

export type NavGroup = {
  id: string;
  label: string;
  sections: AppSection[];
};

// Sidebar navigation groups. "account" is intentionally excluded; it is reached
// through the user card in the sidebar footer rather than a nav entry.
export const navGroups: NavGroup[] = [
  {
    id: "workspace",
    label: "Workspace",
    sections: ["dashboard", "projects", "issues", "board", "sprints"],
  },
  {
    id: "organization",
    label: "Organization",
    sections: ["notifications", "team", "labels"],
  },
  {
    id: "administration",
    label: "Administration",
    sections: ["administration"],
  },
];

const appSectionPaths: Record<AppSection, string> = {
  dashboard: "/",
  projects: "/projects",
  issues: "/issues",
  board: "/board",
  sprints: "/sprints",
  notifications: "/notifications",
  team: "/team",
  labels: "/labels",
  administration: "/administration",
  account: "/account",
};

const inviteAcceptPath = "/accept-invite";
const forgotPasswordPathname = "/forgot-password";
const passwordResetPathname = "/reset-password";
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

export function isForgotPasswordRoute(pathname: string) {
  return pathname === forgotPasswordPathname;
}

export function isPasswordResetRoute(pathname: string) {
  return pathname === passwordResetPathname;
}

export function inviteAcceptTokenFromLocation(
  location: Pick<Location, "pathname" | "search">,
) {
  if (!isInviteAcceptRoute(location.pathname)) {
    return null;
  }

  return new URLSearchParams(location.search).get("token")?.trim() ?? "";
}

export function forgotPasswordRouteFromLocation(
  location: Pick<Location, "pathname">,
) {
  return isForgotPasswordRoute(location.pathname);
}

export function passwordResetTokenFromLocation(
  location: Pick<Location, "pathname" | "search">,
) {
  if (!isPasswordResetRoute(location.pathname)) {
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

export function currentForgotPasswordRouteFromLocation(
  location: Pick<Location, "pathname"> | undefined = undefined,
) {
  if (location) {
    return forgotPasswordRouteFromLocation(location);
  }

  if (typeof window === "undefined") {
    return false;
  }

  return forgotPasswordRouteFromLocation(window.location);
}

export function currentPasswordResetTokenFromLocation(
  location: Pick<Location, "pathname" | "search"> | undefined = undefined,
) {
  if (location) {
    return passwordResetTokenFromLocation(location);
  }

  if (typeof window === "undefined") {
    return null;
  }

  return passwordResetTokenFromLocation(window.location);
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
