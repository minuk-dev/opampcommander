'use client';

import Link from 'next/link';
import { fetcher, useSWRImmutable } from '@shared/api';
import { WEB_BUILD, WEB_VERSION } from '@shared/lib';
import { Tooltip } from '@shared/ui';

interface VersionInfo {
  gitVersion?: string;
  gitCommit?: string;
  buildDate?: string;
  goVersion?: string;
  platform?: string;
  [key: string]: string | undefined;
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex gap-2 overflow-hidden font-mono text-[11px]">
      <span className="w-6 shrink-0 text-muted-foreground">{label}</span>
      <span className="truncate">{value}</span>
    </div>
  );
}

export default function VersionFooter() {
  // Version is effectively static for the life of a server build — fetch it
  // once and never revalidate on focus/reconnect.
  const { data: info } = useSWRImmutable<VersionInfo>('/api/v1/version', fetcher);

  return (
    <Tooltip
      side="right"
      content={
        <div className="space-y-0.5 font-mono text-[11px]">
          <div>web: {WEB_VERSION}</div>
          {WEB_BUILD.gitCommit && <div>commit: {WEB_BUILD.gitCommit.slice(0, 12)}</div>}
          {WEB_BUILD.buildDate && <div>built: {WEB_BUILD.buildDate}</div>}
          {info ? (
            <>
              {info.gitVersion && <div>api: {info.gitVersion}</div>}
              {info.gitCommit && <div>commit: {info.gitCommit.slice(0, 12)}</div>}
              {info.buildDate && <div>built: {info.buildDate}</div>}
              {info.goVersion && <div>go: {info.goVersion}</div>}
              {info.platform && <div>platform: {info.platform}</div>}
            </>
          ) : (
            <div>api: loading…</div>
          )}
        </div>
      }
    >
      <Link
        href="/version"
        className="block border-t border-border px-3 py-2 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      >
        <Row label="web" value={WEB_VERSION} />
        <Row label="api" value={info?.gitVersion || '—'} />
      </Link>
    </Tooltip>
  );
}
