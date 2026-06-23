import type { AgentGroup, AgentGroupRemoteConfig } from '../model/types';

/** Every remote-config entry declared on a group (refs and inline). */
export function remoteConfigs(g: AgentGroup | null | undefined): AgentGroupRemoteConfig[] {
  return g?.spec.agentConfig?.agentRemoteConfigs ?? [];
}

/** The standalone-resource refs a group currently points at, in declared order. */
export function remoteConfigRefs(g: AgentGroup | null | undefined): string[] {
  return remoteConfigs(g)
    .map((rc) => rc.agentRemoteConfigRef)
    .filter((ref): ref is string => !!ref);
}

/** Inline (non-ref) entries on a group, preserved when the UI rewrites refs. */
export function inlineRemoteConfigs(g: AgentGroup | null | undefined): AgentGroupRemoteConfig[] {
  return remoteConfigs(g).filter((rc) => !rc.agentRemoteConfigRef);
}

/**
 * Describe how a group currently sources its remote config (named refs and/or
 * inline specs), so the UI can show what is applied. Returns 'none' when empty.
 */
export function describeRemoteConfigSources(g: AgentGroup | null | undefined): string {
  const list = remoteConfigs(g);
  if (list.length === 0) return 'none';
  return list
    .map((rc) => {
      if (rc.agentRemoteConfigRef) return `ref → ${rc.agentRemoteConfigRef}`;
      if (rc.agentRemoteConfigName) return `inline (${rc.agentRemoteConfigName})`;
      return 'inline';
    })
    .join(', ');
}

/** Whether the group currently has any remote config (a ref or inline) set. */
export function hasRemoteConfig(g: AgentGroup | null | undefined): boolean {
  return remoteConfigs(g).length > 0;
}

/**
 * Build the `agentRemoteConfigs` list for a group whose ref set is being
 * replaced with `refs`, preserving any inline (non-ref) entries already present.
 * Duplicate refs are collapsed so the same resource is never applied twice.
 */
export function withRemoteConfigRefs(
  g: AgentGroup | null | undefined,
  refs: string[],
): AgentGroupRemoteConfig[] {
  const unique = Array.from(new Set(refs));
  return [...inlineRemoteConfigs(g), ...unique.map((ref) => ({ agentRemoteConfigRef: ref }))];
}
