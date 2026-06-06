import { describe, expect, it } from 'vitest';
import { pageRange } from './pagination';

describe('pageRange', () => {
  it('computes the range and total on the first page', () => {
    // 50 shown, 150 still to come → 200 total, showing 1–50.
    expect(pageRange(0, 50, 50, 150)).toEqual({ start: 1, end: 50, total: 200 });
  });

  it('offsets the range on later pages', () => {
    // Page 2 (0-based), 50 shown, 50 remaining → 200 total, showing 101–150.
    expect(pageRange(2, 50, 50, 50)).toEqual({ start: 101, end: 150, total: 200 });
  });

  it('handles a short final page', () => {
    // Last page with 30 items and nothing remaining.
    expect(pageRange(3, 50, 30, 0)).toEqual({ start: 151, end: 180, total: 180 });
  });

  it('reports a zero range for an empty result', () => {
    expect(pageRange(0, 50, 0, 0)).toEqual({ start: 0, end: 0, total: 0 });
  });

  it('treats a negative remaining count as zero', () => {
    expect(pageRange(0, 50, 10, -1)).toEqual({ start: 1, end: 10, total: 10 });
  });
});
