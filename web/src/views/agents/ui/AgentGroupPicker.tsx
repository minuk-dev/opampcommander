'use client';

import { useEffect, useState } from 'react';
import type { AgentGroup } from '@entities/agent-group';
import { api, type ListResponse } from '@shared/api';
import { Combobox } from '@shared/ui';

interface Props {
  namespace: string;
  value: string;
  onChange: (value: string) => void;
  id?: string;
  'aria-describedby'?: string;
}

export default function AgentGroupPicker({ namespace, ...props }: Props) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [cursor, setCursor] = useState('');
  const [attempt, setAttempt] = useState(0);
  const [result, setResult] = useState<{
    key: string;
    names: string[];
    next?: string;
    error?: string;
  }>();
  const key = JSON.stringify([namespace, search, cursor, attempt]);
  const current = result?.key === key ? result : undefined;

  useEffect(() => {
    if (!open) return;
    const controller = new AbortController();
    const timer = setTimeout(() => {
      void api
        .get<ListResponse<AgentGroup>>(
          `/api/v1/namespaces/${encodeURIComponent(namespace)}/agentgroups`,
          { query: { name: search, limit: 50, continue: cursor }, signal: controller.signal },
        )
        .then((data) => {
          if (controller.signal.aborted) return;
          const next = data.metadata.remainingItemCount > 0 ? data.metadata.continue : undefined;
          if (data.metadata.remainingItemCount > 0 && (!next || next === cursor)) {
            throw new Error('Agent group cursor did not advance');
          }
          setResult({ key, names: data.items.map((group) => group.metadata.name), next });
        })
        .catch((err: unknown) => {
          if (controller.signal.aborted) return;
          setResult({
            key,
            names: [],
            error: err instanceof Error ? err.message : 'Failed to load agent groups',
          });
        });
    }, 250);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [open, namespace, search, cursor, key]);

  return (
    <Combobox
      {...props}
      placeholder="Select an agent group…"
      searchPlaceholder="Agent group name starts with…"
      emptyMessage="No agent groups found"
      options={(current?.names ?? []).map((name) => ({ value: name, label: name }))}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) {
          setSearch('');
          setCursor('');
        }
      }}
      onSearchChange={(value) => {
        setSearch(value);
        setCursor('');
      }}
      loading={open && !current}
      error={current?.error}
      onRetry={() => setAttempt((value) => value + 1)}
      onNextPage={current?.next ? () => setCursor(current.next!) : undefined}
    />
  );
}
