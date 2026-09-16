'use client';

import { type ReactNode, useEffect, useState } from 'react';
import { fromYAML, loadSamples, type SamplesPath, toYAML } from '@shared/lib';
import Alert from './Alert';
import Button from './Button';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './Dialog';
import SampleMenu from './SampleMenu';
import { SegmentedControl, SegmentedItem } from './SegmentedControl';
import Textarea from './Textarea';

export type CodeFormat = 'yaml' | 'json';

export interface CodeSample {
  label: string;
  description?: string;
  value: unknown;
}

interface Props {
  open: boolean;
  title: string;
  description?: ReactNode;
  initialValue: unknown;
  defaultFormat?: CodeFormat;
  // Inline list of samples (synchronous). Prefer samplesUrl in new code.
  samples?: CodeSample[];
  // Path under /samples/*.yaml (see web/public/samples/). When set, the menu
  // is populated on dialog open from this file. {{namespace}}, {{now}}, etc.
  // in the YAML are substituted from samplesVars (plus a built-in `now`).
  samplesUrl?: SamplesPath;
  samplesVars?: Record<string, string>;
  onClose: () => void;
  onSave: (parsed: unknown) => Promise<void> | void;
}

function serialize(value: unknown, format: CodeFormat): string {
  if (format === 'yaml') return toYAML(value);
  return JSON.stringify(value ?? {}, null, 2);
}

function parse(text: string, format: CodeFormat): unknown {
  const trimmed = text.trim();
  if (trimmed === '') return {};
  if (format === 'yaml') return fromYAML(text);
  return JSON.parse(text);
}

// CodeEditorDialog is the canonical editor for whole structured payloads. YAML
// is the default surface; users can flip to JSON, and the current buffer is
// re-serialized in the new format so they don't lose work.
export default function CodeEditorDialog({
  open,
  title,
  description,
  initialValue,
  defaultFormat = 'yaml',
  samples,
  samplesUrl,
  samplesVars,
  onClose,
  onSave,
}: Props) {
  const [format, setFormat] = useState<CodeFormat>(defaultFormat);
  const [text, setText] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loadedSamples, setLoadedSamples] = useState<CodeSample[] | null>(null);
  const [samplesError, setSamplesError] = useState<string | null>(null);

  // Reset the buffer only on the closed→open transition, reading the props as
  // they are at that moment. Parents commonly pass a freshly-constructed
  // initialValue (e.g. emptyFoo()) each render, so keying off its identity
  // would wipe in-progress edits on every parent re-render.
  // Starts false so a dialog mounted already-open still counts as a transition.
  const [wasOpen, setWasOpen] = useState(false);
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) {
      setFormat(defaultFormat);
      setText(serialize(initialValue, defaultFormat));
      setError(null);
    }
  }

  // Stable JSON key so we don't refetch every render when the parent creates a
  // fresh samplesVars object each time; the effect below parses it back rather
  // than depending on that object's identity.
  const varsKey = samplesVars ? JSON.stringify(samplesVars) : '';

  // Drop samples loaded for a previous open (or a previous file) so the menu
  // shows "Loading…" rather than the entries that no longer apply.
  const samplesKey = open && samplesUrl ? `${samplesUrl}|${varsKey}` : '';
  const [prevSamplesKey, setPrevSamplesKey] = useState(samplesKey);
  if (samplesKey !== prevSamplesKey) {
    setPrevSamplesKey(samplesKey);
    setLoadedSamples(null);
    setSamplesError(null);
  }

  useEffect(() => {
    if (!open || !samplesUrl) return;
    const vars = varsKey ? (JSON.parse(varsKey) as Record<string, string>) : {};
    let cancelled = false;
    loadSamples(samplesUrl, vars)
      .then((list) => {
        if (!cancelled) setLoadedSamples(list);
      })
      .catch((err) => {
        if (!cancelled) {
          setSamplesError(err instanceof Error ? err.message : String(err));
          setLoadedSamples([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [open, samplesUrl, varsKey]);

  const switchFormat = (next: CodeFormat) => {
    if (next === format) return;
    // Re-serialize the current buffer so unsaved edits survive the toggle.
    try {
      const parsed = parse(text, format);
      setText(serialize(parsed, next));
      setError(null);
      setFormat(next);
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
      const parsed = parse(text, format);
      await onSave(parsed);
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
          <DialogTitle className="flex-1">{title}</DialogTitle>
          {(samples || samplesUrl) && (
            <SampleMenu
              samples={samples ?? loadedSamples}
              onPick={(sample) => {
                setText(serialize(sample.value, format));
                setError(null);
              }}
            />
          )}
          <SegmentedControl
            type="single"
            value={format}
            onValueChange={(v: string) => v && switchFormat(v as CodeFormat)}
            aria-label="format"
          >
            <SegmentedItem value="yaml">YAML</SegmentedItem>
            <SegmentedItem value="json">JSON</SegmentedItem>
          </SegmentedControl>
        </DialogHeader>
        <DialogBody className="space-y-3">
          {description && <p className="text-xs text-muted-foreground">{description}</p>}
          {samplesError && <Alert severity="warning">Failed to load samples: {samplesError}</Alert>}
          {error && <Alert severity="error">{error}</Alert>}
          <Textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            aria-label="editor"
            rows={18}
            mono
            className="min-h-[45vh]"
          />
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button onClick={() => void save()} disabled={busy}>
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
