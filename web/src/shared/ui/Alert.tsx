'use client';

import { cva, type VariantProps } from 'class-variance-authority';
import { AlertTriangle, CheckCircle2, Info, X, XCircle } from 'lucide-react';
import type { ComponentProps, ReactNode } from 'react';
import { cn } from '@shared/lib';

const alertVariants = cva('relative flex gap-2.5 rounded-md border px-3 py-2.5 text-sm', {
  variants: {
    severity: {
      info: 'border-primary/25 bg-primary/8 text-foreground [&>svg]:text-primary',
      success: 'border-success/25 bg-success/8 text-foreground [&>svg]:text-success',
      warning: 'border-warning/30 bg-warning/10 text-foreground [&>svg]:text-warning',
      error: 'border-destructive/30 bg-destructive/8 text-foreground [&>svg]:text-destructive',
    },
  },
  defaultVariants: { severity: 'info' },
});

const icons = {
  info: Info,
  success: CheckCircle2,
  warning: AlertTriangle,
  error: XCircle,
} as const;

// `title` is overridden with a ReactNode heading, so the DOM tooltip attribute
// is dropped rather than shadowed.
interface Props extends Omit<ComponentProps<'div'>, 'title'>, VariantProps<typeof alertVariants> {
  title?: ReactNode;
  onClose?: () => void;
}

export default function Alert({
  className,
  severity = 'info',
  title,
  onClose,
  children,
  ...props
}: Props) {
  const Icon = icons[severity ?? 'info'];
  return (
    <div role="alert" className={cn(alertVariants({ severity }), className)} {...props}>
      <Icon className="mt-0.5 size-4 shrink-0" aria-hidden />
      <div className="min-w-0 flex-1">
        {title && <p className="font-medium">{title}</p>}
        {children && <div className={cn('min-w-0', title && 'mt-0.5')}>{children}</div>}
      </div>
      {onClose && (
        <button
          type="button"
          onClick={onClose}
          aria-label="dismiss"
          className="-mr-1 -mt-0.5 size-6 shrink-0 rounded text-muted-foreground hover:bg-black/5 dark:hover:bg-white/10"
        >
          <X className="mx-auto size-3.5" aria-hidden />
        </button>
      )}
    </div>
  );
}
