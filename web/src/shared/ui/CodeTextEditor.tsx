'use client';

import { type ChangeEvent, type UIEvent, useMemo, useRef } from 'react';
import { cn, type HighlightLanguage, type TokenKind, tokenizeLines } from '@shared/lib';

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
  className?: string;
}

// Shared text metrics — the highlight layer and the textarea MUST agree
// exactly, or the caret drifts away from the rendered glyphs.
const textMetrics = 'm-0 whitespace-pre px-3 py-2 font-mono text-[13px] leading-5 tracking-normal';

const tokenClass: Record<TokenKind, string> = {
  plain: 'text-foreground',
  comment: 'text-muted-foreground',
  key: 'text-[var(--tok-key)]',
  string: 'text-[var(--tok-string)]',
  number: 'text-[var(--tok-number)]',
  literal: 'text-[var(--tok-literal)]',
  punctuation: 'text-muted-foreground',
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
  className,
}: Props) {
  const highlightRef = useRef<HTMLPreElement>(null);
  const gutterRef = useRef<HTMLDivElement>(null);

  const lines = useMemo(() => tokenizeLines(value, language), [value, language]);

  const onScroll = (e: UIEvent<HTMLTextAreaElement>) => {
    const { scrollTop, scrollLeft } = e.currentTarget;
    if (highlightRef.current) {
      highlightRef.current.scrollTop = scrollTop;
      highlightRef.current.scrollLeft = scrollLeft;
    }
    if (gutterRef.current) gutterRef.current.scrollTop = scrollTop;
  };

  return (
    <div
      className={cn(
        'flex overflow-hidden rounded-md border border-input bg-card focus-within:ring-2 focus-within:ring-ring/60',
        // Token colours live here so both themes stay in one place.
        '[--tok-key:var(--color-primary)] [--tok-literal:oklch(0.55_0.16_300)] [--tok-number:oklch(0.55_0.15_60)] [--tok-string:oklch(0.5_0.12_155)]',
        'dark:[--tok-literal:oklch(0.78_0.13_300)] dark:[--tok-number:oklch(0.8_0.13_75)] dark:[--tok-string:oklch(0.78_0.13_155)]',
        className,
      )}
      style={{ height }}
    >
      <div
        ref={gutterRef}
        aria-hidden
        className={cn(
          textMetrics,
          'shrink-0 overflow-hidden border-r border-border bg-muted/50 px-2 text-right text-muted-foreground select-none',
        )}
        style={{ width: `calc(${Math.max(2, String(lines.length).length)}ch + 1rem)` }}
      >
        {lines.map((_, i) => (
          <div key={i} className={cn(errorLine === i + 1 && 'font-bold text-destructive')}>
            {i + 1}
          </div>
        ))}
      </div>
      <div className="relative min-w-0 flex-1">
        <pre
          ref={highlightRef}
          aria-hidden
          className={cn(textMetrics, 'pointer-events-none absolute inset-0 overflow-hidden')}
        >
          {lines.map((tokens, i) => (
            <span key={i}>
              {tokens.map((t, j) => (
                <span key={j} className={tokenClass[t.kind]}>
                  {t.text}
                </span>
              ))}
              {'\n'}
            </span>
          ))}
        </pre>
        <textarea
          value={value}
          onChange={(e: ChangeEvent<HTMLTextAreaElement>) => onChange(e.target.value)}
          onScroll={onScroll}
          spellCheck={false}
          autoCapitalize="off"
          autoCorrect="off"
          placeholder={placeholder}
          aria-label={ariaLabel}
          disabled={disabled}
          className={cn(
            textMetrics,
            'absolute inset-0 size-full resize-none overflow-auto border-0 bg-transparent text-transparent caret-foreground outline-none',
            'selection:bg-primary/35 placeholder:text-muted-foreground',
          )}
        />
      </div>
    </div>
  );
}
