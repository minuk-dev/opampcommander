'use client';

import { IconButton, Tooltip } from '@mui/material';
import { Refresh as RefreshIcon } from '@mui/icons-material';
import { useRouter } from 'next/navigation';
import { useTransition } from 'react';

// Client island for the dashboard's refresh control. router.refresh() re-runs
// the Server Component data fetch; useTransition gives us a pending state
// without a manual loading flag (best-practices guide 6.11).
export default function DashboardRefresh() {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  return (
    <Tooltip title="Refresh">
      <IconButton
        color="primary"
        aria-label="refresh"
        disabled={pending}
        onClick={() => startTransition(() => router.refresh())}
      >
        <RefreshIcon />
      </IconButton>
    </Tooltip>
  );
}
