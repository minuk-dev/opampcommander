import type { AgentGroup } from '../model/types';

/**
 * Describe how a group currently sources its remote config (a named ref, an
 * inline spec, or none), so the UI can show what applying a new config would
 * overwrite.
 */
export function describeRemoteConfigSource(g: AgentGroup): string {
  const rc = g.spec.agentConfig?.agentRemoteConfig;
  if (!rc) return 'none';
  if (rc.agentRemoteConfigRef) return `ref → ${rc.agentRemoteConfigRef}`;
  if (rc.agentRemoteConfigName) return `inline (${rc.agentRemoteConfigName})`;
  return 'none';
}

/** The remote-config ref a group currently points at, or '' if none. */
export function currentRemoteConfigRef(g: AgentGroup | null | undefined): string {
  return g?.spec.agentConfig?.agentRemoteConfig?.agentRemoteConfigRef ?? '';
}

/** Whether the group currently has any remote config (a ref or inline) set. */
export function hasRemoteConfig(g: AgentGroup | null | undefined): boolean {
  const rc = g?.spec.agentConfig?.agentRemoteConfig;
  return !!(rc?.agentRemoteConfigRef || rc?.agentRemoteConfigName || rc?.agentRemoteConfigSpec);
}
