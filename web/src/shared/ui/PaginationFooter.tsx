'use client';

import { ChevronLeft, ChevronRight } from 'lucide-react';
import type { CursorPagination } from '@shared/lib';
import Button from './Button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './Select';

interface Props {
  // Accepts the object returned by useCursorPagination (only the fields used
  // here are required, so any element type works).
  pagination: Pick<
    CursorPagination<unknown>,
    | 'page'
    | 'pageSize'
    | 'range'
    | 'canPrev'
    | 'canNext'
    | 'next'
    | 'prev'
    | 'setPageSize'
    | 'isLoading'
  >;
  rowsPerPageOptions?: number[];
}

const DEFAULT_OPTIONS = [25, 50, 100, 200];

// "start–end of total" with prev/next arrows and a rows-per-page selector.
// The arrows step one page at a time because cursor pagination cannot jump to
// an arbitrary page, so each side is disabled from canPrev/canNext rather than
// from a computed page count.
export default function PaginationFooter({
  pagination,
  rowsPerPageOptions = DEFAULT_OPTIONS,
}: Props) {
  const { pageSize, range, canPrev, canNext, next, prev, setPageSize, isLoading } = pagination;

  // While a fresh page is loading, `range` is derived from an empty result and
  // its numbers are meaningless — show an ellipsis instead.
  const label = isLoading
    ? '…'
    : range.total === 0
      ? '0 of 0'
      : `${range.start}–${range.end} of ${range.total}`;

  return (
    <div className="flex flex-wrap items-center justify-end gap-x-4 gap-y-2 px-1 py-2 text-xs text-muted-foreground">
      <div className="flex items-center gap-2">
        <span className="hidden sm:inline">Rows per page</span>
        <Select value={String(pageSize)} onValueChange={(v) => setPageSize(Number(v))}>
          <SelectTrigger className="h-7 w-18 text-xs" aria-label="Rows per page">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {rowsPerPageOptions.map((n) => (
              <SelectItem key={n} value={String(n)}>
                {n}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <span className="tnum">{label}</span>
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label="Previous page"
          disabled={!canPrev}
          onClick={prev}
        >
          <ChevronLeft aria-hidden />
        </Button>
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label="Next page"
          disabled={!canNext}
          onClick={next}
        >
          <ChevronRight aria-hidden />
        </Button>
      </div>
    </div>
  );
}
