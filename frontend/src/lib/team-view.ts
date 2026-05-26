import { type TeamMember } from "./api-types.ts";

export function memberInitials(displayName: string) {
  const initials = displayName
    .trim()
    .split(/\s+/)
    .map((part) => part[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();

  return initials || "TM";
}

export function memberDisplayName(members: TeamMember[], memberId: string | null) {
  if (!memberId) {
    return "Unassigned";
  }

  return members.find((member) => member.id === memberId)?.display_name ?? memberId;
}

export function memberOptionLabel(member: TeamMember) {
  return member.is_active ? member.display_name : `${member.display_name} (inactive)`;
}

export function activeTeamMembers(members: TeamMember[]) {
  return members.filter((member) => member.is_active);
}

export function assignableTeamMembers(
  members: TeamMember[],
  currentAssigneeId: string | null,
) {
  return members.filter(
    (member) => member.is_active || member.id === currentAssigneeId,
  );
}
