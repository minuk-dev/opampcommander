import type { AgentRemoteConfig } from '../model/types';

export const SKIP_SCHEMA_VALIDATION = 'opampcommander.io/skip-schema-validation';

export function skipsSchemaValidation(config?: AgentRemoteConfig): boolean {
  return ['1', 't', 'T', 'true', 'TRUE', 'True'].includes(
    config?.metadata.attributes?.[SKIP_SCHEMA_VALIDATION] ?? '',
  );
}

export function schemaRefsSource(config: AgentRemoteConfig): string {
  if (!config.spec.schemaRefs?.length) return 'No schema pinned';
  const source = config.status?.schemaRefsSource;
  if (source === 'auto') return 'Auto-resolved';
  if (source === 'explicit') return 'Explicitly set';
  return 'Source unavailable (existing config)';
}
