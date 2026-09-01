'use client';

import type { ComponentProps } from 'react';
import { cn } from '@shared/lib';

interface Props extends ComponentProps<'textarea'> {
  invalid?: boolean;
  mono?: boolean;
}

export default function Textarea({ className, invalid, mono, ...props }: Props) {
  return (
    <textarea
      spellCheck={mono ? false : props.spellCheck}
      className={cn(
        'w-full rounded-md border bg-card px-2.5 py-1.5 text-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 disabled:cursor-not-allowed disabled:opacity-60',
        invalid ? 'border-destructive' : 'border-input',
        mono && 'font-mono text-[13px] leading-5',
        className,
      )}
      {...props}
    />
  );
}
