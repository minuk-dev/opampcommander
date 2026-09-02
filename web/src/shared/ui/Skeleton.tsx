'use client';

import type { ComponentProps } from 'react';
import { cn } from '@shared/lib';

export default function Skeleton({ className, ...props }: ComponentProps<'div'>) {
  return <div className={cn('animate-pulse rounded-md bg-muted', className)} {...props} />;
}
