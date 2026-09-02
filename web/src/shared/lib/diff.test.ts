import { describe, expect, it } from 'vitest';
import { collapseContext, diffLines, diffStat } from './diff';

describe('diffLines', () => {
  it('reports no changes for identical text', () => {
    const rows = diffLines('a\nb\nc\n', 'a\nb\nc\n');
    expect(rows.every((r) => r.kind === 'equal')).toBe(true);
    expect(diffStat(rows)).toEqual({ added: 0, removed: 0 });
  });

  it('treats a trailing newline as a terminator, not an extra line', () => {
    expect(diffStat(diffLines('a\nb', 'a\nb\n'))).toEqual({ added: 0, removed: 0 });
  });

  it('aligns an inserted line against the surrounding context', () => {
    const rows = diffLines('a\nc', 'a\nb\nc');
    expect(rows.map((r) => [r.kind, r.text])).toEqual([
      ['equal', 'a'],
      ['add', 'b'],
      ['equal', 'c'],
    ]);
  });

  it('numbers lines on the side they belong to', () => {
    const rows = diffLines('a\nold\nc', 'a\nnew\nc');
    const changed = rows.filter((r) => r.kind !== 'equal');
    expect(changed).toEqual([
      { kind: 'remove', text: 'old', oldLine: 2 },
      { kind: 'add', text: 'new', newLine: 2 },
    ]);
  });

  it('diffs a realistic config edit', () => {
    const before = 'exporters:\n  debug:\n    verbosity: basic\n';
    const after = 'exporters:\n  debug:\n    verbosity: detailed\n';
    expect(diffStat(diffLines(before, after))).toEqual({ added: 1, removed: 1 });
  });

  it('handles one side being empty', () => {
    expect(diffStat(diffLines('', 'a\nb'))).toEqual({ added: 2, removed: 0 });
    expect(diffStat(diffLines('a\nb', ''))).toEqual({ added: 0, removed: 2 });
  });
});

describe('collapseContext', () => {
  it('hides unchanged runs beyond the context window', () => {
    const before = Array.from({ length: 30 }, (_, i) => `line ${i}`).join('\n');
    const after = before.replace('line 15', 'line fifteen');
    const chunks = collapseContext(diffLines(before, after), 2);

    expect(chunks).toHaveLength(1);
    expect(chunks[0].hiddenBefore).toBe(13);
    // 2 context lines each side + the removed and added line.
    expect(chunks[0].rows).toHaveLength(6);
  });

  it('keeps everything when every line changed', () => {
    const chunks = collapseContext(diffLines('a\nb', 'c\nd'), 3);
    expect(chunks).toHaveLength(1);
    expect(chunks[0].hiddenBefore).toBe(0);
    expect(chunks[0].rows).toHaveLength(4);
  });
});
