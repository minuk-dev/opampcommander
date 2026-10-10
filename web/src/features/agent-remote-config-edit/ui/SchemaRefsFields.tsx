'use client';

import { listRemoteConfigSchemas } from '@entities/remoteconfigschema';
import { useSWR } from '@shared/api';
import { Alert, Checkbox, Switch } from '@shared/ui';

interface Props {
  open: boolean;
  namespace: string;
  refs: string[];
  onRefsChange: (refs: string[]) => void;
  skip: boolean;
  onSkipChange: (skip: boolean) => void;
  source: string;
  creating: boolean;
}

export default function SchemaRefsFields({
  open,
  namespace,
  refs,
  onRefsChange,
  skip,
  onSkipChange,
  source,
  creating,
}: Props) {
  const {
    data: schemas,
    error,
    isLoading,
  } = useSWR(open ? ['schema-options', namespace] : null, () => listRemoteConfigSchemas(namespace));
  const names = [...new Set([...(schemas ?? []).map((schema) => schema.metadata.name), ...refs])];
  return (
    <div className="min-w-0 space-y-2">
      <fieldset className="min-w-0 space-y-2">
        <legend className="text-sm font-medium">Schema references</legend>
        <p className="text-xs text-muted-foreground">{source}</p>
        {isLoading && <p className="text-xs">Loading schemas…</p>}
        {error && (
          <Alert severity="error">
            Failed to load schemas: {error instanceof Error ? error.message : 'Unknown error'}
          </Alert>
        )}
        {!isLoading && !error && names.length === 0 && (
          <p className="text-xs text-muted-foreground">No schemas in this namespace.</p>
        )}
        <div className="max-h-40 space-y-2 overflow-y-auto">
          {names.map((name) => {
            const schema = schemas?.find((schema) => schema.metadata.name === name);
            return (
              <label key={name} className="flex items-start gap-2 text-sm">
                <Checkbox
                  aria-label={name}
                  checked={refs.includes(name)}
                  onCheckedChange={(checked) =>
                    onRefsChange(
                      checked === true ? [...refs, name] : refs.filter((ref) => ref !== name),
                    )
                  }
                />
                <span className="min-w-0 break-words">
                  {name}
                  <span className="ml-2 text-xs text-muted-foreground">
                    {schema
                      ? `${schema.spec.binary} ${schema.spec.version}`
                      : schemas
                        ? '(unavailable)'
                        : ''}
                  </span>
                </span>
              </label>
            );
          })}
        </div>
        <p className="text-xs text-muted-foreground">
          {creating
            ? 'Leave empty to auto-resolve compatible schemas on create, unless validation is skipped.'
            : 'Clearing references leaves the config unpinned; updates do not auto-resolve them.'}
        </p>
      </fieldset>
      <label className="flex items-center gap-2 text-sm">
        <Switch checked={skip} onCheckedChange={onSkipChange} />
        Skip schema validation
      </label>
    </div>
  );
}
