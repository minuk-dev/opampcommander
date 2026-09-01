'use client';

import {
  ArrowRight,
  Boxes,
  Cable,
  CircleCheck,
  CircleSlash,
  Cpu,
  HeartPulse,
  Package,
  Server as ServerIcon,
  ShieldCheck,
  SlidersHorizontal,
} from 'lucide-react';
import Link from 'next/link';
import { type ComponentType, type ReactNode } from 'react';
import { cn } from '@shared/lib';
import { Badge, Card, PageHeader, Progress, Separator } from '@shared/ui';
import DashboardRefresh from './DashboardRefresh';
import type { Agent } from '@entities/agent';
import type { AgentGroup } from '@entities/agent-group';
import type { Connection } from '@entities/connection';
import type { Server } from '@entities/server';
import type { VersionInfo } from '@entities/version';

export interface DashboardData {
  agents: Agent[];
  agentTotal: number;
  groups: AgentGroup[];
  connections: Connection[];
  servers: Server[];
  packages: number | null;
  remoteConfigs: number | null;
  certificates: number | null;
  version: VersionInfo | null;
}

interface QuadrantProps {
  title: string;
  href: string;
  icon: ComponentType<{ className?: string; 'aria-hidden'?: boolean }>;
  children: ReactNode;
}

function Quadrant({ title, href, icon: Icon, children }: QuadrantProps) {
  return (
    <Card className="flex h-full flex-col">
      <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
        <Icon className="size-4 text-primary" aria-hidden />
        <h2 className="flex-1 text-sm font-semibold tracking-tight">{title}</h2>
        <Link
          href={href}
          aria-label={`view ${title}`}
          className="rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <ArrowRight className="size-4" aria-hidden />
        </Link>
      </div>
      <div className="flex-1 p-4">{children}</div>
    </Card>
  );
}

function Stat({
  label,
  value,
  tone,
}: {
  label: string;
  value: ReactNode;
  tone?: 'success' | 'warning' | 'default';
}) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p
        className={cn(
          'tnum text-xl font-semibold',
          tone === 'success' && 'text-success',
          tone === 'warning' && 'text-warning',
        )}
      >
        {value}
      </p>
    </div>
  );
}

export default function DashboardView({
  namespace,
  data,
}: {
  namespace: string;
  data: DashboardData;
}) {
  const agentCount = data.agents.length;
  const connectedCount = data.agents.filter((a) => a.status.connected).length;
  const healthyCount = data.agents.filter((a) => a.status.componentHealth?.healthy).length;
  const aliveConns = data.connections.filter((c) => c.alive).length;
  const aliveServers = data.servers.filter((s) =>
    s.conditions?.some((c) => c.type === 'Alive' && c.status === 'True'),
  ).length;
  const allHealthy = agentCount > 0 && healthyCount === agentCount;

  const topGroups = data.groups
    .toSorted((a, b) => b.status.numAgents - a.status.numAgents)
    .slice(0, 5);

  const resourceTiles = [
    { href: '/agentpackages', icon: Package, total: data.packages, label: 'Packages' },
    {
      href: '/agentremoteconfigs',
      icon: SlidersHorizontal,
      total: data.remoteConfigs,
      label: 'Remote Configs',
    },
    { href: '/certificates', icon: ShieldCheck, total: data.certificates, label: 'Certificates' },
  ];

  return (
    <div>
      <PageHeader
        title="Dashboard"
        subtitle={`Overview for namespace "${namespace}"`}
        actions={<DashboardRefresh />}
      />

      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        {/* Quadrant 1 — Agents */}
        <Quadrant title="Agents" href="/agents" icon={Cpu}>
          <div className="grid grid-cols-3 gap-3">
            <Stat label="Total" value={data.agentTotal} />
            <Stat
              label="Connected"
              value={`${connectedCount}/${agentCount}`}
              tone={connectedCount > 0 ? 'success' : 'default'}
            />
            <Stat
              label="Healthy"
              value={`${healthyCount}/${agentCount}`}
              tone={allHealthy ? 'success' : 'warning'}
            />
          </div>
          <div className="mt-3">
            <p className="mb-1 text-xs text-muted-foreground">Health on the current page</p>
            <Progress
              value={agentCount > 0 ? (healthyCount / agentCount) * 100 : 0}
              className={cn('[&>div]:bg-warning', allHealthy && '[&>div]:bg-success')}
            />
          </div>
          {agentCount === 0 && (
            <p className="mt-3 text-sm text-muted-foreground">
              No agents reported yet in this namespace.
            </p>
          )}
        </Quadrant>

        {/* Quadrant 2 — Agent Groups */}
        <Quadrant title="Agent Groups" href="/agentgroups" icon={Boxes}>
          {topGroups.length === 0 ? (
            <p className="text-sm text-muted-foreground">No agent groups yet.</p>
          ) : (
            <ul className="space-y-1">
              {topGroups.map((g) => (
                <li key={g.metadata.name}>
                  <Link
                    href={`/agents?agentGroup=${encodeURIComponent(g.metadata.name)}`}
                    className="flex items-center gap-2 rounded-md border border-border px-2.5 py-1.5 transition-colors hover:border-primary/50 hover:bg-accent"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{g.metadata.name}</p>
                      <p className="text-xs text-muted-foreground">
                        priority {g.spec.priority} · {g.status.numHealthyAgents}/
                        {g.status.numAgents} healthy
                      </p>
                    </div>
                    <Badge
                      variant={
                        g.status.numAgents > 0 && g.status.numConnectedAgents === g.status.numAgents
                          ? 'success'
                          : 'outline'
                      }
                      className="tnum"
                    >
                      {g.status.numConnectedAgents}/{g.status.numAgents}
                    </Badge>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </Quadrant>

        {/* Quadrant 3 — Resources */}
        <Quadrant title="Resources" href="/agentpackages" icon={Package}>
          <div className="grid grid-cols-3 divide-x divide-border">
            {resourceTiles.map((r) => {
              const Icon = r.icon;
              return (
                <Link
                  key={r.label}
                  href={r.href}
                  className="flex flex-col items-center gap-0.5 rounded-md px-1 py-1.5 transition-colors hover:bg-accent"
                >
                  <Icon className="size-4 text-primary" aria-hidden />
                  <span className="tnum text-xl font-semibold">{r.total ?? '—'}</span>
                  <span className="text-center text-xs text-muted-foreground">{r.label}</span>
                </Link>
              );
            })}
          </div>
        </Quadrant>

        {/* Quadrant 4 — Cluster */}
        <Quadrant title="Cluster" href="/servers" icon={ServerIcon}>
          <div className="grid grid-cols-2 gap-2">
            <Link
              href="/connections"
              className="flex items-center gap-2 rounded-md p-1.5 transition-colors hover:bg-accent"
            >
              <Cable className="size-4 text-primary" aria-hidden />
              <div>
                <p className="tnum text-base font-semibold">
                  {aliveConns}/{data.connections.length}
                </p>
                <p className="text-xs text-muted-foreground">Connections (alive)</p>
              </div>
            </Link>
            <Link
              href="/servers"
              className="flex items-center gap-2 rounded-md p-1.5 transition-colors hover:bg-accent"
            >
              {aliveServers > 0 ? (
                <CircleCheck className="size-4 text-success" aria-hidden />
              ) : (
                <CircleSlash className="size-4 text-warning" aria-hidden />
              )}
              <div>
                <p className="tnum text-base font-semibold">
                  {aliveServers}/{data.servers.length}
                </p>
                <p className="text-xs text-muted-foreground">Servers (alive)</p>
              </div>
            </Link>
          </div>
          <Separator className="my-2" />
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <HeartPulse className="size-3.5" aria-hidden />
            Server build:
            <span className="font-mono text-foreground">{data.version?.gitVersion ?? '—'}</span>
          </div>
        </Quadrant>
      </div>
    </div>
  );
}
