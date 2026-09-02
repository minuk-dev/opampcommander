'use client';

import { type ReactNode, useId } from 'react';
import { cn } from '@shared/lib';
import Label from './Label';

interface Props {
  label?: ReactNode;
  // Rendered under the control; `error` replaces it and turns destructive.
  hint?: ReactNode;
  error?: ReactNode;
  required?: boolean;
  className?: string;
  // Receives the generated id so the control and its label stay associated.
  children: (props: { id: string; 'aria-describedby'?: string }) => ReactNode;
}

// Field is the label + control + message wrapper every form in the app uses,
// so spacing and error presentation stay identical across dialogs.
export default function Field({ label, hint, error, required, className, children }: Props) {
  const id = useId();
  const messageId = `${id}-message`;
  const message = error ?? hint;

  return (
    <div className={cn('grid gap-1', className)}>
      {label && (
        <div className="flex items-center gap-0.5">
          {/* The asterisk sits outside the <label> so the control's accessible
              name stays exactly the label text. */}
          <Label htmlFor={id}>{label}</Label>
          {required && (
            <span aria-hidden className="text-xs text-destructive">
              *
            </span>
          )}
        </div>
      )}
      {children({ id, 'aria-describedby': message ? messageId : undefined })}
      {message && (
        <p
          id={messageId}
          className={cn('text-xs', error ? 'text-destructive' : 'text-muted-foreground')}
        >
          {message}
        </p>
      )}
    </div>
  );
}
