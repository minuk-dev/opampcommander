'use client';

import dynamic from 'next/dynamic';
import { useEffect, useMemo, useRef, useState } from 'react';
import { api, describeApiError } from '@shared/api';
import {
  languageFor,
  loadSamples,
  parseAttributes,
  toYAML,
  validateResourceName,
} from '@shared/lib';
import {
  Alert,
  Button,
  type CodeSample,
  Collapsible,
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  Input,
  SampleMenu,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Textarea,
} from '@shared/ui';
import type { AgentRemoteConfig } from '@entities/agent-remote-config';
import { validateConfigBody } from '../lib/validate';

// The highlighted editor and the diff renderer are only needed once this dialog
// is open, and the dialog itself is already lazily imported by the page — this
// keeps them in their own chunks and degrades to a plain textarea meanwhile.
const CodeTextEditor = dynamic(() => import('@shared/ui/CodeTextEditor'), {
  loading: () => <PlainFallback />,
  ssr: false,
});
const DiffView = dynamic(() => import('@shared/ui/DiffView'), { ssr: false });

function PlainFallback() {
  return <Textarea mono disabled rows={12} value="Loading editor…" aria-label="loading editor" />;
}

const CONTENT_TYPES = ['text/yaml', 'application/json', 'text/plain'];

interface Props {
  open: boolean;
  mode: 'create' | 'edit';
  namespace: string;
  initial?: AgentRemoteConfig;
  onClose: () => void;
  onSaved: () => void;
}

export default function AgentRemoteConfigEditDialog({
  open,
  mode,
  namespace,
  initial,
  onClose,
  onSaved,
}: Props) {
  const [name, setName] = useState('');
  const [contentType, setContentType] = useState(CONTENT_TYPES[0]);
  const [body, setBody] = useState('');
  const [attributesText, setAttributesText] = useState('');
  // Snapshot of the buffers as loaded, so "Save changes" can tell whether
  // anything actually changed.
  const [loaded, setLoaded] = useState({
    contentType: CONTENT_TYPES[0],
    body: '',
    attributesText: '',
  });
  const [tab, setTab] = useState<'edit' | 'diff'>('edit');
  const [busy, setBusy] = useState(false);
  const [saveError, setSaveError] = useState<{ message: string; hint?: string } | null>(null);
  const [samples, setSamples] = useState<CodeSample[] | null>(null);

  // Reset only on the closed→open transition: the list refreshes behind the
  // dialog and would otherwise hand us a new `initial` reference mid-edit.
  const initialRef = useRef(initial);
  initialRef.current = initial;
  const wasOpen = useRef(false);
  useEffect(() => {
    if (open && !wasOpen.current) {
      const i = initialRef.current;
      const buffers = {
        contentType: i?.spec.contentType || CONTENT_TYPES[0],
        body: i?.spec.value ?? '',
        attributesText:
          i?.metadata.attributes && Object.keys(i.metadata.attributes).length > 0
            ? toYAML(i.metadata.attributes)
            : '',
      };
      setName(i?.metadata.name ?? '');
      setContentType(buffers.contentType);
      setBody(buffers.body);
      setAttributesText(buffers.attributesText);
      setLoaded(buffers);
      setTab('edit');
      setSaveError(null);
    }
    wasOpen.current = open;
  }, [open]);

  useEffect(() => {
    if (!open) {
      setSamples(null);
      return;
    }
    let cancelled = false;
    loadSamples('/samples/agentremoteconfigs.yaml', { namespace })
      .then((list) => !cancelled && setSamples(list))
      .catch(() => !cancelled && setSamples([]));
    return () => {
      cancelled = true;
    };
  }, [open, namespace]);

  const applySample = (sample: CodeSample) => {
    const value = sample.value as Partial<AgentRemoteConfig> | undefined;
    if (mode === 'create' && value?.metadata?.name) setName(value.metadata.name);
    if (value?.spec?.contentType) setContentType(value.spec.contentType);
    setBody(value?.spec?.value ?? '');
  };

  const nameError = mode === 'create' ? validateResourceName(name) : null;
  const parseProblem = useMemo(() => validateConfigBody(body, contentType), [body, contentType]);
  const dirty =
    body !== loaded.body ||
    contentType !== loaded.contentType ||
    attributesText !== loaded.attributesText;

  const save = async () => {
    setBusy(true);
    setSaveError(null);
    try {
      const attributes = parseAttributes(attributesText);
      if (mode === 'create') {
        const created: AgentRemoteConfig = {
          metadata: { name, namespace, attributes, createdAt: new Date().toISOString() },
          spec: { value: body, contentType },
        };
        await api.post(`/api/v1/namespaces/${namespace}/agentremoteconfigs`, created);
      } else if (initial) {
        // Spread the loaded resource so fields this dialog does not edit
        // (schemaRefs, status, kind/apiVersion) round-trip untouched.
        const updated: AgentRemoteConfig = {
          ...initial,
          metadata: { ...initial.metadata, attributes },
          spec: { ...initial.spec, value: body, contentType },
        };
        await api.put(
          `/api/v1/namespaces/${namespace}/agentremoteconfigs/${initial.metadata.name}`,
          updated,
        );
      }
      onSaved();
    } catch (err) {
      setSaveError(describeApiError(err));
    } finally {
      setBusy(false);
    }
  };

  // Nothing to send when an edit left every buffer untouched.
  const canSave = !busy && !nameError && !parseProblem && (mode === 'create' || dirty);

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle className="flex-1">
            {mode === 'create' ? 'Create remote config' : `Edit ${initial?.metadata.name ?? ''}`}
          </DialogTitle>
          <SampleMenu samples={samples} onPick={applySample} />
        </DialogHeader>
        <DialogBody className="space-y-3">
          <div className="grid gap-3 sm:grid-cols-[1fr_12rem]">
            <Field label="Name" error={name === '' ? undefined : nameError} required>
              {(field) => (
                <Input
                  {...field}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  disabled={mode === 'edit'}
                  invalid={Boolean(nameError) && name !== ''}
                />
              )}
            </Field>
            <Field label="Content type">
              {(field) => (
                <Select value={contentType} onValueChange={setContentType}>
                  <SelectTrigger {...field}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {CONTENT_TYPES.map((ct) => (
                      <SelectItem key={ct} value={ct}>
                        {ct}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </Field>
          </div>

          <Tabs value={tab} onValueChange={(v) => setTab(v as 'edit' | 'diff')}>
            {mode === 'edit' && (
              <TabsList className="mb-2">
                <TabsTrigger value="edit">Config</TabsTrigger>
                <TabsTrigger value="diff">Diff</TabsTrigger>
              </TabsList>
            )}
            <TabsContent value="edit">
              <CodeTextEditor
                value={body}
                onChange={setBody}
                language={languageFor(contentType)}
                height="min(45vh, 24rem)"
                errorLine={parseProblem?.line ?? null}
                ariaLabel="config body"
                placeholder={'receivers:\n  otlp:\n    protocols:\n      grpc: {}'}
              />
              {parseProblem && (
                <Alert
                  severity="error"
                  className="mt-2"
                  title={`${languageFor(contentType) === 'json' ? 'Invalid JSON' : 'Invalid YAML'}${
                    parseProblem.line ? ` (line ${parseProblem.line})` : ''
                  }`}
                >
                  <pre className="m-0 overflow-x-auto text-xs whitespace-pre-wrap">
                    {parseProblem.message}
                  </pre>
                </Alert>
              )}
            </TabsContent>
            <TabsContent value="diff">
              <DiffView
                oldText={initial?.spec.value ?? ''}
                newText={body}
                oldLabel="stored"
                newLabel="editing"
                maxHeight="min(45vh, 24rem)"
              />
            </TabsContent>
          </Tabs>

          <Collapsible label="Attributes">
            <Field label="Attributes (YAML map)">
              {(field) => (
                <Textarea
                  {...field}
                  mono
                  rows={3}
                  value={attributesText}
                  onChange={(e) => setAttributesText(e.target.value)}
                  placeholder="team: platform"
                />
              )}
            </Field>
          </Collapsible>

          {saveError && (
            <Alert severity="error" title={saveError.hint}>
              {saveError.message}
            </Alert>
          )}
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button onClick={() => void save()} disabled={!canSave}>
            {mode === 'create' ? 'Create' : 'Save changes'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
