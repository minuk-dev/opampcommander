import { api } from './api-client';

// Shared agent-deletion helpers, used by both the agents list and detail pages so
// the confirmation copy and the API path stay in one place.

export function agentDeleteConfirmMessage(instanceUid: string): string {
  return `Permanently delete agent "${instanceUid}"? This removes its stored record; if the agent reconnects it will reappear.`;
}

export function deleteAgent(namespace: string, instanceUid: string): Promise<void> {
  return api.delete(`/api/v1/namespaces/${namespace}/agents/${instanceUid}`);
}

// Decode an OpAMP AgentCapabilities bitmask into the individual capability
// names. Shared by the agents list and detail pages so the bit table lives in
// one place.
export function capabilityNames(bitmask: number | undefined): string[] {
  if (!bitmask) return [];
  const table: Array<[number, string]> = [
    [1, 'ReportsStatus'],
    [2, 'AcceptsRemoteConfig'],
    [4, 'ReportsEffectiveConfig'],
    [8, 'AcceptsPackages'],
    [16, 'ReportsPackageStatuses'],
    [32, 'ReportsOwnTraces'],
    [64, 'ReportsOwnMetrics'],
    [128, 'ReportsOwnLogs'],
    [256, 'AcceptsOpAMPConnectionSettings'],
    [512, 'AcceptsOtherConnectionSettings'],
    [1024, 'AcceptsRestartCommand'],
    [2048, 'ReportsHealth'],
    [4096, 'ReportsRemoteConfig'],
    [8192, 'ReportsHeartbeat'],
    [16384, 'ReportsAvailableComponents'],
  ];
  return table.filter(([b]) => (bitmask & b) !== 0).map(([, name]) => name);
}
