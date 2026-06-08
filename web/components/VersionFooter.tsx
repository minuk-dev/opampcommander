'use client';

import { Box, Stack, Tooltip, Typography } from '@mui/material';
import Link from 'next/link';
import { fetcher, useSWRImmutable } from '@/lib/swr';
import { WEB_BUILD, WEB_VERSION } from '@/lib/web-version';

interface VersionInfo {
  gitVersion?: string;
  gitCommit?: string;
  buildDate?: string;
  goVersion?: string;
  platform?: string;
  [key: string]: string | undefined;
}

export default function VersionFooter() {
  // Version is effectively static for the life of a server build — fetch it
  // once and never revalidate on focus/reconnect.
  const { data: info } = useSWRImmutable<VersionInfo>('/api/v1/version', fetcher);

  const apiLabel = info?.gitVersion || '—';

  return (
    <Tooltip
      title={
        <Box sx={{ fontFamily: 'monospace', fontSize: 11 }}>
          <div>
            <strong>web</strong>: {WEB_VERSION}
          </div>
          {WEB_BUILD.gitCommit && <div>commit: {WEB_BUILD.gitCommit.slice(0, 12)}</div>}
          {WEB_BUILD.buildDate && <div>built: {WEB_BUILD.buildDate}</div>}
          {info ? (
            <>
              {info.gitVersion && (
                <div>
                  <strong>api</strong>: {info.gitVersion}
                </div>
              )}
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
          <Stack direction="row" spacing={1} alignItems="baseline" sx={{ overflow: 'hidden' }}>
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
          <Stack direction="row" spacing={1} alignItems="baseline" sx={{ overflow: 'hidden' }}>
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
