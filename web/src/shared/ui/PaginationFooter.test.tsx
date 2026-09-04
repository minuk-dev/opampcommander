import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import type { CursorPagination } from '@shared/lib';
import PaginationFooter from './PaginationFooter';

function pagination(
  overrides: Partial<CursorPagination<unknown>> = {},
): React.ComponentProps<typeof PaginationFooter>['pagination'] {
  return {
    page: 0,
    pageSize: 50,
    range: { start: 1, end: 50, total: 200 },
    canPrev: false,
    canNext: true,
    next: vi.fn(),
    prev: vi.fn(),
    setPageSize: vi.fn(),
    isLoading: false,
    ...overrides,
  };
}

describe('PaginationFooter', () => {
  it('reports the server range when nothing was filtered locally', () => {
    render(<PaginationFooter pagination={pagination()} />);
    expect(screen.getByText('1–50 of 200')).toBeInTheDocument();
  });

  // A client-side pass narrows the fetched page, but the server never applied
  // that filter — so "of 200" would count a different set than the rows on
  // screen. The footer reports only what it can know.
  it('drops the unknowable total when rows were filtered client-side', () => {
    render(<PaginationFooter pagination={pagination()} clientFilteredCount={3} />);
    expect(screen.getByText('3 of 50 on this page')).toBeInTheDocument();
    expect(screen.queryByText(/of 200/)).not.toBeInTheDocument();
  });

  it('still shows an ellipsis while a page is loading', () => {
    render(
      <PaginationFooter pagination={pagination({ isLoading: true })} clientFilteredCount={3} />,
    );
    expect(screen.getByText('…')).toBeInTheDocument();
  });
});
