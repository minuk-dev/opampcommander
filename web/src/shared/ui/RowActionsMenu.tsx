'use client';

import { MoreVertical } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { type ReactNode } from 'react';
import Button from './Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from './DropdownMenu';

export interface RowAction {
  label: string;
  icon?: ReactNode;
  // Either set href (navigation) or onClick (inline action), not both.
  href?: string;
  onClick?: () => void | Promise<void>;
  destructive?: boolean;
  // Optional separator before this item.
  divider?: boolean;
}

interface Props {
  actions: RowAction[];
  tooltip?: string;
}

// Three-dot "more" menu for table rows. Each item either navigates (href) or
// runs onClick. Clicks stop propagating so the menu inside a clickable row
// doesn't also trigger the row's own link.
export default function RowActionsMenu({ actions, tooltip = 'Actions' }: Props) {
  const router = useRouter();

  // Don't render an empty action surface — a menu with no items is just a
  // confusing flash on click.
  if (actions.length === 0) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon-sm"
          aria-label={tooltip}
          onClick={(e) => e.stopPropagation()}
        >
          <MoreVertical aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent onClick={(e) => e.stopPropagation()}>
        {actions.flatMap((a, i) => {
          const item = (
            <DropdownMenuItem
              key={`m-${i}`}
              destructive={a.destructive}
              onSelect={() => {
                if (a.href) router.push(a.href);
                else if (a.onClick) void a.onClick();
              }}
            >
              {a.icon}
              {a.label}
            </DropdownMenuItem>
          );
          return a.divider ? [<DropdownMenuSeparator key={`d-${i}`} />, item] : [item];
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
