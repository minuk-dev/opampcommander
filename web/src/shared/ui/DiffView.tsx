'use client';

import { Box, Paper, Stack, Typography } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { useMemo } from 'react';
import { collapseContext, diffLines, diffStat } from '@shared/lib';

interface Props {
  oldText: string;
  newText: string;
  oldLabel?: string;
  newLabel?: string;
  maxHeight?: number | string;
  // Unchanged lines kept around each change.
  context?: number;
}

const cellSx = {
  fontFamily: 'var(--font-geist-mono), monospace',
  fontSize: 12,
  lineHeight: '18px',
  whiteSpace: 'pre' as const,
};

// DiffView renders a unified line diff, collapsing long unchanged runs.
export default function DiffView({
  oldText,
  newText,
  oldLabel = 'current',
  newLabel = 'new',
  maxHeight = 320,
  context = 3,
}: Props) {
  const { chunks, stat } = useMemo(() => {
    const rows = diffLines(oldText, newText);
    return { chunks: collapseContext(rows, context), stat: diffStat(rows) };
  }, [oldText, newText, context]);

  if (stat.added === 0 && stat.removed === 0) {
    return (
      <Typography variant="body2" color="text.secondary">
        No changes.
      </Typography>
    );
  }

  return (
    <Box>
      <Stack direction="row" spacing={2} sx={{ mb: 1 }}>
        <Typography variant="caption" color="text.secondary">
          {oldLabel} → {newLabel}
        </Typography>
        <Typography variant="caption" color="success.main">
          +{stat.added}
        </Typography>
        <Typography variant="caption" color="error.main">
          −{stat.removed}
        </Typography>
      </Stack>
      <Paper variant="outlined" sx={{ overflow: 'auto', maxHeight, bgcolor: 'background.default' }}>
        {chunks.map((chunk, ci) => (
          <Box key={ci}>
            {chunk.hiddenBefore > 0 && (
              <Box sx={{ ...cellSx, px: 1, color: 'text.disabled', bgcolor: 'action.hover' }}>
                {`⋯ ${chunk.hiddenBefore} unchanged line${chunk.hiddenBefore === 1 ? '' : 's'}`}
              </Box>
            )}
            {chunk.rows.map((row, ri) => (
              <Box
                key={ri}
                sx={{
                  ...cellSx,
                  display: 'flex',
                  bgcolor: (theme) =>
                    row.kind === 'add'
                      ? alpha(theme.palette.success.main, 0.16)
                      : row.kind === 'remove'
                        ? alpha(theme.palette.error.main, 0.14)
                        : 'transparent',
                  color: row.kind === 'equal' ? 'text.secondary' : 'text.primary',
                }}
              >
                <Box
                  component="span"
                  sx={{ width: '3ch', flex: '0 0 auto', textAlign: 'center', userSelect: 'none' }}
                >
                  {row.kind === 'add' ? '+' : row.kind === 'remove' ? '-' : ' '}
                </Box>
                <Box component="span" sx={{ pr: 1 }}>
                  {row.text === '' ? ' ' : row.text}
                </Box>
              </Box>
            ))}
          </Box>
        ))}
      </Paper>
    </Box>
  );
}
