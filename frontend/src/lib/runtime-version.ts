import type { RuntimeVersion } from "./api-types.ts";

const notProvided = "Not provided";
const unknown = "Unknown";

export type RuntimeVersionDisplay = {
  version: string;
  commit: string;
  environment: string;
  buildTime: string;
};

export function runtimeVersionDisplay(
  runtimeVersion: RuntimeVersion | null,
): RuntimeVersionDisplay {
  return {
    version: displayValue(runtimeVersion?.version),
    commit: displayCommit(runtimeVersion?.commit),
    environment: displayValue(runtimeVersion?.environment),
    buildTime: displayValue(runtimeVersion?.build_time),
  };
}

export function displayValue(value: string | null | undefined) {
  const normalizedValue = value?.trim() ?? "";
  return normalizedValue || notProvided;
}

export function displayCommit(value: string | null | undefined) {
  const normalizedValue = value?.trim() ?? "";
  if (!normalizedValue || normalizedValue.toLowerCase() === "unknown") {
    return unknown;
  }

  return normalizedValue.length > 12
    ? normalizedValue.slice(0, 12)
    : normalizedValue;
}
