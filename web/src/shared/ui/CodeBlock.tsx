'use client';

import { Check, Copy } from 'lucide-react';
import { type ReactNode, useState } from 'react';
import { cn, toYAML } from '@shared/lib';
import Button from './Button';
import { SegmentedControl, SegmentedItem } from './SegmentedControl';

export type CodeFormat = 'yaml' | 'json';

interface Props {
  value: unknown;
  title?: ReactNode;
  defaultFormat?: CodeFormat;
  maxHeight?: number | string;
  // When set, treat the content as already-serialized text and skip conversion.
  rawText?: string;
  className?: string;
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
  maxHeight = 420,
  rawText,
  className,
}: Props) {
  const [format, setFormat] = useState<CodeFormat>(defaultFormat);
  const [copied, setCopied] = useState(false);
  const isRawString = typeof value === 'string';
  const text = rawText ?? (isRawString ? value : serialize(value, format));
  const showToggle = !rawText && !isRawString;

  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    } catch {
      /* clipboard blocked — silently ignore */
    }
  };

  return (
    <div className={cn('min-w-0', className)}>
      <div className="mb-1.5 flex items-center justify-between gap-2">
        <div className="min-w-0 truncate text-xs font-medium text-muted-foreground">{title}</div>
        <div className="flex shrink-0 items-center gap-1">
          {showToggle && (
            <SegmentedControl
              type="single"
              value={format}
              onValueChange={(v: string) => v && setFormat(v as CodeFormat)}
              aria-label="format"
            >
              <SegmentedItem value="yaml">YAML</SegmentedItem>
              <SegmentedItem value="json">JSON</SegmentedItem>
            </SegmentedControl>
          )}
          <Button variant="ghost" size="icon-sm" aria-label="Copy" onClick={() => void onCopy()}>
            {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
          </Button>
        </div>
      </div>
      <pre
        className="overflow-auto rounded-md border border-border bg-muted/40 p-3 font-mono text-[13px] leading-5"
        style={{ maxHeight }}
      >
        {text}
      </pre>
    </div>
  );
}
