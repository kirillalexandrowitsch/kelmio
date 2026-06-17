import { ApiError } from "./api.ts";
import { formatDateTime } from "./formatting.ts";
import { isValidEmail, normalizeEmail, normalizeText } from "./validation.ts";

export type PasswordResetCompleteValues = {
  password: string;
  confirmPassword: string;
};

export function validatePasswordResetEmail(email: string) {
  if (!isValidEmail(email)) {
    return "Email is invalid.";
  }

  return "";
}

export function normalizedPasswordResetEmail(email: string) {
  return normalizeEmail(email);
}

export function validatePasswordResetComplete(values: PasswordResetCompleteValues) {
  const password = normalizeText(values.password);
  const confirmPassword = normalizeText(values.confirmPassword);

  if (password.length < 8) {
    return "Password must be at least 8 characters.";
  }
  if (password !== confirmPassword) {
    return "Password confirmation does not match.";
  }

  return "";
}

export function normalizedPasswordResetComplete(values: PasswordResetCompleteValues) {
  return {
    password: normalizeText(values.password),
    confirmPassword: normalizeText(values.confirmPassword),
  };
}

export function passwordResetTokenErrorMessage(error: unknown) {
  if (!(error instanceof ApiError)) {
    return "Could not load password reset link.";
  }

  switch (error.code) {
    case "password_reset_not_found":
      return "Password reset link was not found. Request a new link.";
    case "password_reset_expired":
      return "Password reset link has expired. Request a new link.";
    case "password_reset_used":
      return "Password reset link was already used. Request a new link.";
    case "password_reset_revoked":
      return "Password reset link was revoked. Request a new link.";
    default:
      return error.message;
  }
}

export function passwordResetPreviewText(email: string, expiresAt: string) {
  return `Reset password for ${email}. This link expires at ${formatDateTime(
    expiresAt,
  )}.`;
}
