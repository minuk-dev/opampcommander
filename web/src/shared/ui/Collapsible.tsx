'use client';

import * as CollapsiblePrimitive from '@radix-ui/react-collapsible';
import { ChevronDown } from 'lucide-react';
import { type ReactNode, useState } from 'react';
import { cn } from '@shared/lib';

interface Props {
  label: ReactNode;
  defaultOpen?: boolean;
  className?: string;
  children: ReactNode;
}

// Disclosure used for the "Advanced" sections in the editors.
export default function Collapsible({ label, defaultOpen = false, className, children }: Props) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <CollapsiblePrimitive.Root open={open} onOpenChange={setOpen} className={cn(className)}>
      <CollapsiblePrimitive.Trigger className="flex items-center gap-1 rounded text-xs font-medium text-muted-foreground transition-colors hover:text-foreground">
        <ChevronDown
          className={cn('size-3.5 transition-transform', open && 'rotate-180')}
          aria-hidden
        />
        {label}
      </CollapsiblePrimitive.Trigger>
      <CollapsiblePrimitive.Content className="overflow-hidden data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0">
        <div className="pt-2">{children}</div>
      </CollapsiblePrimitive.Content>
    </CollapsiblePrimitive.Root>
  );
}
