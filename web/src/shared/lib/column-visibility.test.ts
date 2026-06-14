import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { type ColumnConfig, useColumnVisibility } from './column-visibility';

const COLUMNS: ColumnConfig[] = [
  { id: 'a', label: 'A', locked: true },
  { id: 'b', label: 'B' }, // defaultVisible omitted -> true
  { id: 'c', label: 'C', defaultVisible: false },
];

const KEY = 'opamp.columns.t';

afterEach(() => {
  window.localStorage.clear();
});

describe('useColumnVisibility', () => {
  it('uses defaults when nothing is stored', () => {
    const { result } = renderHook(() => useColumnVisibility('t', COLUMNS));
    expect(result.current.isVisible('a')).toBe(true);
    expect(result.current.isVisible('b')).toBe(true);
    expect(result.current.isVisible('c')).toBe(false);
  });

  it('persists a toggle and reloads it on remount', () => {
    const first = renderHook(() => useColumnVisibility('t', COLUMNS));
    act(() => first.result.current.toggle('c'));
    expect(first.result.current.isVisible('c')).toBe(true);
    first.unmount();

    const second = renderHook(() => useColumnVisibility('t', COLUMNS));
    expect(second.result.current.isVisible('c')).toBe(true);
  });

  it('falls back to defaults on corrupt JSON', () => {
    window.localStorage.setItem(KEY, '{ not json');
    const { result } = renderHook(() => useColumnVisibility('t', COLUMNS));
    expect(result.current.isVisible('c')).toBe(false);
  });

  it('ignores an incompatible schema version', () => {
    window.localStorage.setItem(KEY, JSON.stringify({ v: 999, columns: { c: true } }));
    const { result } = renderHook(() => useColumnVisibility('t', COLUMNS));
    // Stored override discarded -> 'c' keeps its default (hidden).
    expect(result.current.isVisible('c')).toBe(false);
  });

  it('ignores a missing/legacy envelope (no version field)', () => {
    window.localStorage.setItem(KEY, JSON.stringify({ c: true }));
    const { result } = renderHook(() => useColumnVisibility('t', COLUMNS));
    expect(result.current.isVisible('c')).toBe(false);
  });

  it('drops non-boolean fields but keeps valid overrides', () => {
    window.localStorage.setItem(
      KEY,
      JSON.stringify({ v: 1, columns: { b: 'nope', c: true, unknown: true } }),
    );
    const { result } = renderHook(() => useColumnVisibility('t', COLUMNS));
    expect(result.current.isVisible('b')).toBe(true); // bad value ignored -> default
    expect(result.current.isVisible('c')).toBe(true); // valid override applied
  });

  it('never lets a stored value turn a locked column off', () => {
    window.localStorage.setItem(KEY, JSON.stringify({ v: 1, columns: { a: false } }));
    const { result } = renderHook(() => useColumnVisibility('t', COLUMNS));
    expect(result.current.isVisible('a')).toBe(true);
  });
});
