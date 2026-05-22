'use client';

import {
  Box,
  IconButton,
  Paper,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
  Tooltip,
  Typography,
} from '@mui/material';
import { ContentCopy as CopyIcon } from '@mui/icons-material';
import { ReactNode, useState } from 'react';
import { toYAML } from '@/lib/yaml';

export type CodeFormat = 'yaml' | 'json';

interface Props {
  value: unknown;
  title?: ReactNode;
  defaultFormat?: CodeFormat;
  maxHeight?: number | string;
  // When true, treat `value` as already-serialized text and skip conversion.
  rawText?: string;
}

function serialize(value: unknown, format: CodeFormat): string {
  if (format === 'yaml') return toYAML(value);
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

// CodeBlock is the canonical viewer for structured payloads. It shows a
// YAML/JSON toggle for structured values. Plain strings (e.g. an
// already-serialized config file body) bypass the toggle and render as-is.
export default function CodeBlock({
  value,
  title,
  defaultFormat = 'yaml',
  maxHeight = 480,
  rawText,
}: Props) {
  const [format, setFormat] = useState<CodeFormat>(defaultFormat);
  const isRawString = typeof value === 'string';
  const text = rawText ?? (isRawString ? (value as string) : serialize(value, format));
  const showToggle = !rawText && !isRawString;

  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
    } catch {
      /* clipboard blocked — silently ignore */
    }
  };

  return (
    <Box>
      <Stack
        direction="row"
        alignItems="center"
        justifyContent="space-between"
        sx={{ mb: 1 }}
      >
        <Box>
          {title &&
            (typeof title === 'string' ? (
              <Typography variant="subtitle2">{title}</Typography>
            ) : (
              title
            ))}
        </Box>
        <Stack direction="row" gap={1} alignItems="center">
          {showToggle && (
            <ToggleButtonGroup
              size="small"
              exclusive
              value={format}
              onChange={(_, v: CodeFormat | null) => v && setFormat(v)}
              aria-label="format"
            >
              <ToggleButton value="yaml">YAML</ToggleButton>
              <ToggleButton value="json">JSON</ToggleButton>
            </ToggleButtonGroup>
          )}
          <Tooltip title="Copy">
            <IconButton size="small" onClick={onCopy}>
              <CopyIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Stack>
      </Stack>
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
