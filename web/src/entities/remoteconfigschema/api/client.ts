import { api, type ListResponse } from '@shared/api';
import type { RemoteConfigSchema } from '../model/types';

export const remoteConfigSchemasPath = (namespace: string) =>
  `/api/v1/namespaces/${encodeURIComponent(namespace)}/remoteconfigschemas`;

export async function listRemoteConfigSchemas(namespace: string): Promise<RemoteConfigSchema[]> {
  const items: RemoteConfigSchema[] = [];
  let cursor = '';
  do {
    const page = await api.get<ListResponse<RemoteConfigSchema>>(
      remoteConfigSchemasPath(namespace),
      {
        query: { limit: 100, continue: cursor },
      },
    );
    items.push(...page.items);
    cursor = page.metadata.continue;
  } while (cursor);
  return items;
}
