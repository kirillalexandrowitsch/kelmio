import assert from "node:assert/strict";
import { test } from "vitest";

import { appendPaginationParams, collectPaginatedItems } from "./pagination.ts";

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

test("collects every page in cursor order", async () => {
  const cursors: Array<string | undefined> = [];
  const items = await collectPaginatedItems(async (cursor) => {
    cursors.push(cursor);
    if (!cursor) {
      return { items: ["one", "two"], nextCursor: "page-2" };
    }

    return { items: ["three"], nextCursor: null };
  });

  assert.deepEqual(items, ["one", "two", "three"]);
  assert.deepEqual(cursors, [undefined, "page-2"]);
});
