import { readNamespace, serverGet } from '@shared/api/server';
import type { Agent } from '@entities/agent';
import type { AgentGroup } from '@entities/agent-group';
import type { Connection } from '@entities/connection';
import type { Server } from '@entities/server';
import type { VersionInfo } from '@entities/version';
import type { ListResponse } from '@shared/api';
import { DashboardView, type DashboardData } from '@widgets/dashboard';

async function safeServerList<T>(path: string): Promise<{ items: T[]; total: number | null }> {
  try {
    const res = await serverGet<ListResponse<T>>(`${path}?limit=200`);
    const items = res.items ?? [];
    return {
      items,
      total: items.length + (res.metadata?.remainingItemCount ?? 0),
    };
  } catch {
    return { items: [], total: null };
  }
}

async function loadDashboard(namespace: string): Promise<DashboardData> {
  const [agents, groups, conns, servers, packages, configs, certs, version] = await Promise.all([
    safeServerList<Agent>(`/api/v1/namespaces/${namespace}/agents`),
    safeServerList<AgentGroup>(`/api/v1/namespaces/${namespace}/agentgroups`),
    safeServerList<Connection>(`/api/v1/namespaces/${namespace}/connections`),
    safeServerList<Server>('/api/v1/servers'),
    safeServerList<unknown>(`/api/v1/namespaces/${namespace}/agentpackages`),
    safeServerList<unknown>(`/api/v1/namespaces/${namespace}/agentremoteconfigs`),
    safeServerList<unknown>(`/api/v1/namespaces/${namespace}/certificates`),
    serverGet<VersionInfo>('/api/v1/version').catch(() => null),
  ]);
  return {
    agents: agents.items,
    agentTotal: agents.total ?? agents.items.length,
    groups: groups.items,
    connections: conns.items,
    servers: servers.items,
    packages: packages.total,
    remoteConfigs: configs.total,
    certificates: certs.total,
    version,
  };
}

// Server Component: fetch all dashboard data server-side (parallel) using the
// session cookie, then hand the serializable result to the client view, which
// owns the interactive bits (links, refresh) that can't live in an RSC.
export default async function DashboardPage() {
  const namespace = await readNamespace();
  const data = await loadDashboard(namespace);
  return <DashboardView namespace={namespace} data={data} />;
}
