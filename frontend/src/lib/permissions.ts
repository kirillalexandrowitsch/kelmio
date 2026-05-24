export const PROJECT_PERMISSION_NOTE = {
  eyebrow: "Read-only",
  title: "Project management",
  body: "You can view projects and work with issues. Creating, editing, and archiving projects is limited to workspace admins.",
} as const;

export const TEAM_PERMISSION_NOTE = {
  eyebrow: "Read-only",
  title: "Team management",
  body: "You can view workspace members here. Creating members, editing roles, deactivating accounts, and resetting passwords is limited to workspace admins.",
} as const;

export const DASHBOARD_ACTION_COPY = {
  projects: {
    admin: "Create or review project spaces before adding team work.",
    member: "Review project spaces before creating or updating team work.",
  },
  team: {
    admin: "Create members, update roles, and keep workspace access clean.",
    member: "Review workspace members and ask an admin for access changes.",
  },
} as const;
