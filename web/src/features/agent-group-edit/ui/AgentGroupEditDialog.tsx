'use client';

import { useEffect, useRef, useState } from 'react';
import { useNamespace } from '@entities/namespace';
import { api } from '@shared/api';
import { loadAgentGroupSamples, type AgentGroupSample } from '../model/samples';
import { fromYAML, toYAML } from '@shared/lib';
import {
  Alert,
  Button,
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  Input,
  SampleMenu,
  SegmentedControl,
  SegmentedItem,
  Textarea,
} from '@shared/ui';
import type { AgentGroup, AgentGroupSpec } from '@entities/agent-group';

type Format = 'yaml' | 'json';

interface Props {
  open: boolean;
  mode: 'create' | 'edit';
  initial?: AgentGroup;
  onClose: () => void;
  onSaved: () => void;
}

function defaultSpec(): AgentGroupSpec {
  return {
    priority: 0,
    selector: {
      identifyingAttributes: {},
      nonIdentifyingAttributes: {},
    },
  };
}

function serialize(value: unknown, format: Format): string {
  if (format === 'yaml') return toYAML(value);
  return JSON.stringify(value ?? {}, null, 2);
}
function parse(text: string, format: Format): unknown {
  const t = text.trim();
  if (!t) return {};
  if (format === 'yaml') return fromYAML(text);
  return JSON.parse(text);
}

export default function AgentGroupEditDialog({ open, mode, initial, onClose, onSaved }: Props) {
  const { namespace } = useNamespace();
  const [format, setFormat] = useState<Format>('yaml');
  const [name, setName] = useState('');
  const [specText, setSpecText] = useState('');
  const [attributesText, setAttributesText] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [samples, setSamples] = useState<AgentGroupSample[] | null>(null);
  const [samplesError, setSamplesError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setSamples(null);
      setSamplesError(null);
      return;
    }
    let cancelled = false;
    loadAgentGroupSamples()
      .then((list) => {
        if (!cancelled) setSamples(list);
      })
      .catch((err) => {
        if (!cancelled) {
          setSamplesError(err instanceof Error ? err.message : String(err));
          setSamples([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open]);

  const applySample = (s: AgentGroupSample) => {
    if (mode === 'create') {
      setName(s.name);
    }
    setAttributesText(serialize(s.attributes, format));
    setSpecText(serialize(s.spec, format));
    setError(null);
  };

  // Reset on the closed→open transition only. Parents may pass a freshly
  // fetched `initial` reference for the same logical row mid-edit (e.g. list
  // refresh), which must not stomp the user's in-progress buffers.
  const wasOpen = useRef(false);
  const initialRef = useRef(initial);
  initialRef.current = initial;
  useEffect(() => {
    if (open && !wasOpen.current) {
      const i = initialRef.current;
      setError(null);
      setFormat('yaml');
      setName(i?.metadata.name ?? '');
      setSpecText(serialize(i?.spec ?? defaultSpec(), 'yaml'));
      setAttributesText(serialize(i?.metadata.attributes ?? {}, 'yaml'));
    }
    wasOpen.current = open;
  }, [open]);

  const switchFormat = (next: Format) => {
    if (next === format) return;
    try {
      const spec = parse(specText, format);
      const attrs = parse(attributesText, format);
      setSpecText(serialize(spec, next));
      setAttributesText(serialize(attrs, next));
      setFormat(next);
      setError(null);
    } catch (err) {
      setError(
        `Cannot switch to ${next.toUpperCase()} — current ${format.toUpperCase()} buffer is invalid: ${
          err instanceof Error ? err.message : String(err)
        }`,
      );
    }
  };

  const save = async () => {
    setBusy(true);
    setError(null);
    try {
      const parsedSpec = parse(specText, format);
      const parsedAttrs = parse(attributesText, format);
      if (!parsedSpec || typeof parsedSpec !== 'object' || Array.isArray(parsedSpec)) {
        throw new Error('spec must be an object');
      }
      if (!parsedAttrs || typeof parsedAttrs !== 'object' || Array.isArray(parsedAttrs)) {
        throw new Error('attributes must be an object');
      }
      const spec = parsedSpec as AgentGroupSpec;
      const attributes = parsedAttrs as Record<string, string>;
      if (mode === 'create') {
        const body: Partial<AgentGroup> = {
          metadata: {
            namespace,
            name,
            attributes,
            createdAt: new Date().toISOString(),
          },
          spec,
        };
        await api.post(`/api/v1/namespaces/${namespace}/agentgroups`, body);
      } else if (initial) {
        const body: AgentGroup = {
          ...initial,
          metadata: { ...initial.metadata, attributes },
          spec,
        };
        await api.put(`/api/v1/namespaces/${namespace}/agentgroups/${initial.metadata.name}`, body);
      }
      onSaved();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle className="flex-1">
            {mode === 'create' ? 'Create agent group' : 'Edit agent group'}
          </DialogTitle>
          <SampleMenu samples={samples} onPick={applySample} />
          <SegmentedControl
            type="single"
            value={format}
            onValueChange={(v: string) => v && switchFormat(v as Format)}
            aria-label="format"
          >
            <SegmentedItem value="yaml">YAML</SegmentedItem>
            <SegmentedItem value="json">JSON</SegmentedItem>
          </SegmentedControl>
        </DialogHeader>
        <DialogBody className="space-y-3">
          {samplesError && <Alert severity="warning">Failed to load samples: {samplesError}</Alert>}
          {error && <Alert severity="error">{error}</Alert>}
          <Field label="Name" required>
            {(field) => (
              <Input
                {...field}
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={mode === 'edit'}
              />
            )}
          </Field>
          <Field label={`Attributes (${format.toUpperCase()}, key/value pairs)`}>
            {(field) => (
              <Textarea
                {...field}
                mono
                rows={3}
                value={attributesText}
                onChange={(e) => setAttributesText(e.target.value)}
              />
            )}
          </Field>
          <Field label={`Spec (${format.toUpperCase()})`} hint="priority, selector, agentConfig.">
            {(field) => (
              <Textarea
                {...field}
                mono
                rows={14}
                value={specText}
                onChange={(e) => setSpecText(e.target.value)}
              />
            )}
          </Field>
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button onClick={() => void save()} disabled={busy || (mode === 'create' && !name)}>
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
