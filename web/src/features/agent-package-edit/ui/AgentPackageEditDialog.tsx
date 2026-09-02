'use client';

import { useEffect, useRef, useState } from 'react';
import { api, describeApiError } from '@shared/api';
import { loadSamples, parseAttributes, toYAML, validateResourceName } from '@shared/lib';
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
  Textarea,
} from '@shared/ui';
import type { AgentPackage } from '@entities/agent-package';
import { PACKAGE_TYPES, validateBase64, validateDownloadUrl } from '../lib/validate';

interface Props {
  open: boolean;
  mode: 'create' | 'edit';
  namespace: string;
  initial?: AgentPackage;
  onClose: () => void;
  onSaved: () => void;
}

export default function AgentPackageEditDialog({
  open,
  mode,
  namespace,
  initial,
  onClose,
  onSaved,
}: Props) {
  const [name, setName] = useState('');
  const [packageType, setPackageType] = useState<string>(PACKAGE_TYPES[0]);
  const [version, setVersion] = useState('');
  const [downloadUrl, setDownloadUrl] = useState('');
  const [contentHash, setContentHash] = useState('');
  const [signature, setSignature] = useState('');
  const [hash, setHash] = useState('');
  const [headersText, setHeadersText] = useState('');
  const [attributesText, setAttributesText] = useState('');
  const [busy, setBusy] = useState(false);
  const [saveError, setSaveError] = useState<{ message: string; hint?: string } | null>(null);
  const [samples, setSamples] = useState<CodeSample[] | null>(null);

  // Reset only on the closed→open transition — the list behind the dialog
  // refreshes and would otherwise replace `initial` mid-edit.
  const initialRef = useRef(initial);
  initialRef.current = initial;
  const wasOpen = useRef(false);
  useEffect(() => {
    if (open && !wasOpen.current) {
      const i = initialRef.current;
      setName(i?.metadata.name ?? '');
      setPackageType(i?.spec.packageType || PACKAGE_TYPES[0]);
      setVersion(i?.spec.version ?? '');
      setDownloadUrl(i?.spec.downloadUrl ?? '');
      setContentHash(i?.spec.contentHash ?? '');
      setSignature(i?.spec.signature ?? '');
      setHash(i?.spec.hash ?? '');
      setHeadersText(i?.spec.headers ? toYAML(i.spec.headers) : '');
      setAttributesText(
        i?.metadata.attributes && Object.keys(i.metadata.attributes).length > 0
          ? toYAML(i.metadata.attributes)
          : '',
      );
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
    loadSamples('/samples/agentpackages.yaml', { namespace })
      .then((list) => !cancelled && setSamples(list))
      .catch(() => !cancelled && setSamples([]));
    return () => {
      cancelled = true;
    };
  }, [open, namespace]);

  const applySample = (sample: CodeSample) => {
    const value = sample.value as Partial<AgentPackage> | undefined;
    if (mode === 'create' && value?.metadata?.name) setName(value.metadata.name);
    setPackageType(value?.spec?.packageType || PACKAGE_TYPES[0]);
    setVersion(value?.spec?.version ?? '');
    setDownloadUrl(value?.spec?.downloadUrl ?? '');
    setContentHash(value?.spec?.contentHash ?? '');
    setSignature(value?.spec?.signature ?? '');
    setHeadersText(value?.spec?.headers ? toYAML(value.spec.headers) : '');
  };

  const nameError = mode === 'create' ? validateResourceName(name) : null;
  const urlError = validateDownloadUrl(downloadUrl);
  const contentHashError = validateBase64(contentHash, 'Content hash');
  const signatureError = validateBase64(signature, 'Signature');
  const hashError = validateBase64(hash, 'Hash');
  const blocked =
    Boolean(nameError) ||
    Boolean(urlError) ||
    Boolean(contentHashError) ||
    Boolean(signatureError) ||
    Boolean(hashError);

  const save = async () => {
    setBusy(true);
    setSaveError(null);
    try {
      const attributes = parseAttributes(attributesText);
      const headers = parseAttributes(headersText);
      const spec = {
        packageType,
        version,
        downloadUrl,
        ...(contentHash ? { contentHash } : {}),
        ...(signature ? { signature } : {}),
        ...(hash ? { hash } : {}),
        ...(Object.keys(headers).length > 0 ? { headers } : {}),
      };
      if (mode === 'create') {
        const created: AgentPackage = {
          metadata: { name, namespace, attributes, createdAt: new Date().toISOString() },
          spec,
        };
        await api.post(`/api/v1/namespaces/${namespace}/agentpackages`, created);
      } else if (initial) {
        // Spread the loaded resource so status and any field this dialog does
        // not edit round-trip untouched.
        const updated: AgentPackage = {
          ...initial,
          metadata: { ...initial.metadata, attributes },
          spec,
        };
        await api.put(
          `/api/v1/namespaces/${namespace}/agentpackages/${initial.metadata.name}`,
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

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle className="flex-1">
            {mode === 'create' ? 'Create agent package' : `Edit ${initial?.metadata.name ?? ''}`}
          </DialogTitle>
          <SampleMenu samples={samples} onPick={applySample} />
        </DialogHeader>
        <DialogBody className="space-y-3">
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

          <div className="grid gap-3 sm:grid-cols-[10rem_1fr]">
            <Field label="Package type">
              {(field) => (
                <Select value={packageType} onValueChange={setPackageType}>
                  <SelectTrigger {...field}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PACKAGE_TYPES.map((t) => (
                      <SelectItem key={t} value={t}>
                        {t}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </Field>
            <Field label="Version">
              {(field) => (
                <Input
                  {...field}
                  value={version}
                  onChange={(e) => setVersion(e.target.value)}
                  placeholder="0.110.0"
                />
              )}
            </Field>
          </div>

          <Field label="Download URL" error={downloadUrl === '' ? undefined : urlError} required>
            {(field) => (
              <Input
                {...field}
                className="font-mono text-[13px]"
                value={downloadUrl}
                onChange={(e) => setDownloadUrl(e.target.value)}
                invalid={Boolean(urlError) && downloadUrl !== ''}
                placeholder="https://example.com/otelcol.tar.gz"
              />
            )}
          </Field>

          <Field
            label="Content hash (base64)"
            hint="SHA-256 of the artifact, base64-encoded."
            error={contentHashError}
          >
            {(field) => (
              <Input
                {...field}
                className="font-mono text-[13px]"
                value={contentHash}
                onChange={(e) => setContentHash(e.target.value)}
                invalid={Boolean(contentHashError)}
              />
            )}
          </Field>

          <Collapsible label="Advanced">
            <div className="space-y-3">
              <Field label="Signature (base64)" error={signatureError}>
                {(field) => (
                  <Input
                    {...field}
                    className="font-mono text-[13px]"
                    value={signature}
                    onChange={(e) => setSignature(e.target.value)}
                    invalid={Boolean(signatureError)}
                  />
                )}
              </Field>
              <Field
                label="Package hash (base64)"
                hint="Hash advertised to agents for the whole package."
                error={hashError}
              >
                {(field) => (
                  <Input
                    {...field}
                    className="font-mono text-[13px]"
                    value={hash}
                    onChange={(e) => setHash(e.target.value)}
                    invalid={Boolean(hashError)}
                  />
                )}
              </Field>
              <Field label="Download headers (YAML map)">
                {(field) => (
                  <Textarea
                    {...field}
                    mono
                    rows={2}
                    value={headersText}
                    onChange={(e) => setHeadersText(e.target.value)}
                    placeholder="Authorization: Bearer TOKEN"
                  />
                )}
              </Field>
              <Field label="Attributes (YAML map)">
                {(field) => (
                  <Textarea
                    {...field}
                    mono
                    rows={2}
                    value={attributesText}
                    onChange={(e) => setAttributesText(e.target.value)}
                    placeholder="team: platform"
                  />
                )}
              </Field>
            </div>
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
          <Button onClick={() => void save()} disabled={busy || blocked}>
            {mode === 'create' ? 'Create' : 'Save changes'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
