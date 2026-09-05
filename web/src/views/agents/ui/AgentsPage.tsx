'use client';

import { Eye, Pencil, RefreshCw, RotateCcw, Search, Trash2, Users, X } from 'lucide-react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  ColumnPicker,
  ConfirmDialog,
  Field,
  Input,
  Label,
  PageHeader,
  PaginationFooter,
  RowActionsMenu,
  type RowAction,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Spinner,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
  TableWrap,
  Tooltip,
} from '@shared/ui';
import { TimeDisplay } from '@shared/preferences';
import { useNamespace } from '@entities/namespace';
import {
  agentDeleteConfirmMessage,
  agentTypeLabel,
  capabilityNames,
  deleteAgent,
  isOtelCollector,
  type Agent,
} from '@entities/agent';
import { cn, type ColumnConfig, useColumnVisibility, useCursorPagination } from '@shared/lib';
import { useApi, type ListResponse } from '@shared/api';
import type { AgentGroup } from '@entities/agent-group';

type SearchMode = 'uid' | 'group' | 'description' | 'attributes' | 'nattribute';

// `lcNeedle` is expected to be pre-lowercased by the caller so we don't redo
// it for every agent in a filter pass.
function attrMatchesDescription(agent: Agent, lcNeedle: string): boolean {
  const desc = agent.metadata.description;
  const collect = [
    ...Object.entries(desc?.identifyingAttributes ?? {}),
    ...Object.entries(desc?.nonIdentifyingAttributes ?? {}),
  ];
  return collect.some(
    ([k, v]) => k.toLowerCase().includes(lcNeedle) || v.toLowerCase().includes(lcNeedle),
  );
}

// Columns for the agents table. `instanceUid` is locked (the row identifier);
// `sequence` and the verbose capability/attribute columns are off by default so
// the table stays readable, and users opt into them via the column picker.
const AGENT_COLUMNS: ColumnConfig[] = [
  { id: 'instanceUid', label: 'Instance UID', locked: true },
  { id: 'connected', label: 'Connected' },
  { id: 'healthy', label: 'Healthy' },
  { id: 'agentType', label: 'Type' },
  // `type` predates the agent Type column and shows the connection transport
  // (HTTP/WebSocket); keep the id for persisted visibility, clarify the label.
  { id: 'type', label: 'Connection' },
  { id: 'lastReported', label: 'Last Reported' },
  { id: 'sequence', label: 'Sequence', defaultVisible: false },
  { id: 'capabilities', label: 'Capabilities', defaultVisible: false },
  {
    id: 'identifyingAttributes',
    label: 'Description (identifying attributes)',
    defaultVisible: false,
  },
  {
    id: 'nonIdentifyingAttributes',
    label: 'Description (non-identifying attributes)',
    defaultVisible: false,
  },
];

// Render an attribute map as compact key=value badges for a table cell. When
// `onSelect` is provided the badges become buttons that search by that exact
// attribute; clicks are kept from bubbling up to the row's navigation.
function AttrBadges({
  attrs,
  onSelect,
}: {
  attrs: Record<string, string> | undefined;
  onSelect?: (key: string, value: string) => void;
}) {
  const entries = Object.entries(attrs ?? {});
  if (entries.length === 0) return <>-</>;
  return (
    <div className="flex max-w-80 flex-wrap gap-1">
      {entries.map(([k, v]) =>
        onSelect ? (
          <button
            key={k}
            type="button"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onSelect(k, v);
            }}
          >
            <Badge variant="outline" className="hover:border-primary/50 hover:bg-accent">
              {k}={v}
            </Badge>
          </button>
        ) : (
          <Badge key={k} variant="outline">
            {k}={v}
          </Badge>
        ),
      )}
    </div>
  );
}

// A removable filter pill for the active search criteria.
function FilterChip({ label, onClear }: { label: string; onClear: () => void }) {
  return (
    <Badge variant="primary" className="gap-1 pr-1">
      {label}
      <button
        type="button"
        onClick={onClear}
        aria-label={`Clear ${label}`}
        className="rounded-full p-0.5 hover:bg-primary/20"
      >
        <X className="size-3" aria-hidden />
      </button>
    </Badge>
  );
}

function AgentsInner() {
  const router = useRouter();
  const search = useSearchParams();
  const { namespace } = useNamespace();

  const agentGroupParam = search.get('agentGroup') || '';
  const qParam = search.get('q') || '';
  const descParam = search.get('desc') || '';
  // `selector` and `labelSelector` are what this page sent before the API split
  // the metadata selectors apart. Both carried the same `key=value` grammar over
  // the same attributes, so old bookmarks keep working by being read as the
  // attribute selector they were always describing.
  const attributeSelectorParam =
    search.get('attributeSelector') || search.get('labelSelector') || search.get('selector') || '';
  const nonIdentifyingSelectorParam = search.get('nonIdentifyingSelector') || '';
  const modeParam =
    (search.get('mode') as SearchMode | null) ||
    (agentGroupParam
      ? 'group'
      : descParam
        ? 'description'
        : attributeSelectorParam
          ? 'attributes'
          : nonIdentifyingSelectorParam
            ? 'nattribute'
            : 'uid');

  const [mode, setMode] = useState<SearchMode>(modeParam);
  const [query, setQuery] = useState(qParam);
  const [deleting, setDeleting] = useState<Agent | null>(null);
  // Disconnected agents are hidden by default; the toggle reveals them. When
  // hidden we pass connected=true so the server filters them out (keeping the
  // paginated total accurate) rather than filtering the current page client-side.
  const [showDisconnected, setShowDisconnected] = useState(false);

  const { visible, isVisible, toggle } = useColumnVisibility('agents', AGENT_COLUMNS);
  // +1 for the always-present Actions column.
  const colSpan = AGENT_COLUMNS.filter((c) => isVisible(c.id)).length + 1;

  // The search modes hit different endpoints: group lists a group's agents, UID
  // hits the search endpoint, attributes filter the plain list in the datastore,
  // and description reuses the plain list but filters client-side (see
  // visibleAgents below) — the API has no substring search over attribute values.
  //
  // An agent is selected on attributeSelector rather than labelSelector: its
  // description is reported over OpAMP, not metadata an operator set, and the
  // server answers labelSelector here with a 400 saying so.
  let listPath: string;
  const listQuery: Record<string, string> = {};
  if (agentGroupParam) {
    listPath = `/api/v1/namespaces/${namespace}/agentgroups/${agentGroupParam}/agents`;
  } else if (qParam) {
    listPath = `/api/v1/namespaces/${namespace}/agents/search`;
    listQuery.q = qParam;
  } else {
    listPath = `/api/v1/namespaces/${namespace}/agents`;
    if (attributeSelectorParam) {
      listQuery.attributeSelector = attributeSelectorParam;
    }
    if (nonIdentifyingSelectorParam) {
      listQuery.nonIdentifyingSelector = nonIdentifyingSelectorParam;
    }
  }
  if (!showDisconnected) {
    listQuery.connected = 'true';
  }

  const pagination = useCursorPagination<Agent>(listPath, { query: listQuery });
  const { items: agents, isLoading: loading, error: fetchError } = pagination;
  const error =
    fetchError instanceof Error ? fetchError.message : fetchError ? 'Failed to fetch agents' : null;

  // Group list for the "Group" autocomplete + chip cross-reference. SWR dedupes
  // this with the same fetch on other pages and silently no-ops on RBAC denial.
  const { data: groupData } = useApi<ListResponse<AgentGroup>>([
    `/api/v1/namespaces/${namespace}/agentgroups`,
    { limit: 500 },
  ]);
  const groupOptions = groupData?.items ?? [];

  // Keep the local input synced when URL drives the value (mode change etc.)
  useEffect(() => {
    if (mode === 'uid') setQuery(qParam);
    else if (mode === 'description') setQuery(descParam);
    else if (mode === 'group') setQuery(agentGroupParam);
    else if (mode === 'attributes') setQuery(attributeSelectorParam);
    else if (mode === 'nattribute') setQuery(nonIdentifyingSelectorParam);
  }, [
    mode,
    qParam,
    descParam,
    agentGroupParam,
    attributeSelectorParam,
    nonIdentifyingSelectorParam,
  ]);

  // Sync mode state if URL changes externally
  useEffect(() => {
    setMode(modeParam);
  }, [modeParam]);

  const updateUrl = (next: {
    q?: string;
    agentGroup?: string;
    desc?: string;
    attributeSelector?: string;
    nonIdentifyingSelector?: string;
    mode?: SearchMode;
  }) => {
    const params = new URLSearchParams();
    const m = next.mode ?? mode;
    const q = next.q ?? (m === 'uid' ? qParam : '');
    const g = next.agentGroup ?? (m === 'group' ? agentGroupParam : '');
    const d = next.desc ?? (m === 'description' ? descParam : '');
    const s = next.attributeSelector ?? (m === 'attributes' ? attributeSelectorParam : '');
    const ns =
      next.nonIdentifyingSelector ?? (m === 'nattribute' ? nonIdentifyingSelectorParam : '');
    if (q) params.set('q', q);
    if (g) params.set('agentGroup', g);
    if (d) params.set('desc', d);
    if (s) params.set('attributeSelector', s);
    if (ns) params.set('nonIdentifyingSelector', ns);
    if (m !== 'uid' || !q) params.set('mode', m);
    const qs = params.toString();
    router.replace(qs ? `/agents?${qs}` : '/agents');
  };

  const onSearch = (e: React.FormEvent) => {
    e.preventDefault();
    const value = query.trim();
    if (mode === 'uid') {
      updateUrl({
        q: value,
        agentGroup: '',
        desc: '',
        attributeSelector: '',
        nonIdentifyingSelector: '',
        mode: 'uid',
      });
    } else if (mode === 'group') {
      updateUrl({
        agentGroup: value,
        q: '',
        desc: '',
        attributeSelector: '',
        nonIdentifyingSelector: '',
        mode: 'group',
      });
    } else if (mode === 'attributes') {
      updateUrl({
        attributeSelector: value,
        q: '',
        agentGroup: '',
        desc: '',
        nonIdentifyingSelector: '',
        mode: 'attributes',
      });
    } else if (mode === 'nattribute') {
      updateUrl({
        nonIdentifyingSelector: value,
        q: '',
        agentGroup: '',
        desc: '',
        attributeSelector: '',
        mode: 'nattribute',
      });
    } else {
      updateUrl({
        desc: value,
        q: '',
        agentGroup: '',
        attributeSelector: '',
        nonIdentifyingSelector: '',
        mode: 'description',
      });
    }
  };

  const setSearchMode = (next: SearchMode) => {
    setMode(next);
    setQuery('');
    updateUrl({
      q: '',
      agentGroup: '',
      desc: '',
      attributeSelector: '',
      nonIdentifyingSelector: '',
      mode: next,
    });
  };

  // Triggered by clicking an identifying-attribute chip: jump to an exact,
  // server-side label selector for that attribute.
  const searchByAttribute = (key: string, value: string) => {
    const attributeSelector = `${key}=${value}`;
    setMode('attributes');
    setQuery(attributeSelector);
    updateUrl({
      attributeSelector,
      q: '',
      agentGroup: '',
      desc: '',
      nonIdentifyingSelector: '',
      mode: 'attributes',
    });
  };

  // Triggered by clicking a non-identifying-attribute chip: jump to an exact,
  // server-side search for that attribute.
  const searchByNonIdentifyingAttribute = (key: string, value: string) => {
    const selector = `${key}=${value}`;
    setMode('nattribute');
    setQuery(selector);
    updateUrl({
      nonIdentifyingSelector: selector,
      q: '',
      agentGroup: '',
      desc: '',
      attributeSelector: '',
      mode: 'nattribute',
    });
  };

  const clearGroup = () => updateUrl({ agentGroup: '' });
  const clearSearch = () => {
    setQuery('');
    updateUrl({ q: '' });
  };

  // Only disconnected agents can be deleted; a connected one would just be
  // recreated on its next report (the server enforces this with a 409 too).
  // reset() (not refresh()) revalidates from page 0 so deleting the last row on a
  // later page doesn't strand the user on a now-empty cursor page.
  const onDelete = async () => {
    if (!deleting) return;
    await deleteAgent(namespace, deleting.metadata.instanceUid);
    setDeleting(null);
    pagination.reset();
  };

  const filterActive = Boolean(
    agentGroupParam || qParam || descParam || attributeSelectorParam || nonIdentifyingSelectorParam,
  );
  const lcDesc = descParam.toLowerCase();
  const visibleAgents = descParam
    ? agents.filter((a) => attrMatchesDescription(a, lcDesc))
    : agents;

  const searchPlaceholder =
    mode === 'uid'
      ? 'Instance UID contains… (server-side)'
      : mode === 'attributes'
        ? 'service.name=otel-collector, os.type in (linux,darwin) (server-side)'
        : mode === 'nattribute'
          ? 'key=value non-identifying attribute (exact, deprecated — use Attributes)'
          : 'Attribute key/value contains… (client-side, current page)';

  return (
    <div>
      <PageHeader
        title="Agents"
        subtitle={`Namespace: ${namespace}`}
        actions={
          <>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label="Refresh"
              onClick={() => pagination.refresh()}
            >
              <RefreshCw className={cn(loading && 'animate-spin')} aria-hidden />
            </Button>
            <ColumnPicker columns={AGENT_COLUMNS} visible={visible} onToggle={toggle} />
          </>
        }
      />

      <Card className="mb-3 p-3">
        <form onSubmit={onSearch} className="flex flex-col gap-2 sm:flex-row">
          <Field label="Search by" className="sm:w-52">
            {(field) => (
              <Select value={mode} onValueChange={(v) => setSearchMode(v as SearchMode)}>
                <SelectTrigger {...field}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="group">Agent Group</SelectItem>
                  <SelectItem value="uid">Instance UID</SelectItem>
                  <SelectItem value="attributes">Attributes (reported by the agent)</SelectItem>
                  <SelectItem value="nattribute">Non-identifying Attribute</SelectItem>
                  <SelectItem value="description">Description</SelectItem>
                </SelectContent>
              </Select>
            )}
          </Field>

          <Field label={mode === 'group' ? 'Agent group' : 'Query'} className="flex-1">
            {(field) =>
              mode === 'group' ? (
                <>
                  {/* Native datalist keeps the free-text + suggestions behaviour
                      of the old free-solo autocomplete without a popup widget. */}
                  <Input
                    {...field}
                    list="agent-group-options"
                    startSlot={<Users aria-hidden />}
                    placeholder="Select or type an agent group name…"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                  />
                  <datalist id="agent-group-options">
                    {groupOptions.map((g) => (
                      <option key={g.metadata.name} value={g.metadata.name} />
                    ))}
                  </datalist>
                </>
              ) : (
                <Input
                  {...field}
                  startSlot={<Search aria-hidden />}
                  placeholder={searchPlaceholder}
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                />
              )
            }
          </Field>

          <Button type="submit" className="self-end">
            Search
          </Button>
        </form>

        <Label className="mt-2 flex w-fit cursor-pointer items-center gap-1.5">
          <Switch checked={showDisconnected} onCheckedChange={setShowDisconnected} />
          Show disconnected agents
        </Label>

        {mode === 'description' && (
          <p className="mt-2 text-xs text-muted-foreground">
            Description search filters the current page on the client. Combine with a smaller
            namespace or refine via UID / group for large deployments.
          </p>
        )}

        {filterActive && (
          <div className="mt-2 flex flex-wrap items-center gap-1.5">
            <span className="text-xs text-muted-foreground">Filters:</span>
            {agentGroupParam && (
              <Tooltip content="Open agent group">
                <span>
                  <FilterChip label={`Group: ${agentGroupParam}`} onClear={clearGroup} />
                </span>
              </Tooltip>
            )}
            {qParam && <FilterChip label={`UID contains: ${qParam}`} onClear={clearSearch} />}
            {descParam && (
              <FilterChip
                label={`Description: ${descParam}`}
                onClear={() => {
                  setQuery('');
                  updateUrl({ desc: '' });
                }}
              />
            )}
            {attributeSelectorParam && (
              <FilterChip
                label={`Attributes: ${attributeSelectorParam}`}
                onClear={() => {
                  setQuery('');
                  updateUrl({ attributeSelector: '' });
                }}
              />
            )}
            {nonIdentifyingSelectorParam && (
              <FilterChip
                label={`Non-identifying: ${nonIdentifyingSelectorParam}`}
                onClear={() => {
                  setQuery('');
                  updateUrl({ nonIdentifyingSelector: '' });
                }}
              />
            )}
          </div>
        )}
      </Card>

      {error && (
        <Alert severity="error" className="mb-3">
          {error}
        </Alert>
      )}

      <TableWrap>
        <Table>
          <TableHead>
            <TableRow className="hover:bg-transparent">
              {isVisible('instanceUid') && <TableHeaderCell>Instance UID</TableHeaderCell>}
              {isVisible('connected') && <TableHeaderCell>Connected</TableHeaderCell>}
              {isVisible('healthy') && <TableHeaderCell>Healthy</TableHeaderCell>}
              {isVisible('agentType') && <TableHeaderCell>Type</TableHeaderCell>}
              {isVisible('type') && <TableHeaderCell>Connection</TableHeaderCell>}
              {isVisible('lastReported') && <TableHeaderCell>Last Reported</TableHeaderCell>}
              {isVisible('sequence') && <TableHeaderCell>Sequence</TableHeaderCell>}
              {isVisible('capabilities') && <TableHeaderCell>Capabilities</TableHeaderCell>}
              {isVisible('identifyingAttributes') && (
                <TableHeaderCell>Description (identifying attributes)</TableHeaderCell>
              )}
              {isVisible('nonIdentifyingAttributes') && (
                <TableHeaderCell>Description (non-identifying attributes)</TableHeaderCell>
              )}
              <TableHeaderCell className="w-10 text-right">Actions</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={colSpan} className="py-8">
                  <Spinner className="mx-auto size-5" />
                </TableCell>
              </TableRow>
            ) : visibleAgents.length === 0 ? (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={colSpan} className="py-8 text-center text-muted-foreground">
                  {descParam
                    ? `No agents on this page match description "${descParam}"`
                    : 'No agents found'}
                </TableCell>
              </TableRow>
            ) : (
              visibleAgents.map((agent) => {
                const href = `/agents/${agent.metadata.instanceUid}`;
                const caps = capabilityNames(agent.metadata.capabilities);
                return (
                  <TableRow
                    key={agent.metadata.instanceUid}
                    // The whole row navigates, but the UID cell still holds a
                    // real link so keyboard and middle-click work.
                    onClick={() => router.push(href)}
                    className="cursor-pointer"
                  >
                    {isVisible('instanceUid') && (
                      <TableCell className="font-mono text-xs">
                        <Link href={href} className="hover:underline">
                          {agent.metadata.instanceUid}
                        </Link>
                      </TableCell>
                    )}
                    {isVisible('connected') && (
                      <TableCell>
                        <Badge variant={agent.status.connected ? 'success' : 'muted'}>
                          {agent.status.connected ? 'Connected' : 'Disconnected'}
                        </Badge>
                      </TableCell>
                    )}
                    {isVisible('healthy') && (
                      <TableCell>
                        <Badge
                          variant={agent.status.componentHealth?.healthy ? 'success' : 'warning'}
                        >
                          {agent.status.componentHealth?.healthy ? 'Healthy' : 'Unhealthy'}
                        </Badge>
                      </TableCell>
                    )}
                    {isVisible('agentType') && (
                      <TableCell>
                        <Badge
                          variant={isOtelCollector(agent.metadata.type) ? 'primary' : 'outline'}
                        >
                          {agentTypeLabel(agent.metadata.type)}
                        </Badge>
                      </TableCell>
                    )}
                    {isVisible('type') && (
                      <TableCell>{agent.status.connectionType || '-'}</TableCell>
                    )}
                    {isVisible('lastReported') && (
                      <TableCell>
                        <TimeDisplay value={agent.status.lastReportedAt} />
                      </TableCell>
                    )}
                    {isVisible('sequence') && (
                      <TableCell className="tnum">{agent.status.sequenceNum ?? '-'}</TableCell>
                    )}
                    {isVisible('capabilities') && (
                      <TableCell>
                        {caps.length === 0 ? (
                          '-'
                        ) : (
                          <div className="flex max-w-80 flex-wrap gap-1">
                            {caps.map((c) => (
                              <Badge key={c} variant="outline">
                                {c}
                              </Badge>
                            ))}
                          </div>
                        )}
                      </TableCell>
                    )}
                    {isVisible('identifyingAttributes') && (
                      <TableCell>
                        <AttrBadges
                          attrs={agent.metadata.description?.identifyingAttributes}
                          onSelect={searchByAttribute}
                        />
                      </TableCell>
                    )}
                    {isVisible('nonIdentifyingAttributes') && (
                      <TableCell>
                        <AttrBadges
                          attrs={agent.metadata.description?.nonIdentifyingAttributes}
                          onSelect={searchByNonIdentifyingAttribute}
                        />
                      </TableCell>
                    )}
                    <TableCell className="text-right">
                      <RowActionsMenu
                        actions={[
                          { label: 'View detail', icon: <Eye aria-hidden />, href },
                          {
                            label: 'Edit spec',
                            icon: <Pencil aria-hidden />,
                            href: `${href}?action=edit`,
                          },
                          {
                            label: 'Request restart',
                            icon: <RotateCcw aria-hidden />,
                            href: `${href}?action=restart`,
                          },
                          // Deleting a connected agent is pointless (it reappears),
                          // so only surface it for disconnected ones.
                          ...(!agent.status.connected
                            ? [
                                {
                                  label: 'Delete agent',
                                  icon: <Trash2 aria-hidden />,
                                  onClick: () => setDeleting(agent),
                                  destructive: true,
                                  divider: true,
                                } satisfies RowAction,
                              ]
                            : []),
                        ]}
                      />
                    </TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </TableWrap>

      <PaginationFooter pagination={pagination} />

      <ConfirmDialog
        open={deleting !== null}
        title="Delete agent"
        message={deleting ? agentDeleteConfirmMessage(deleting.metadata.instanceUid) : ''}
        confirmLabel="Delete"
        destructive
        onClose={() => setDeleting(null)}
        onConfirm={onDelete}
      />
    </div>
  );
}

export default function AgentsPage() {
  return (
    <Suspense fallback={null}>
      <AgentsInner />
    </Suspense>
  );
}
