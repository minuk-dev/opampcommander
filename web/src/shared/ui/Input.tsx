'use client';

import type { ComponentProps, ReactNode } from 'react';
import { cn } from '@shared/lib';

interface Props extends ComponentProps<'input'> {
  invalid?: boolean;
  // Rendered inside the field, before/after the text (icons, units, buttons).
  startSlot?: ReactNode;
  endSlot?: ReactNode;
}

export const fieldBase =
  'flex h-8 w-full rounded-md border bg-card px-2.5 py-1 text-sm transition-colors placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-60';

export default function Input({ className, invalid, startSlot, endSlot, ...props }: Props) {
  const field = (
    <input
      className={cn(
        fieldBase,
        invalid ? 'border-destructive' : 'border-input',
        startSlot && 'rounded-l-none border-l-0 pl-0',
        endSlot && 'rounded-r-none border-r-0 pr-0',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60',
        className,
      )}
      {...props}
    />
  );

  if (!startSlot && !endSlot) return field;

  return (
    <div
      className={cn(
        'flex items-stretch rounded-md border focus-within:ring-2 focus-within:ring-ring/60',
        invalid ? 'border-destructive' : 'border-input',
      )}
    >
      {startSlot && (
        <span className="flex items-center pl-2.5 text-muted-foreground [&_svg]:size-4">
          {startSlot}
        </span>
      )}
      {field}
      {endSlot && (
        <span className="flex items-center pr-1 text-muted-foreground [&_svg]:size-4">
          {endSlot}
        </span>
      )}
    </div>
  );
}
