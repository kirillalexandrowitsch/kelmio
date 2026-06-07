import assert from "node:assert/strict";
import { test } from "vitest";

import { appendPaginationParams } from "./pagination.ts";

test("appends pagination query params when provided", () => {
  const params = new URLSearchParams();

  appendPaginationParams(params, {
    limit: 25,
    cursor: "cursor-token",
  });

  assert.equal(params.get("limit"), "25");
  assert.equal(params.get("cursor"), "cursor-token");
});

test("leaves query params unchanged when pagination is empty", () => {
  const params = new URLSearchParams("status=todo");

  appendPaginationParams(params);

  assert.equal(params.toString(), "status=todo");
});
