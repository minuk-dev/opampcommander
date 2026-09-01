'use client';

import type { ComponentProps } from 'react';
import { cn } from '@shared/lib';

// Indeterminate when `value` is omitted — the loading bar the app shell shows
// while a page's data is in flight.
export default function Progress({ className, value }: ComponentProps<'div'> & { value?: number }) {
  return (
    <div className={cn('h-1 w-full overflow-hidden rounded-full bg-muted', className)}>
      <div
        className={cn(
          'h-full bg-primary transition-[width]',
          value === undefined && 'w-1/3 animate-pulse',
        )}
        style={value === undefined ? undefined : { width: `${Math.min(100, value)}%` }}
      />
    </div>
  );
}
