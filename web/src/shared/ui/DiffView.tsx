'use client';

import { useMemo } from 'react';
import { cn, collapseContext, diffLines, diffStat } from '@shared/lib';

interface Props {
  oldText: string;
  newText: string;
  oldLabel?: string;
  newLabel?: string;
  maxHeight?: number | string;
  // Unchanged lines kept around each change.
  context?: number;
  className?: string;
}

const cell = 'whitespace-pre font-mono text-xs leading-[18px]';

// DiffView renders a unified line diff, collapsing long unchanged runs.
export default function DiffView({
  oldText,
  newText,
  oldLabel = 'current',
  newLabel = 'new',
  maxHeight = 320,
  context = 3,
  className,
}: Props) {
  const { chunks, stat } = useMemo(() => {
    const rows = diffLines(oldText, newText);
    return { chunks: collapseContext(rows, context), stat: diffStat(rows) };
  }, [oldText, newText, context]);

  if (stat.added === 0 && stat.removed === 0) {
    return <p className="text-sm text-muted-foreground">No changes.</p>;
  }

  return (
    <div className={cn('min-w-0', className)}>
      <div className="mb-1.5 flex items-center gap-3 text-xs text-muted-foreground">
        <span>
          {oldLabel} → {newLabel}
        </span>
        <span className="tnum text-success">+{stat.added}</span>
        <span className="tnum text-destructive">−{stat.removed}</span>
      </div>
      <div
        className="overflow-auto rounded-md border border-border bg-muted/30"
        style={{ maxHeight }}
      >
        {chunks.map((chunk, ci) => (
          <div key={ci}>
            {chunk.hiddenBefore > 0 && (
              <div className={cn(cell, 'bg-muted px-2 text-muted-foreground')}>
                {`⋯ ${chunk.hiddenBefore} unchanged line${chunk.hiddenBefore === 1 ? '' : 's'}`}
              </div>
            )}
            {chunk.rows.map((row, ri) => (
              <div
                key={ri}
                className={cn(
                  cell,
                  'flex',
                  row.kind === 'add' && 'bg-success/12',
                  row.kind === 'remove' && 'bg-destructive/12',
                  row.kind === 'equal' && 'text-muted-foreground',
                )}
              >
                <span
                  className={cn(
                    'w-[3ch] shrink-0 text-center select-none',
                    row.kind === 'add' && 'text-success',
                    row.kind === 'remove' && 'text-destructive',
                  )}
                >
                  {row.kind === 'add' ? '+' : row.kind === 'remove' ? '-' : ' '}
                </span>
                <span className="pr-2">{row.text === '' ? ' ' : row.text}</span>
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
