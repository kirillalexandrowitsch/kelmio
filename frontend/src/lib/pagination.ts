import type { PaginationParams } from "./api-types";

export function appendPaginationParams(
  params: URLSearchParams,
  pagination: PaginationParams = {},
) {
  if (pagination.limit !== undefined) {
    params.set("limit", String(pagination.limit));
  }
  if (pagination.cursor) {
    params.set("cursor", pagination.cursor);
  }
}
