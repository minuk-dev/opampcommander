'use client';

import { TablePagination } from '@mui/material';
import type { CursorPagination } from '@shared/lib';

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

// Footer that renders MUI's "start–end of total" label with prev/next arrows
// and a rows-per-page selector, driven by cursor pagination. The arrows step
// one page at a time (cursor pagination can't jump to an arbitrary page), so
// we map MUI's page delta onto next()/prev() and disable each side using our
// own canNext/canPrev rather than MUI's count-derived bounds.
export default function PaginationFooter({
  pagination,
  rowsPerPageOptions = DEFAULT_OPTIONS,
}: Props) {
  const { page, pageSize, range, canPrev, canNext, next, prev, setPageSize, isLoading } =
    pagination;

  // While a fresh page is loading `range` is derived from an empty result, so
  // its numbers are meaningless — show an ellipsis instead. `count` is clamped
  // so `page` is never out of MUI's range (which would log a dev warning); the
  // visible "of N" comes from `range.total`, not this value.
  const count = Math.max(range.total, (page + 1) * pageSize);

  return (
    <TablePagination
      component="div"
      count={count}
      page={page}
      rowsPerPage={pageSize}
      rowsPerPageOptions={rowsPerPageOptions}
      labelDisplayedRows={() =>
        isLoading
          ? '…'
          : range.total === 0
            ? '0 of 0'
            : `${range.start}–${range.end} of ${range.total}`
      }
      onPageChange={(_, newPage) => {
        if (newPage > page) next();
        else prev();
      }}
      onRowsPerPageChange={(e) => setPageSize(parseInt(e.target.value, 10))}
      slotProps={{
        actions: {
          previousButton: { disabled: !canPrev },
          nextButton: { disabled: !canNext },
        },
      }}
    />
  );
}
