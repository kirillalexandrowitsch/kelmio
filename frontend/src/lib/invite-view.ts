import type {
  TeamInviteEmailDeliveryStatus,
  TeamInviteStatus,
} from "./api-types.ts";
import {
  isValidEmail,
  isValidUsername,
  normalizeEmail,
  normalizeText,
  normalizeUsername,
} from "./validation.ts";

export type InviteAcceptFormValues = {
  username: string;
  displayName: string;
  password: string;
  confirmPassword: string;
};

export function inviteStatusLabel(status: TeamInviteStatus) {
  switch (status) {
    case "pending":
      return "Pending";
    case "accepted":
      return "Accepted";
    case "revoked":
      return "Revoked";
    case "expired":
      return "Expired";
  }
}

export function inviteDeliveryStatusLabel(status: TeamInviteEmailDeliveryStatus) {
  switch (status) {
    case "not_sent":
      return "Not sent";
    case "pending":
      return "Pending";
    case "processing":
      return "Processing";
    case "sent":
      return "Sent";
    case "failed":
      return "Failed";
  }
}

export function buildInviteAcceptURL(path: string, origin = "") {
  if (!path) {
    return "";
  }

  return `${origin.replace(/\/$/, "")}${path}`;
}

export function validateInviteEmail(email: string) {
  if (!isValidEmail(email)) {
    return "Email is invalid.";
  }

  return "";
}

export function validateInviteAcceptForm(values: InviteAcceptFormValues) {
  const username = normalizeUsername(values.username);
  const displayName = normalizeText(values.displayName);
  const password = normalizeText(values.password);
  const confirmPassword = normalizeText(values.confirmPassword);

  if (!isValidUsername(username)) {
    return "Username must be 3-32 characters and contain lowercase letters, numbers, underscores, or hyphens.";
  }
  if (!displayName) {
    return "Display name is required.";
  }
  if (password.length < 8) {
    return "Password must be at least 8 characters.";
  }
  if (password !== confirmPassword) {
    return "Password confirmation does not match.";
  }

  return "";
}

export function normalizedInviteAcceptInput(values: InviteAcceptFormValues) {
  return {
    username: normalizeUsername(values.username),
    display_name: normalizeText(values.displayName),
    password: normalizeText(values.password),
  };
}

export function normalizedInviteEmail(email: string) {
  return normalizeEmail(email);
}
