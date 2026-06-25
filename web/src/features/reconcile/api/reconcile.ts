import { api } from '@shared/api';

// Reconcilable resource kinds, matching the server's reconcile registry
// (GET /api/v1/reconcile/kinds). For kind 'agent', `name` is the instance UID.
export type ReconcileKind = 'agent' | 'agentgroup' | 'agentremoteconfig';

// reconcileResource asks the server to re-run the side effects that normally fire when the
// named resource is created/updated, re-enforcing its domain rules on demand.
export function reconcileResource(
  kind: ReconcileKind,
  namespace: string,
  name: string,
): Promise<void> {
  return api.post(`/api/v1/namespaces/${namespace}/reconcile/${kind}/${encodeURIComponent(name)}`);
}
