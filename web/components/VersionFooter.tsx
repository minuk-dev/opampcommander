'use client';

import { Box, Tooltip, Typography } from '@mui/material';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import { api } from '@/lib/api-client';

interface VersionInfo {
  gitVersion?: string;
  gitCommit?: string;
  buildDate?: string;
  goVersion?: string;
  platform?: string;
  [key: string]: string | undefined;
}

export default function VersionFooter() {
  const [info, setInfo] = useState<VersionInfo | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const data = await api.get<VersionInfo>('/api/v1/version');
        if (!cancelled) setInfo(data);
      } catch {
        // best-effort; leave footer empty
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const label = info?.gitVersion || 'unknown';

  return (
    <Tooltip
      title={
        info ? (
          <Box sx={{ fontFamily: 'monospace', fontSize: 11 }}>
            {info.gitVersion && <div>version: {info.gitVersion}</div>}
            {info.gitCommit && <div>commit: {info.gitCommit.slice(0, 12)}</div>}
            {info.buildDate && <div>built: {info.buildDate}</div>}
            {info.goVersion && <div>go: {info.goVersion}</div>}
            {info.platform && <div>platform: {info.platform}</div>}
          </Box>
        ) : (
          'Loading version…'
        )
      }
      placement="right"
    >
      <Box
        component={Link}
        href="/version"
        sx={{
          display: 'block',
          px: 2,
          py: 1,
          borderTop: '1px solid',
          borderColor: 'divider',
          textDecoration: 'none',
          color: 'text.secondary',
          '&:hover': { bgcolor: 'action.hover', color: 'text.primary' },
        }}
      >
        <Typography
          variant="caption"
          sx={{
            display: 'block',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            fontFamily: 'monospace',
          }}
        >
          {label}
        </Typography>
      </Box>
    </Tooltip>
  );
}
