'use client';

import { Divider, IconButton, ListItemIcon, Menu, MenuItem, Tooltip } from '@mui/material';
import { MoreVert as MoreVertIcon } from '@mui/icons-material';
import { useRouter } from 'next/navigation';
import { type ReactNode, useState } from 'react';

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

// Three-dot vertical "more" menu for table rows. Each menu item either
// navigates (href) or runs onClick. Click-to-stopPropagation is applied so
// the menu inside a clickable row doesn't trigger the row's own link.
export default function RowActionsMenu({ actions, tooltip = 'Actions' }: Props) {
  const router = useRouter();
  const [anchor, setAnchor] = useState<null | HTMLElement>(null);
  const open = Boolean(anchor);

  // Don't render an empty action surface — a menu with no items is just
  // a confusing flash on click.
  if (actions.length === 0) return null;

  return (
    <>
      <Tooltip title={tooltip}>
        <IconButton
          size="small"
          aria-label={tooltip}
          onClick={(e) => {
            e.stopPropagation();
            e.preventDefault();
            setAnchor(e.currentTarget);
          }}
        >
          <MoreVertIcon fontSize="small" />
        </IconButton>
      </Tooltip>
      <Menu
        anchorEl={anchor}
        open={open}
        onClose={() => setAnchor(null)}
        onClick={(e) => e.stopPropagation()}
        slotProps={{
          paper: { sx: { minWidth: 180 } },
        }}
      >
        {actions.flatMap((a, i) => {
          const item = (
            <MenuItem
              key={`m-${i}`}
              onClick={(e) => {
                e.stopPropagation();
                setAnchor(null);
                if (a.href) {
                  router.push(a.href);
                } else if (a.onClick) {
                  void a.onClick();
                }
              }}
              sx={
                a.destructive
                  ? {
                      color: 'error.main',
                      '& .MuiListItemIcon-root': { color: 'error.main' },
                    }
                  : undefined
              }
            >
              {a.icon && <ListItemIcon>{a.icon}</ListItemIcon>}
              {a.label}
            </MenuItem>
          );
          return a.divider ? [<Divider key={`d-${i}`} />, item] : [item];
        })}
      </Menu>
    </>
  );
}
