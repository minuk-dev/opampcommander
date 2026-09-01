'use client';

import * as TooltipPrimitive from '@radix-ui/react-tooltip';
import { type ComponentProps, type ReactNode } from 'react';
import { cn } from '@shared/lib';

export const TooltipProvider = TooltipPrimitive.Provider;

interface Props {
  content: ReactNode;
  children: ReactNode;
  side?: ComponentProps<typeof TooltipPrimitive.Content>['side'];
  className?: string;
}

// One-shot tooltip: the trigger renders its child as-is, so an icon button
// keeps its own styling and accessibility.
export default function Tooltip({ content, children, side = 'top', className }: Props) {
  if (!content) return <>{children}</>;
  return (
    <TooltipPrimitive.Root>
      <TooltipPrimitive.Trigger asChild>{children}</TooltipPrimitive.Trigger>
      <TooltipPrimitive.Portal>
        <TooltipPrimitive.Content
          side={side}
          sideOffset={6}
          className={cn(
            'z-50 max-w-xs rounded-md bg-foreground px-2 py-1 text-xs text-background shadow-md data-[state=delayed-open]:animate-in data-[state=delayed-open]:fade-in-0',
            className,
          )}
        >
          {content}
        </TooltipPrimitive.Content>
      </TooltipPrimitive.Portal>
    </TooltipPrimitive.Root>
  );
}
