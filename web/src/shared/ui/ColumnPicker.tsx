'use client';

import { Columns3 } from 'lucide-react';
import type { ColumnConfig, ColumnVisibility } from '@shared/lib';
import Button from './Button';
import Checkbox from './Checkbox';
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from './DropdownMenu';

interface Props {
  columns: ColumnConfig[];
  visible: ColumnVisibility;
  onToggle: (id: string) => void;
}

// A button that opens a checklist of the table's columns, letting the user pick
// which ones are shown. Locked columns appear checked and disabled so the full
// set stays discoverable. Pair with `useColumnVisibility` to persist the choice.
export default function ColumnPicker({ columns, visible, onToggle }: Props) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon-sm" aria-label="Columns">
          <Columns3 aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="max-h-80 overflow-y-auto">
        {columns.map((c) => {
          const checked = c.locked ? true : (visible[c.id] ?? true);
          return (
            <label
              key={c.id}
              className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent data-disabled:cursor-default data-disabled:opacity-50"
              data-disabled={c.locked ? '' : undefined}
            >
              <Checkbox
                checked={checked}
                disabled={c.locked}
                onCheckedChange={() => onToggle(c.id)}
              />
              {c.label}
            </label>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
