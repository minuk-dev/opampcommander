'use client';

import { Search, Tags, X } from 'lucide-react';
import { type ReactNode, useState } from 'react';
import { EMPTY_LIST_FILTERS, type ListFilters } from '@shared/lib';
import Badge from './Badge';
import Button from './Button';
import { Card } from './Card';
import Field from './Field';
import Input from './Input';

interface Props {
  // The applied filters, i.e. the ones currently in the request.
  value: ListFilters;
  // Called with the new filters when the user applies or clears one.
  onChange: (next: ListFilters) => void;
  // Overrides the name field's label/placeholder for resources whose name is not
  // called "name" (an agent's instance UID, a user's email).
  nameLabel?: string;
  namePlaceholder?: string;
  // Extra controls rendered after the inputs (a status toggle, for instance).
  children?: ReactNode;
}

// A removable pill for one applied filter.
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

// ListFilterBar is the filter row shared by the list pages. Both fields are
// answered by the server: the name prefix by an index range scan, the label
// selector by the datastore query. Nothing here filters the fetched page, so the
// paginated total always describes the set the rows were drawn from.
//
// Filters apply on submit rather than on every keystroke: each change is a new
// request and a reset to page 0, so typing into a live filter would fetch once
// per character.
export default function ListFilterBar({
  value,
  onChange,
  nameLabel = 'Name',
  namePlaceholder = 'Name starts with…',
  children,
}: Props) {
  const [name, setName] = useState(value.name);
  const [labelSelector, setLabelSelector] = useState(value.labelSelector);

  // Keep the draft in step when the applied filters change from the outside — a
  // chip cleared, or a page rendered with filters already set. Adjusted during
  // render rather than in an effect so the new value is on screen in the same
  // paint, and so clearing a chip cannot flash the stale text.
  const appliedKey = `${value.name}\u0000${value.labelSelector}`;
  const [prevAppliedKey, setPrevAppliedKey] = useState(appliedKey);
  if (appliedKey !== prevAppliedKey) {
    setPrevAppliedKey(appliedKey);
    setName(value.name);
    setLabelSelector(value.labelSelector);
  }

  const apply = (event: React.FormEvent) => {
    event.preventDefault();
    onChange({ ...value, name: name.trim(), labelSelector: labelSelector.trim() });
  };

  const applied = [
    value.name && { key: 'name', label: `${nameLabel}: ${value.name}`, patch: { name: '' } },
    value.labelSelector && {
      key: 'labels',
      label: `Labels: ${value.labelSelector}`,
      patch: { labelSelector: '' },
    },
    value.fieldSelector && {
      key: 'fields',
      label: `Fields: ${value.fieldSelector}`,
      patch: { fieldSelector: '' },
    },
  ].filter(Boolean) as { key: string; label: string; patch: Partial<ListFilters> }[];

  return (
    <Card className="mb-3 p-3">
      <form onSubmit={apply} className="flex flex-col gap-2 sm:flex-row">
        <Field label={nameLabel} className="flex-1">
          {(field) => (
            <Input
              {...field}
              startSlot={<Search aria-hidden />}
              placeholder={namePlaceholder}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          )}
        </Field>

        <Field label="Labels" className="flex-1">
          {(field) => (
            <Input
              {...field}
              startSlot={<Tags aria-hidden />}
              placeholder="env=prod, tier notin (canary,dev), !deprecated"
              value={labelSelector}
              onChange={(e) => setLabelSelector(e.target.value)}
            />
          )}
        </Field>

        <Button type="submit" className="self-end">
          Filter
        </Button>
      </form>

      {children}

      {applied.length > 0 && (
        <div className="mt-2 flex flex-wrap items-center gap-1.5">
          <span className="text-xs text-muted-foreground">Filters:</span>
          {applied.map((chip) => (
            <FilterChip
              key={chip.key}
              label={chip.label}
              onClear={() => onChange({ ...value, ...chip.patch })}
            />
          ))}
          <Button
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-xs"
            onClick={() => onChange(EMPTY_LIST_FILTERS)}
          >
            Clear all
          </Button>
        </div>
      )}
    </Card>
  );
}
