'use client';

import * as ToastPrimitive from '@radix-ui/react-toast';
import { CheckCircle2, X, XCircle } from 'lucide-react';
import { createContext, type ReactNode, useCallback, useContext, useMemo, useState } from 'react';
import { cn } from '@shared/lib';

export type ToastSeverity = 'success' | 'error' | 'info';

interface ToastItem {
  id: number;
  severity: ToastSeverity;
  message: ReactNode;
}

interface ToastApi {
  toast: (severity: ToastSeverity, message: ReactNode) => void;
}

const ToastContext = createContext<ToastApi | null>(null);

// useToast replaces the per-page snackbar state each view used to carry.
export function useToast(): ToastApi {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast must be used inside <ToastProvider>');
  return ctx;
}

const icons = { success: CheckCircle2, error: XCircle, info: null } as const;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);

  const toast = useCallback((severity: ToastSeverity, message: ReactNode) => {
    setItems((prev) => [...prev, { id: Date.now() + Math.random(), severity, message }]);
  }, []);

  const api = useMemo(() => ({ toast }), [toast]);
  const dismiss = (id: number) => setItems((prev) => prev.filter((t) => t.id !== id));

  return (
    <ToastContext.Provider value={api}>
      <ToastPrimitive.Provider duration={5000} swipeDirection="right">
        {children}
        {items.map((item) => {
          const Icon = icons[item.severity];
          return (
            <ToastPrimitive.Root
              key={item.id}
              onOpenChange={(open) => !open && dismiss(item.id)}
              className={cn(
                'flex items-start gap-2 rounded-md border bg-popover p-3 text-sm shadow-lg data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:slide-in-from-bottom-2 data-[swipe=end]:animate-out',
                item.severity === 'error' ? 'border-destructive/40' : 'border-border',
              )}
            >
              {Icon && (
                <Icon
                  className={cn(
                    'mt-0.5 size-4 shrink-0',
                    item.severity === 'success' ? 'text-success' : 'text-destructive',
                  )}
                  aria-hidden
                />
              )}
              <ToastPrimitive.Description className="min-w-0 flex-1">
                {item.message}
              </ToastPrimitive.Description>
              <ToastPrimitive.Close
                aria-label="dismiss"
                className="shrink-0 rounded text-muted-foreground hover:text-foreground"
              >
                <X className="size-3.5" aria-hidden />
              </ToastPrimitive.Close>
            </ToastPrimitive.Root>
          );
        })}
        <ToastPrimitive.Viewport className="fixed right-0 bottom-0 z-100 flex w-full max-w-sm flex-col gap-2 p-4" />
      </ToastPrimitive.Provider>
    </ToastContext.Provider>
  );
}
