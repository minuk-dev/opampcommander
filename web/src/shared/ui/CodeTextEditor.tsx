'use client';

import { Box, Paper, useTheme } from '@mui/material';
import { type ChangeEvent, type UIEvent, useMemo, useRef } from 'react';
import { type HighlightLanguage, type TokenKind, tokenizeLines } from '@shared/lib';

interface Props {
  value: string;
  onChange: (next: string) => void;
  language: HighlightLanguage;
  // Height of the editor box; the caller owns the layout.
  height?: number | string;
  // 1-based line to flag in the gutter (e.g. the line a parse error points at).
  errorLine?: number | null;
  placeholder?: string;
  ariaLabel: string;
  disabled?: boolean;
}

// Shared text metrics — the highlight layer and the textarea MUST agree
// exactly, or the caret drifts away from the rendered glyphs.
const textSx = {
  margin: 0,
  fontFamily: 'var(--font-geist-mono), monospace',
  fontSize: 13,
  lineHeight: '20px',
  letterSpacing: 0,
  tabSize: 2,
  whiteSpace: 'pre' as const,
  padding: '8px 12px',
  border: 0,
};

// CodeTextEditor is a plain <textarea> with a syntax-highlighted layer painted
// underneath it: the textarea's own text is transparent, so what the user sees
// is the highlight layer and what they edit is a normal, fully accessible
// textarea (native undo, spellcheck off, IME, selection all intact).
export default function CodeTextEditor({
  value,
  onChange,
  language,
  height = 360,
  errorLine,
  placeholder,
  ariaLabel,
  disabled,
}: Props) {
  const theme = useTheme();
  const highlightRef = useRef<HTMLPreElement>(null);
  const gutterRef = useRef<HTMLDivElement>(null);

  const lines = useMemo(() => tokenizeLines(value, language), [value, language]);
  const gutterWidth = `${Math.max(2, String(lines.length).length)}ch`;

  const colors: Record<TokenKind, string> = useMemo(() => {
    const dark = theme.palette.mode === 'dark';
    const pick = (c: { light: string; dark: string; main: string }) => (dark ? c.light : c.dark);
    return {
      plain: theme.palette.text.primary,
      comment: theme.palette.text.disabled,
      key: pick(theme.palette.primary),
      string: pick(theme.palette.success),
      number: pick(theme.palette.warning),
      literal: pick(theme.palette.secondary),
      punctuation: theme.palette.text.secondary,
    };
  }, [theme]);

  const onScroll = (e: UIEvent<HTMLTextAreaElement>) => {
    const { scrollTop, scrollLeft } = e.currentTarget;
    if (highlightRef.current) {
      highlightRef.current.scrollTop = scrollTop;
      highlightRef.current.scrollLeft = scrollLeft;
    }
    if (gutterRef.current) {
      gutterRef.current.scrollTop = scrollTop;
    }
  };

  return (
    <Paper
      variant="outlined"
      sx={{ display: 'flex', height, overflow: 'hidden', bgcolor: 'background.default' }}
    >
      <Box
        ref={gutterRef}
        aria-hidden
        sx={{
          ...textSx,
          flex: '0 0 auto',
          width: `calc(${gutterWidth} + 16px)`,
          textAlign: 'right',
          color: 'text.disabled',
          bgcolor: 'action.hover',
          overflow: 'hidden',
          userSelect: 'none',
        }}
      >
        {lines.map((_, i) => (
          <Box
            key={i}
            component="div"
            sx={
              errorLine === i + 1 ? { color: 'error.main', fontWeight: 700 } : { color: 'inherit' }
            }
          >
            {i + 1}
          </Box>
        ))}
      </Box>
      <Box sx={{ position: 'relative', flex: 1, minWidth: 0 }}>
        <Box
          component="pre"
          ref={highlightRef}
          aria-hidden
          sx={{
            ...textSx,
            position: 'absolute',
            inset: 0,
            overflow: 'hidden',
            pointerEvents: 'none',
          }}
        >
          {lines.map((tokens, i) => (
            <span key={i}>
              {tokens.map((t, j) => (
                <span key={j} style={{ color: colors[t.kind] }}>
                  {t.text}
                </span>
              ))}
              {'\n'}
            </span>
          ))}
        </Box>
        <Box
          component="textarea"
          value={value}
          onChange={(e: ChangeEvent<HTMLTextAreaElement>) => onChange(e.target.value)}
          onScroll={onScroll}
          spellCheck={false}
          autoCapitalize="off"
          autoCorrect="off"
          placeholder={placeholder}
          aria-label={ariaLabel}
          disabled={disabled}
          sx={{
            ...textSx,
            position: 'absolute',
            inset: 0,
            width: '100%',
            height: '100%',
            resize: 'none',
            outline: 'none',
            overflow: 'auto',
            background: 'transparent',
            color: 'transparent',
            caretColor: theme.palette.text.primary,
            // Selection must stay visible through the transparent text.
            '&::selection': { backgroundColor: theme.palette.primary.main, color: 'transparent' },
            '&::placeholder': { color: theme.palette.text.disabled },
          }}
        />
      </Box>
    </Paper>
  );
}
