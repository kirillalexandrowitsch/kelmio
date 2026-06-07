import assert from "node:assert/strict";
import { test } from "vitest";

import { runtimeVersionDisplay } from "./runtime-version.ts";

test("formats full runtime metadata", () => {
  assert.deepEqual(
    runtimeVersionDisplay({
      version: "v3.0.0",
      commit: "abcdef1234567890",
      environment: "production",
      build_time: "2026-06-05T20:00:00Z",
    }),
    {
      version: "v3.0.0",
      commit: "abcdef123456",
      environment: "production",
      buildTime: "2026-06-05T20:00:00Z",
    },
  );
});

test("formats missing build time", () => {
  assert.equal(
    runtimeVersionDisplay({
      version: "development",
      commit: "local",
      environment: "development",
      build_time: null,
    }).buildTime,
    "Not provided",
  );
});

test("formats unknown commit", () => {
  assert.equal(
    runtimeVersionDisplay({
      version: "production",
      commit: "unknown",
      environment: "production",
      build_time: "",
    }).commit,
    "Unknown",
  );
});
