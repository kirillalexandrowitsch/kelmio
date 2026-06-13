import type { PaginationParams } from "./api-types";

export type PaginatedPage<T> = {
  items: T[];
  nextCursor?: string | null;
};

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

export async function collectPaginatedItems<T>(
  loadPage: (cursor?: string) => Promise<PaginatedPage<T>>,
) {
  const items: T[] = [];
  let cursor: string | undefined;

  do {
    const page = await loadPage(cursor);
    items.push(...page.items);
    cursor = page.nextCursor ?? undefined;
  } while (cursor);

  return items;
}
