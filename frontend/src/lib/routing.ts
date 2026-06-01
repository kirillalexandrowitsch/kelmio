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

export function appSectionPath(section: AppSection) {
  return appSectionPaths[section];
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
