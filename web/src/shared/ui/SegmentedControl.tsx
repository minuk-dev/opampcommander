'use client';

import * as ToggleGroupPrimitive from '@radix-ui/react-toggle-group';
import type { ComponentProps } from 'react';
import { cn } from '@shared/lib';

// Single-choice segmented control (the YAML/JSON switches, time formats).
export function SegmentedControl({
  className,
  ...props
}: ComponentProps<typeof ToggleGroupPrimitive.Root>) {
  return (
    <ToggleGroupPrimitive.Root
      className={cn('inline-flex items-center gap-0.5 rounded-md bg-muted p-0.5', className)}
      {...props}
    />
  );
}

export function SegmentedItem({
  className,
  ...props
}: ComponentProps<typeof ToggleGroupPrimitive.Item>) {
  return (
    <ToggleGroupPrimitive.Item
      className={cn(
        'rounded-[5px] px-2 py-1 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 data-[state=on]:bg-card data-[state=on]:text-foreground data-[state=on]:shadow-sm',
        className,
      )}
      {...props}
    />
  );
}
