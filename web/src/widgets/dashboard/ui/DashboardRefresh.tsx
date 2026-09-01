'use client';

import { RefreshCw } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useTransition } from 'react';
import { cn } from '@shared/lib';
import { Button } from '@shared/ui';

// Client island for the dashboard's refresh control. router.refresh() re-runs
// the Server Component data fetch; useTransition gives us a pending state
// without a manual loading flag (best-practices guide 6.11).
export default function DashboardRefresh() {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  return (
    <Button
      variant="ghost"
      size="icon-sm"
      aria-label="refresh"
      disabled={pending}
      onClick={() => startTransition(() => router.refresh())}
    >
      <RefreshCw className={cn(pending && 'animate-spin')} aria-hidden />
    </Button>
  );
}
