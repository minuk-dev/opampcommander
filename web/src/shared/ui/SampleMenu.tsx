'use client';

import { ChevronDown } from 'lucide-react';
import Button from './Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from './DropdownMenu';
// Any sample shape works as long as it can label itself — the AgentGroup
// editor's samples carry split name/attributes/spec fields rather than one
// serialized value.
interface SampleLike {
  label: string;
  description?: string;
}

interface Props<T extends SampleLike> {
  // null while the sample file is still being fetched.
  samples: T[] | null;
  onPick: (sample: T) => void;
}

// The "Load sample" dropdown shared by every editor that pre-fills itself from
// web/public/samples/*.yaml.
export default function SampleMenu<T extends SampleLike>({ samples, onPick }: Props<T>) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" disabled={samples === null}>
          {samples === null ? 'Loading…' : 'Load sample'}
          <ChevronDown aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="max-w-80">
        {(samples ?? []).length === 0 && (
          <DropdownMenuItem disabled>No samples available</DropdownMenuItem>
        )}
        {(samples ?? []).map((s, i) => (
          <DropdownMenuItem
            key={`${i}-${s.label}`}
            onSelect={() => onPick(s)}
            className="flex-col items-start gap-0"
          >
            <span>{s.label}</span>
            {s.description && (
              <span className="text-xs text-muted-foreground">{s.description}</span>
            )}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
