'use client';

// Cursor pagination shared across list pages.
//
// The backend pages with a `continue` token (the _id of the last item on the
// page) plus a `remainingItemCount` (how many matching docs come *after* the
// current page). It does NOT return an absolute total, so we derive one:
//
//   total = (items on previous pages) + (items on this page) + remainingItemCount
//
// Previous pages are always full because the backend only hands back a usable
// `continue` token / non-zero remaining when more docs exist, so we can treat
// the items-before count as `page * pageSize`. That makes `total` exact on
// every page, which is what lets the footer say "1–50 of 200".

import { useCallback, useMemo, useState } from 'react';
import { useApi, type SWRKey, type ListResponse } from '@shared/api';

type Query = Record<string, string | number | boolean | undefined | null>;

export interface PageRange {
  // 1-based index of the first item shown (0 when the page is empty).
  start: number;
  // 1-based index of the last item shown.
  end: number;
  // Exact count of all matching items across every page.
  total: number;
}

// Pure range/total math, split out so it can be unit-tested without React.
export function pageRange(
  page: number,
  pageSize: number,
  itemCount: number,
  remainingItemCount: number,
): PageRange {
  const offset = page * pageSize;
  const remaining = Math.max(remainingItemCount, 0);
  const total = offset + itemCount + remaining;
  return {
    start: itemCount === 0 ? 0 : offset + 1,
    end: offset + itemCount,
    total,
  };
}

export interface CursorPagination<T> {
  items: T[];
  isLoading: boolean;
  isValidating: boolean;
  error: unknown;
  page: number;
  pageSize: number;
  range: PageRange;
  canPrev: boolean;
  canNext: boolean;
  next: () => void;
  prev: () => void;
  setPageSize: (size: number) => void;
  /** Revalidate the current page (used by the Refresh button). */
  refresh: () => void;
  /** Revalidate the current page and reset to page 0 (used after mutations). */
  reset: () => void;
}

export interface CursorPaginationOptions {
  query?: Query;
  initialPageSize?: number;
  /** Disable fetching entirely (e.g. while a dependency isn't ready). */
  enabled?: boolean;
}

const DEFAULT_PAGE_SIZE = 50;

export function useCursorPagination<T>(
  path: string,
  options: CursorPaginationOptions = {},
): CursorPagination<T> {
  const { query, initialPageSize = DEFAULT_PAGE_SIZE, enabled = true } = options;

  const [pageSize, setPageSizeState] = useState(initialPageSize);
  // tokens[i] is the `continue` token used to fetch page i. tokens[0] is
  // undefined (the first page has no cursor).
  const [tokens, setTokens] = useState<(string | undefined)[]>([undefined]);
  const [page, setPage] = useState(0);

  // Serialised identity of everything that should send us back to page 0:
  // the path, the caller's filters, and the page size (the total math assumes
  // a consistent page size across the visited pages). When it changes we reset
  // during render — React's "adjust state when a prop changes" pattern — rather
  // than in an effect, so the new page-0 cursor is used on this same render.
  const queryKey = JSON.stringify(query ?? {});
  const resetKey = `${path}|${queryKey}|${pageSize}`;
  const [prevResetKey, setPrevResetKey] = useState(resetKey);

  let activeTokens = tokens;
  let activePage = page;
  if (resetKey !== prevResetKey) {
    setPrevResetKey(resetKey);
    setTokens([undefined]);
    setPage(0);
    activeTokens = [undefined];
    activePage = 0;
  }

  const token = activeTokens[activePage];
  const key: SWRKey | null = enabled
    ? [path, { limit: pageSize, continue: token, ...query }]
    : null;

  // We deliberately do NOT keep the previous page's data while a new page
  // loads. With cursor pagination `data` must always match the requested page:
  // if it lagged, the "X–Y of N" range (derived from `activePage`) would not
  // match the rows on screen, and Next would stay clickable on a stale cursor.
  // While a fresh page loads `data` is undefined → empty items, remaining 0,
  // canNext false — so the footer/table stay consistent (callers show a
  // spinner on `isLoading`).
  const { data, error, isLoading, isValidating, mutate } = useApi<ListResponse<T>>(key);

  const items = useMemo(() => data?.items ?? [], [data]);
  const remaining = data?.metadata?.remainingItemCount ?? 0;
  const continueToken = data?.metadata?.continue || '';

  const range = pageRange(activePage, pageSize, items.length, remaining);
  const canPrev = activePage > 0 && !isLoading;
  const canNext = remaining > 0 && continueToken !== '';

  const next = useCallback(() => {
    if (remaining <= 0 || continueToken === '') return;
    setTokens((prev) => {
      const copy = prev.slice(0, activePage + 1);
      copy[activePage + 1] = continueToken;
      return copy;
    });
    setPage((p) => p + 1);
  }, [remaining, continueToken, activePage]);

  const prev = useCallback(() => {
    setPage((p) => Math.max(0, p - 1));
  }, []);

  const setPageSize = useCallback((size: number) => {
    setPageSizeState(size);
  }, []);

  const refresh = useCallback(() => {
    void mutate();
  }, [mutate]);

  const reset = useCallback(() => {
    setTokens([undefined]);
    setPage(0);
    void mutate();
  }, [mutate]);

  return {
    items,
    isLoading,
    isValidating,
    error,
    page: activePage,
    pageSize,
    range,
    canPrev,
    canNext,
    next,
    prev,
    setPageSize,
    refresh,
    reset,
  };
}
