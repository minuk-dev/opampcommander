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
  Tab,
  Tabs,
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
import type { CodeSample } from '@shared/ui';
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
  return (
    <TextField
      multiline
      minRows={12}
      disabled
      value="Loading editor…"
      fullWidth
      slotProps={{
        input: { sx: { fontFamily: 'var(--font-geist-mono), monospace', fontSize: 13 } },
      }}
    />
  );
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
  const theme = useTheme();
  const fullScreen = useMediaQuery(theme.breakpoints.down('sm'));

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
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [tab, setTab] = useState<'edit' | 'diff'>('edit');
  const [busy, setBusy] = useState(false);
  const [saveError, setSaveError] = useState<{ message: string; hint?: string } | null>(null);
  const [sampleAnchor, setSampleAnchor] = useState<HTMLElement | null>(null);
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
      setShowAdvanced(false);
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
    setSampleAnchor(null);
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
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md" fullScreen={fullScreen}>
      <DialogTitle component="div">
        <Stack direction="row" alignItems="center" justifyContent="space-between" gap={1}>
          <Typography variant="h6" noWrap>
            {mode === 'create' ? 'Create remote config' : `Edit ${initial?.metadata.name ?? ''}`}
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
        <Stack spacing={2}>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
            <TextField
              label="Name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={mode === 'edit'}
              error={Boolean(nameError) && name !== ''}
              helperText={name === '' ? ' ' : (nameError ?? ' ')}
              fullWidth
              size="small"
            />
            <TextField
              select
              label="Content type"
              value={contentType}
              onChange={(e) => setContentType(e.target.value)}
              helperText=" "
              size="small"
              sx={{ minWidth: { xs: '100%', sm: 200 } }}
            >
              {CONTENT_TYPES.map((ct) => (
                <MenuItem key={ct} value={ct}>
                  {ct}
                </MenuItem>
              ))}
            </TextField>
          </Stack>

          {mode === 'edit' && (
            <Tabs value={tab} onChange={(_, v: 'edit' | 'diff') => setTab(v)}>
              <Tab value="edit" label="Config" />
              <Tab value="diff" label="Diff" />
            </Tabs>
          )}

          {tab === 'edit' ? (
            <Box>
              <CodeTextEditor
                value={body}
                onChange={setBody}
                language={languageFor(contentType)}
                height={fullScreen ? '55vh' : 380}
                errorLine={parseProblem?.line ?? null}
                ariaLabel="config body"
                placeholder="receivers:&#10;  otlp:&#10;    protocols:&#10;      grpc: {}"
              />
              {parseProblem && (
                <Alert severity="error" sx={{ mt: 1 }}>
                  <AlertTitle>
                    {languageFor(contentType) === 'json' ? 'Invalid JSON' : 'Invalid YAML'}
                    {parseProblem.line ? ` (line ${parseProblem.line})` : ''}
                  </AlertTitle>
                  <Box component="pre" sx={{ m: 0, whiteSpace: 'pre-wrap', fontSize: 12 }}>
                    {parseProblem.message}
                  </Box>
                </Alert>
              )}
            </Box>
          ) : (
            <DiffView
              oldText={initial?.spec.value ?? ''}
              newText={body}
              oldLabel="stored"
              newLabel="editing"
              maxHeight={fullScreen ? '55vh' : 380}
            />
          )}

          <Box>
            <Button
              size="small"
              onClick={() => setShowAdvanced((v) => !v)}
              endIcon={showAdvanced ? <ExpandLessIcon /> : <ExpandMoreIcon />}
            >
              Attributes
            </Button>
            {showAdvanced && (
              <TextField
                label="Attributes (YAML map)"
                value={attributesText}
                onChange={(e) => setAttributesText(e.target.value)}
                placeholder="team: platform"
                multiline
                minRows={3}
                fullWidth
                size="small"
                sx={{ mt: 1 }}
                slotProps={{
                  input: { sx: { fontFamily: 'var(--font-geist-mono), monospace', fontSize: 13 } },
                }}
              />
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
        <Button variant="contained" onClick={save} disabled={!canSave}>
          {mode === 'create' ? 'Create' : 'Save changes'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
