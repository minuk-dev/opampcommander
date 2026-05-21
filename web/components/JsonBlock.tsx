'use client';

import { Box, Paper, Typography } from '@mui/material';

interface Props {
  value: unknown;
  title?: string;
  maxHeight?: number | string;
}

export default function JsonBlock({ value, title, maxHeight = 480 }: Props) {
  const text =
    value === undefined
      ? '(undefined)'
      : typeof value === 'string'
        ? value
        : JSON.stringify(value, null, 2);
  return (
    <Box>
      {title && (
        <Typography variant="subtitle2" gutterBottom>
          {title}
        </Typography>
      )}
      <Paper
        variant="outlined"
        sx={{
          p: 2,
          fontFamily: 'var(--font-geist-mono), monospace',
          fontSize: 13,
          whiteSpace: 'pre',
          overflow: 'auto',
          maxHeight,
          bgcolor: 'background.default',
        }}
      >
        {text}
      </Paper>
    </Box>
  );
}
