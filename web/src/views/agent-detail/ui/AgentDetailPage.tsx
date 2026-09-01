'use client';

import { ArrowLeft, Pencil, RefreshCw, RotateCcw, Trash2 } from 'lucide-react';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  ConfirmDialog,
  JsonBlock,
  PageHeader,
  Spinner,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@shared/ui';
import { TimeDisplay } from '@shared/preferences';
import { ReconcileButton } from '@features/reconcile';
import { useNamespace } from '@entities/namespace';
import { api, useApi } from '@shared/api';
import {
  agentDeleteConfirmMessage,
  agentTypeLabel,
  capabilityNames,
  deleteAgent,
  isOtelCollector,
  type Agent,
} from '@entities/agent';
import dynamic from 'next/dynamic';

// Lazy-loaded: the edit dialog embeds the JSON/YAML editor (js-yaml), only
// needed once the user opens it — keep it out of the initial route bundle.
const AgentEditDialog = dynamic(() => import('@features/agent-edit/ui/AgentEditDialog'));

function AgentDetailInner() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const search = useSearchParams();
  const { namespace } = useNamespace();
  const [editOpen, setEditOpen] = useState(false);
  const [restartBusy, setRestartBusy] = useState(false);
  const [actionHandled, setActionHandled] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const {
    data: agent,
    error: fetchError,
    isLoading: loading,
    mutate,
  } = useApi<Agent>(`/api/v1/namespaces/${namespace}/agents/${params.id}`);
  const fetchAgent = () => mutate();
  const error =
    actionError ??
    (fetchError instanceof Error
      ? fetchError.message
      : fetchError
        ? 'Failed to fetch agent'
        : null);

  const requestRestart = useCallback(async () => {
    if (!agent) return;
    setRestartBusy(true);
    try {
      const next: Agent = {
        ...agent,
        spec: {
          ...(agent.spec ?? {}),
          restartRequiredAt: new Date().toISOString(),
        },
      };
      const updated = await api.put<Agent>(
        `/api/v1/namespaces/${namespace}/agents/${params.id}`,
        next,
      );
      // Seed the cache with the server response; no need to refetch.
      await mutate(updated, { revalidate: false });
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to set restart');
    } finally {
      setRestartBusy(false);
    }
  }, [agent, namespace, params.id, mutate]);

  const onDeleteAgent = useCallback(async () => {
    await deleteAgent(namespace, params.id);
    setDeleteOpen(false);
    router.push('/agents');
  }, [namespace, params.id, router]);

  // Honor ?action= once after the agent loads.
  useEffect(() => {
    if (!agent || actionHandled) return;
    const action = search.get('action');
    if (!action) return;
    setActionHandled(true);
    if (action === 'edit') {
      setEditOpen(true);
    } else if (action === 'restart') {
      void requestRestart();
    }
    router.replace(`/agents/${params.id}`);
  }, [agent, actionHandled, search, router, params.id, requestRestart]);

  const capabilities = useMemo(
    () => capabilityNames(agent?.metadata.capabilities),
    [agent?.metadata.capabilities],
  );

  if (loading) {
    return (
      <div className="mt-16 flex justify-center">
        <Spinner className="size-6" />
      </div>
    );
  }

  if (error || !agent) {
    return (
      <div>
        <Button variant="ghost" size="sm" className="mb-3" onClick={() => router.back()}>
          <ArrowLeft aria-hidden />
          Back
        </Button>
        <Alert severity="error">{error || 'Agent not found'}</Alert>
      </div>
    );
  }

  const health = agent.status.componentHealth;
  const effectiveConfig = Object.entries(agent.status.effectiveConfig?.configMap.configMap ?? {});

  return (
    <div>
      <Button
        variant="ghost"
        size="sm"
        className="mb-2 -ml-2"
        onClick={() => router.push('/agents')}
      >
        <ArrowLeft aria-hidden />
        Back to agents
      </Button>
      <PageHeader
        title={agent.metadata.instanceUid}
        subtitle={`Namespace: ${agent.metadata.namespace} · ${agentTypeLabel(agent.metadata.type)}`}
        actions={
          <>
            <Button variant="ghost" size="icon-sm" aria-label="Refresh" onClick={fetchAgent}>
              <RefreshCw aria-hidden />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => void requestRestart()}
              disabled={restartBusy}
            >
              <RotateCcw aria-hidden />
              Request restart
            </Button>
            <ReconcileButton
              kind="agent"
              namespace={namespace}
              name={agent.metadata.instanceUid}
              onReconciled={fetchAgent}
            />
            {!agent.status.connected && (
              <Button variant="outline" size="sm" onClick={() => setDeleteOpen(true)}>
                <Trash2 aria-hidden />
                Delete
              </Button>
            )}
            <Button size="sm" onClick={() => setEditOpen(true)}>
              <Pencil aria-hidden />
              Edit spec
            </Button>
          </>
        }
      />

      <div className="mb-3 grid gap-3 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle>Connection</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1.5">
            <div className="flex flex-wrap gap-1">
              <Badge variant={agent.status.connected ? 'success' : 'muted'}>
                {agent.status.connected ? 'Connected' : 'Disconnected'}
              </Badge>
              {agent.status.connectionType && (
                <Badge variant="outline">{agent.status.connectionType}</Badge>
              )}
              <Badge variant={isOtelCollector(agent.metadata.type) ? 'primary' : 'outline'}>
                {agentTypeLabel(agent.metadata.type)}
              </Badge>
            </div>
            <p className="text-sm">
              Last reported: <TimeDisplay value={agent.status.lastReportedAt} />
            </p>
            <p className="text-sm">Sequence #: {agent.status.sequenceNum ?? '—'}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Health</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1.5">
            <div className="flex flex-wrap gap-1">
              <Badge variant={health?.healthy ? 'success' : 'warning'}>
                {health?.healthy ? 'Healthy' : 'Unhealthy'}
              </Badge>
              {health?.status && <Badge variant="outline">{health.status}</Badge>}
            </div>
            {health?.lastError && <p className="text-sm text-destructive">{health.lastError}</p>}
            <p className="text-sm">Started: {health?.startTime || '—'}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Capabilities</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-1">
              {capabilities.length === 0 ? (
                <span className="text-sm text-muted-foreground">None</span>
              ) : (
                capabilities.map((c) => (
                  <Badge key={c} variant="outline">
                    {c}
                  </Badge>
                ))
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <Tabs defaultValue="description">
          <TabsList className="overflow-x-auto px-2">
            <TabsTrigger value="description">Description</TabsTrigger>
            <TabsTrigger value="effective">Effective config</TabsTrigger>
            <TabsTrigger value="spec">Spec</TabsTrigger>
            <TabsTrigger value="conditions">Conditions</TabsTrigger>
            <TabsTrigger value="raw">Raw</TabsTrigger>
          </TabsList>
          <CardContent className="pt-4">
            <TabsContent value="description" className="space-y-3">
              <JsonBlock
                title="Identifying attributes"
                value={agent.metadata.description?.identifyingAttributes ?? {}}
              />
              <JsonBlock
                title="Non-identifying attributes"
                value={agent.metadata.description?.nonIdentifyingAttributes ?? {}}
              />
              <JsonBlock
                title="Custom capabilities"
                value={agent.metadata.customCapabilities?.capabilities ?? []}
              />
            </TabsContent>
            <TabsContent value="effective" className="space-y-3">
              {effectiveConfig.length === 0 ? (
                <p className="text-sm text-muted-foreground">No effective config reported.</p>
              ) : (
                effectiveConfig.map(([name, file]) => (
                  <JsonBlock key={name} title={`${name} (${file.contentType})`} value={file.body} />
                ))
              )}
            </TabsContent>
            <TabsContent value="spec">
              <JsonBlock value={agent.spec ?? {}} />
            </TabsContent>
            <TabsContent value="conditions">
              <JsonBlock value={agent.status.conditions ?? []} />
            </TabsContent>
            <TabsContent value="raw">
              <JsonBlock value={agent} />
            </TabsContent>
          </CardContent>
        </Tabs>
      </Card>

      {editOpen && (
        <AgentEditDialog
          open
          agent={agent}
          onClose={() => setEditOpen(false)}
          onSaved={(saved) => {
            void mutate(saved, { revalidate: false });
            setEditOpen(false);
          }}
        />
      )}

      <ConfirmDialog
        open={deleteOpen}
        title="Delete agent"
        message={agentDeleteConfirmMessage(agent.metadata.instanceUid)}
        confirmLabel="Delete"
        destructive
        onClose={() => setDeleteOpen(false)}
        onConfirm={onDeleteAgent}
      />
    </div>
  );
}

export default function AgentDetailPage() {
  return (
    <Suspense fallback={null}>
      <AgentDetailInner />
    </Suspense>
  );
}
