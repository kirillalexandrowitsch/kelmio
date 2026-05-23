const emailPattern = /^[^@\s]+@[^@\s]+\.[^@\s]+$/;
const usernamePattern = /^[a-z0-9][a-z0-9_-]{2,31}$/;
const labelColorPattern = /^#[0-9a-fA-F]{6}$/;

export function normalizeText(value: string) {
  return value.trim();
}

export function normalizeEmail(value: string) {
  return normalizeText(value).toLowerCase();
}

export function normalizeUsername(value: string) {
  return normalizeText(value).toLowerCase();
}

export function normalizeLabelColor(value: string) {
  return normalizeText(value).toLowerCase();
}

export function hasText(value: string) {
  return normalizeText(value) !== "";
}

export function hasMinTrimmedLength(value: string, minLength: number) {
  return normalizeText(value).length >= minLength;
}

export function isValidEmail(value: string) {
  return emailPattern.test(normalizeEmail(value));
}

export function isValidUsername(value: string) {
  return usernamePattern.test(normalizeUsername(value));
}

export function isValidLabelColor(value: string) {
  return labelColorPattern.test(normalizeLabelColor(value));
}
