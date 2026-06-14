import { api } from '@shared/api';

// Shared agent-deletion helpers, used by both the agents list and detail pages so
// the confirmation copy and the API path stay in one place.

export function agentDeleteConfirmMessage(instanceUid: string): string {
  return `Permanently delete agent "${instanceUid}"? This removes its stored record; if the agent reconnects it will reappear.`;
}

export function deleteAgent(namespace: string, instanceUid: string): Promise<void> {
  return api.delete(`/api/v1/namespaces/${namespace}/agents/${instanceUid}`);
}
