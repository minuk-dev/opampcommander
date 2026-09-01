'use client';

import {
  Alert,
  AlertTitle,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  IconButton,
  ListItemText,
  Menu,
  MenuItem,
  Stack,
  TextField,
  Typography,
  useMediaQuery,
  useTheme,
} from '@mui/material';
import {
  ArrowDropDown as ArrowDropDownIcon,
  Close as CloseIcon,
  ExpandLess as ExpandLessIcon,
  ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { useEffect, useRef, useState } from 'react';
import { api, describeApiError } from '@shared/api';
import { loadSamples, parseAttributes, toYAML, validateResourceName } from '@shared/lib';
import type { CodeSample } from '@shared/ui';
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

const monoInput = {
  input: { sx: { fontFamily: 'var(--font-geist-mono), monospace', fontSize: 13 } },
};

export default function AgentPackageEditDialog({
  open,
  mode,
  namespace,
  initial,
  onClose,
  onSaved,
}: Props) {
  const theme = useTheme();
  const fullScreen = useMediaQuery(theme.breakpoints.down('sm'));

  const [name, setName] = useState('');
  const [packageType, setPackageType] = useState<string>(PACKAGE_TYPES[0]);
  const [version, setVersion] = useState('');
  const [downloadUrl, setDownloadUrl] = useState('');
  const [contentHash, setContentHash] = useState('');
  const [signature, setSignature] = useState('');
  const [hash, setHash] = useState('');
  const [headersText, setHeadersText] = useState('');
  const [attributesText, setAttributesText] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [busy, setBusy] = useState(false);
  const [saveError, setSaveError] = useState<{ message: string; hint?: string } | null>(null);
  const [sampleAnchor, setSampleAnchor] = useState<HTMLElement | null>(null);
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
      setShowAdvanced(false);
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
    setSampleAnchor(null);
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
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm" fullScreen={fullScreen}>
      <DialogTitle component="div">
        <Stack direction="row" alignItems="center" justifyContent="space-between" gap={1}>
          <Typography variant="h6" noWrap>
            {mode === 'create' ? 'Create agent package' : `Edit ${initial?.metadata.name ?? ''}`}
          </Typography>
          <Stack direction="row" alignItems="center" gap={1}>
            <Button
              size="small"
              variant="outlined"
              endIcon={<ArrowDropDownIcon />}
              onClick={(e) => setSampleAnchor(e.currentTarget)}
              disabled={samples === null}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {samples === null ? 'Loading…' : 'Load sample'}
            </Button>
            <Menu
              anchorEl={sampleAnchor}
              open={Boolean(sampleAnchor)}
              onClose={() => setSampleAnchor(null)}
            >
              {(samples ?? []).length === 0 && <MenuItem disabled>No samples available</MenuItem>}
              {(samples ?? []).map((s, i) => (
                <MenuItem key={`${i}-${s.label}`} onClick={() => applySample(s)}>
                  <ListItemText primary={s.label} secondary={s.description} />
                </MenuItem>
              ))}
            </Menu>
            {fullScreen && (
              <IconButton onClick={onClose} aria-label="close">
                <CloseIcon />
              </IconButton>
            )}
          </Stack>
        </Stack>
      </DialogTitle>
      <DialogContent dividers>
        <Stack spacing={2} sx={{ mt: 1 }}>
          <TextField
            label="Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={mode === 'edit'}
            error={Boolean(nameError) && name !== ''}
            helperText={name === '' ? undefined : (nameError ?? undefined)}
            fullWidth
            size="small"
          />
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
            <TextField
              select
              label="Package type"
              value={packageType}
              onChange={(e) => setPackageType(e.target.value)}
              size="small"
              sx={{ minWidth: { xs: '100%', sm: 160 } }}
            >
              {PACKAGE_TYPES.map((t) => (
                <MenuItem key={t} value={t}>
                  {t}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="Version"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder="0.110.0"
              fullWidth
              size="small"
            />
          </Stack>
          <TextField
            label="Download URL"
            value={downloadUrl}
            onChange={(e) => setDownloadUrl(e.target.value)}
            error={Boolean(urlError) && downloadUrl !== ''}
            helperText={downloadUrl === '' ? undefined : (urlError ?? undefined)}
            placeholder="https://example.com/otelcol.tar.gz"
            fullWidth
            size="small"
            slotProps={monoInput}
          />
          <TextField
            label="Content hash (base64)"
            value={contentHash}
            onChange={(e) => setContentHash(e.target.value)}
            error={Boolean(contentHashError)}
            helperText={contentHashError ?? 'SHA-256 of the artifact, base64-encoded.'}
            fullWidth
            size="small"
            slotProps={monoInput}
          />

          <Box>
            <Button
              size="small"
              onClick={() => setShowAdvanced((v) => !v)}
              endIcon={showAdvanced ? <ExpandLessIcon /> : <ExpandMoreIcon />}
            >
              Advanced
            </Button>
            {showAdvanced && (
              <Stack spacing={2} sx={{ mt: 1 }}>
                <TextField
                  label="Signature (base64)"
                  value={signature}
                  onChange={(e) => setSignature(e.target.value)}
                  error={Boolean(signatureError)}
                  helperText={signatureError ?? undefined}
                  fullWidth
                  size="small"
                  slotProps={monoInput}
                />
                <TextField
                  label="Package hash (base64)"
                  value={hash}
                  onChange={(e) => setHash(e.target.value)}
                  error={Boolean(hashError)}
                  helperText={hashError ?? 'Hash advertised to agents for the whole package.'}
                  fullWidth
                  size="small"
                  slotProps={monoInput}
                />
                <TextField
                  label="Download headers (YAML map)"
                  value={headersText}
                  onChange={(e) => setHeadersText(e.target.value)}
                  placeholder="Authorization: Bearer TOKEN"
                  multiline
                  minRows={2}
                  fullWidth
                  size="small"
                  slotProps={monoInput}
                />
                <TextField
                  label="Attributes (YAML map)"
                  value={attributesText}
                  onChange={(e) => setAttributesText(e.target.value)}
                  placeholder="team: platform"
                  multiline
                  minRows={2}
                  fullWidth
                  size="small"
                  slotProps={monoInput}
                />
              </Stack>
            )}
          </Box>

          {saveError && (
            <Alert severity="error">
              {saveError.hint && <AlertTitle>{saveError.hint}</AlertTitle>}
              {saveError.message}
            </Alert>
          )}
        </Stack>
      </DialogContent>
      <Divider />
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          Cancel
        </Button>
        <Button variant="contained" onClick={save} disabled={busy || blocked}>
          {mode === 'create' ? 'Create' : 'Save changes'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
