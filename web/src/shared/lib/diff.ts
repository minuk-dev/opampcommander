// Line-oriented diff used by the config editors to show what a save will change.
//
// Classic LCS over lines, with the common prefix/suffix trimmed first so the
// quadratic part only runs over the region that actually differs. Configs that
// differ wholesale are capped: past MAX_LCS_CELLS the middle region is reported
// as a plain remove-then-add block instead of a fine-grained alignment.

export type DiffKind = 'equal' | 'add' | 'remove';

export interface DiffRow {
  kind: DiffKind;
  text: string;
  // 1-based line numbers in the old/new text; undefined on the side that does
  // not contain the row.
  oldLine?: number;
  newLine?: number;
}

const MAX_LCS_CELLS = 4_000_000;

function splitLines(text: string): string[] {
  // A trailing newline is a terminator, not an empty final line.
  const normalized = text.replace(/\r\n/g, '\n').replace(/\n$/, '');
  return normalized === '' ? [] : normalized.split('\n');
}

// lcsTable builds the standard dynamic-programming table for the two slices.
function lcsTable(a: string[], b: string[]): Uint32Array {
  const w = b.length + 1;
  const table = new Uint32Array((a.length + 1) * w);
  for (let i = a.length - 1; i >= 0; i--) {
    for (let j = b.length - 1; j >= 0; j--) {
      table[i * w + j] =
        a[i] === b[j]
          ? table[(i + 1) * w + j + 1] + 1
          : Math.max(table[(i + 1) * w + j], table[i * w + j + 1]);
    }
  }
  return table;
}

export function diffLines(oldText: string, newText: string): DiffRow[] {
  const a = splitLines(oldText);
  const b = splitLines(newText);

  let prefix = 0;
  while (prefix < a.length && prefix < b.length && a[prefix] === b[prefix]) prefix++;
  let suffix = 0;
  while (
    suffix < a.length - prefix &&
    suffix < b.length - prefix &&
    a[a.length - 1 - suffix] === b[b.length - 1 - suffix]
  ) {
    suffix++;
  }

  const rows: DiffRow[] = [];
  for (let i = 0; i < prefix; i++) {
    rows.push({ kind: 'equal', text: a[i], oldLine: i + 1, newLine: i + 1 });
  }

  const midA = a.slice(prefix, a.length - suffix);
  const midB = b.slice(prefix, b.length - suffix);
  let oldLine = prefix + 1;
  let newLine = prefix + 1;

  if ((midA.length + 1) * (midB.length + 1) > MAX_LCS_CELLS) {
    for (const text of midA) rows.push({ kind: 'remove', text, oldLine: oldLine++ });
    for (const text of midB) rows.push({ kind: 'add', text, newLine: newLine++ });
  } else {
    const w = midB.length + 1;
    const table = lcsTable(midA, midB);
    let i = 0;
    let j = 0;
    while (i < midA.length && j < midB.length) {
      if (midA[i] === midB[j]) {
        rows.push({ kind: 'equal', text: midA[i], oldLine: oldLine++, newLine: newLine++ });
        i++;
        j++;
      } else if (table[(i + 1) * w + j] >= table[i * w + j + 1]) {
        rows.push({ kind: 'remove', text: midA[i], oldLine: oldLine++ });
        i++;
      } else {
        rows.push({ kind: 'add', text: midB[j], newLine: newLine++ });
        j++;
      }
    }
    for (; i < midA.length; i++) rows.push({ kind: 'remove', text: midA[i], oldLine: oldLine++ });
    for (; j < midB.length; j++) rows.push({ kind: 'add', text: midB[j], newLine: newLine++ });
  }

  for (let k = 0; k < suffix; k++) {
    rows.push({
      kind: 'equal',
      text: a[a.length - suffix + k],
      oldLine: oldLine++,
      newLine: newLine++,
    });
  }

  return rows;
}

export interface DiffStat {
  added: number;
  removed: number;
}

export function diffStat(rows: DiffRow[]): DiffStat {
  return rows.reduce<DiffStat>(
    (acc, r) => {
      if (r.kind === 'add') acc.added++;
      if (r.kind === 'remove') acc.removed++;
      return acc;
    },
    { added: 0, removed: 0 },
  );
}

// collapseContext drops runs of unchanged lines longer than 2*context, leaving
// `context` lines of surroundings on each side. Returned gaps carry the number
// of hidden lines so the UI can label them.
export interface DiffChunk {
  rows: DiffRow[];
  hiddenBefore: number;
}

export function collapseContext(rows: DiffRow[], context = 3): DiffChunk[] {
  const keep = new Array<boolean>(rows.length).fill(false);
  rows.forEach((row, i) => {
    if (row.kind === 'equal') return;
    for (let k = Math.max(0, i - context); k <= Math.min(rows.length - 1, i + context); k++) {
      keep[k] = true;
    }
  });

  const chunks: DiffChunk[] = [];
  let hidden = 0;
  let current: DiffRow[] | null = null;
  rows.forEach((row, i) => {
    if (keep[i]) {
      if (!current) {
        current = [];
        chunks.push({ rows: current, hiddenBefore: hidden });
        hidden = 0;
      }
      current.push(row);
    } else {
      hidden++;
      current = null;
    }
  });
  return chunks;
}
