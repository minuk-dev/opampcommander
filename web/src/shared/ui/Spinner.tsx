'use client';

import { Loader2 } from 'lucide-react';
import { cn } from '@shared/lib';

interface Props {
  className?: string;
  label?: string;
}

export default function Spinner({ className, label = 'Loading' }: Props) {
  return (
    <Loader2
      role="status"
      aria-label={label}
      className={cn('size-4 animate-spin text-muted-foreground', className)}
    />
  );
}
