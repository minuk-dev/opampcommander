'use client';

import { ArrowLeft, CalendarDays, ListChecks, Pencil, RefreshCw, Users } from 'lucide-react';
import Link from 'next/link';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  ConfirmDialog,
  JsonBlock,
  PageHeader,
  Separator,
  Spinner,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Tooltip,
} from '@shared/ui';
import { cn } from '@shared/lib';
import { TimeDisplay } from '@shared/preferences';
import { ReconcileButton } from '@features/reconcile';
import { useNamespace } from '@entities/namespace';
import { api, useApi } from '@shared/api';
import type { AgentGroup } from '@entities/agent-group';
import dynamic from 'next/dynamic';

// Lazy-loaded: the edit dialog embeds the JSON/YAML editor (js-yaml), only
// needed once the user opens it — keep it out of the initial route bundle.
const AgentGroupEditDialog = dynamic(
  () => import('@features/agent-group-edit/ui/AgentGroupEditDialog'),
);
// Lazy-loaded: the remote-config picker fetches the namespace's configs, only
// needed once the user opens it — keep it out of the initial route bundle.
const SelectRemoteConfigDialog = dynamic(
  () => import('@features/apply-remote-config/ui/SelectRemoteConfigDialog'),
);

function AgentGroupDetailInner() {
  const params = useParams<{ name: string }>();
  const router = useRouter();
  const search = useSearchParams();
  const { namespace } = useNamespace();
  const [editing, setEditing] = useState(false);
  const [applyingConfig, setApplyingConfig] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [actionHandled, setActionHandled] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const {
    data: group,
    error: fetchError,
    isLoading: loading,
    mutate,
  } = useApi<AgentGroup>(`/api/v1/namespaces/${namespace}/agentgroups/${params.name}`);
  const fetchGroup = () => mutate();
  const error =
    actionError ??
    (fetchError instanceof Error
      ? fetchError.message
      : fetchError
        ? 'Failed to fetch group'
        : null);

  // Auto-trigger ?action= once after load (e.g. from the list page menu).
  useEffect(() => {
    if (!group || actionHandled) return;
    const action = search.get('action');
    if (!action) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setActionHandled(true);
    if (action === 'edit') {
      setEditing(true);
    } else if (action === 'apply') {
      setApplyingConfig(true);
    } else if (action === 'delete') {
      setDeleting(true);
    }
    router.replace(`/agentgroups/${params.name}`);
  }, [group, actionHandled, search, router, params.name]);

  const onDelete = async () => {
    try {
      await api.delete(`/api/v1/namespaces/${namespace}/agentgroups/${params.name}`);
      router.push('/agentgroups');
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to delete');
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <div className="mt-16 flex justify-center">
        <Spinner className="size-6" />
      </div>
    );
  }
  if (error || !group) {
    return (
      <div>
        <Button variant="ghost" size="sm" className="mb-3" onClick={() => router.back()}>
          <ArrowLeft aria-hidden />
          Back
        </Button>
        <Alert severity="error">{error || 'Group not found'}</Alert>
      </div>
    );
  }

  const agentsHref = `/agents?agentGroup=${encodeURIComponent(group.metadata.name)}`;
  const tiles = [
    { label: 'Total', value: group.status.numAgents, tone: 'default' as const },
    { label: 'Connected', value: group.status.numConnectedAgents, tone: 'success' as const },
    { label: 'Healthy', value: group.status.numHealthyAgents, tone: 'success' as const },
    { label: 'Unhealthy', value: group.status.numUnhealthyAgents, tone: 'warning' as const },
    { label: 'Not connected', value: group.status.numNotConnectedAgents, tone: 'default' as const },
  ];

  return (
    <div>
      <Button
        variant="ghost"
        size="sm"
        className="mb-2 -ml-2"
        onClick={() => router.push('/agentgroups')}
      >
        <ArrowLeft aria-hidden />
        Back to groups
      </Button>
      <PageHeader
        title={group.metadata.name}
        subtitle={`Namespace: ${group.metadata.namespace} · priority ${group.spec.priority}`}
        actions={
          <>
            <Button variant="ghost" size="icon-sm" aria-label="Refresh" onClick={fetchGroup}>
              <RefreshCw aria-hidden />
            </Button>
            <Button variant="outline" size="sm" asChild>
              <Link href={agentsHref}>
                <Users aria-hidden />
                View agents
              </Link>
            </Button>
            <Button variant="outline" size="sm" onClick={() => setApplyingConfig(true)}>
              <ListChecks aria-hidden />
              Apply remote configs
            </Button>
            <ReconcileButton
              kind="agentgroup"
              namespace={namespace}
              name={group.metadata.name}
              onReconciled={fetchGroup}
            />
            <Button size="sm" onClick={() => setEditing(true)}>
              <Pencil aria-hidden />
              Edit
            </Button>
          </>
        }
      />

      <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5">
        {tiles.map((tile) => (
          <Tooltip key={tile.label} content={`View agents in ${group.metadata.name}`}>
            <Link
              href={agentsHref}
              className="rounded-lg border border-border bg-card px-3 py-2 transition-colors hover:border-primary/40 hover:bg-accent"
            >
              <p className="text-xs text-muted-foreground">{tile.label}</p>
              <p
                className={cn(
                  'tnum text-xl font-semibold',
                  tile.tone === 'warning' && tile.value > 0 && 'text-warning',
                  tile.tone === 'success' && tile.value > 0 && 'text-success',
                )}
              >
                {tile.value}
              </p>
            </Link>
          </Tooltip>
        ))}
      </div>

      <Card className="mb-3">
        <CardContent className="flex flex-col gap-3 pt-4 md:flex-row md:items-start md:gap-6">
          <div className="flex items-center gap-2">
            <CalendarDays className="size-4 text-muted-foreground" aria-hidden />
            <div>
              <p className="text-xs text-muted-foreground">Created</p>
              <p className="text-sm">
                <TimeDisplay value={group.metadata.createdAt} />
              </p>
            </div>
          </div>
          {group.metadata.deletedAt && (
            <div>
              <p className="text-xs text-muted-foreground">Deleted</p>
              <p className="text-sm">
                <TimeDisplay value={group.metadata.deletedAt} />
              </p>
            </div>
          )}
          <Separator orientation="vertical" className="hidden h-10 md:block" />
          <div className="min-w-0">
            <p className="text-xs text-muted-foreground">Attributes</p>
            <div className="flex flex-wrap gap-1">
              {Object.entries(group.metadata.attributes || {}).length === 0 ? (
                <span className="text-sm text-muted-foreground">none</span>
              ) : (
                Object.entries(group.metadata.attributes).map(([k, v]) => (
                  <Badge key={k} variant="outline">
                    {k}={v}
                  </Badge>
                ))
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <Tabs defaultValue="selector">
          <TabsList className="px-2">
            <TabsTrigger value="selector">Selector</TabsTrigger>
            <TabsTrigger value="config">Agent config</TabsTrigger>
            <TabsTrigger value="conditions">Conditions</TabsTrigger>
            <TabsTrigger value="raw">Raw</TabsTrigger>
          </TabsList>
          <CardContent className="pt-4">
            <TabsContent value="selector" className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Agents are matched to this group when their identifying / non-identifying attributes
                contain all of the keys/values defined below.
              </p>
              <JsonBlock
                title="Identifying attributes"
                value={group.spec.selector.identifyingAttributes ?? {}}
              />
              <JsonBlock
                title="Non-identifying attributes"
                value={group.spec.selector.nonIdentifyingAttributes ?? {}}
              />
            </TabsContent>
            <TabsContent value="config">
              <JsonBlock value={group.spec.agentConfig ?? {}} />
            </TabsContent>
            <TabsContent value="conditions">
              <JsonBlock value={group.status.conditions ?? []} />
            </TabsContent>
            <TabsContent value="raw">
              <JsonBlock value={group} />
            </TabsContent>
          </CardContent>
        </Tabs>
      </Card>

      {editing && (
        <AgentGroupEditDialog
          open
          mode="edit"
          initial={group}
          onClose={() => setEditing(false)}
          onSaved={() => {
            setEditing(false);
            void fetchGroup();
          }}
        />
      )}
      {applyingConfig && (
        <SelectRemoteConfigDialog
          open
          namespace={namespace}
          group={group}
          onClose={() => setApplyingConfig(false)}
          onApplied={() => {
            setApplyingConfig(false);
            void fetchGroup();
          }}
        />
      )}
      <ConfirmDialog
        open={deleting}
        title="Delete agent group"
        message={`Delete "${group.metadata.name}"? This cannot be undone.`}
        confirmLabel="Delete"
        destructive
        onClose={() => setDeleting(false)}
        onConfirm={onDelete}
      />
    </div>
  );
}

export default function AgentGroupDetailPage() {
  return (
    <Suspense fallback={null}>
      <AgentGroupDetailInner />
    </Suspense>
  );
}
