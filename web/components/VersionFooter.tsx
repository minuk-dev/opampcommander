'use client';

import { Box, Stack, Tooltip, Typography } from '@mui/material';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import { api } from '@/lib/api-client';
import { WEB_VERSION } from '@/lib/web-version';

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
        // best-effort; leave api line as "—"
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const apiLabel = info?.gitVersion || '—';

  return (
    <Tooltip
      title={
        <Box sx={{ fontFamily: 'monospace', fontSize: 11 }}>
          <div>
            <strong>web</strong>: {WEB_VERSION}
          </div>
          {info ? (
            <>
              {info.gitVersion && <div><strong>api</strong>: {info.gitVersion}</div>}
              {info.gitCommit && <div>commit: {info.gitCommit.slice(0, 12)}</div>}
              {info.buildDate && <div>built: {info.buildDate}</div>}
              {info.goVersion && <div>go: {info.goVersion}</div>}
              {info.platform && <div>platform: {info.platform}</div>}
            </>
          ) : (
            <div>api: loading…</div>
          )}
        </Box>
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
        <Stack spacing={0}>
          <Stack
            direction="row"
            spacing={1}
            alignItems="baseline"
            sx={{ overflow: 'hidden' }}
          >
            <Typography
              variant="caption"
              sx={{
                color: 'text.disabled',
                fontFamily: 'monospace',
                flexShrink: 0,
                width: 28,
              }}
            >
              web
            </Typography>
            <Typography
              variant="caption"
              sx={{
                fontFamily: 'monospace',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {WEB_VERSION}
            </Typography>
          </Stack>
          <Stack
            direction="row"
            spacing={1}
            alignItems="baseline"
            sx={{ overflow: 'hidden' }}
          >
            <Typography
              variant="caption"
              sx={{
                color: 'text.disabled',
                fontFamily: 'monospace',
                flexShrink: 0,
                width: 28,
              }}
            >
              api
            </Typography>
            <Typography
              variant="caption"
              sx={{
                fontFamily: 'monospace',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {apiLabel}
            </Typography>
          </Stack>
        </Stack>
      </Box>
    </Tooltip>
  );
}
