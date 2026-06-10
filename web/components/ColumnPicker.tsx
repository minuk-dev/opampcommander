'use client';

import { useState } from 'react';
import {
  Checkbox,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Tooltip,
} from '@mui/material';
import { ViewColumn as ViewColumnIcon } from '@mui/icons-material';
import type { ColumnConfig, ColumnVisibility } from '@/lib/column-visibility';

interface Props {
  columns: ColumnConfig[];
  visible: ColumnVisibility;
  onToggle: (id: string) => void;
}

// A button that opens a checklist of the table's columns, letting the user pick
// which ones are shown. Locked columns appear checked and disabled so the full
// set stays discoverable. Pair with `useColumnVisibility` to persist the choice.
export default function ColumnPicker({ columns, visible, onToggle }: Props) {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);

  return (
    <>
      <Tooltip title="Choose columns">
        <IconButton size="small" onClick={(e) => setAnchor(e.currentTarget)} aria-label="Columns">
          <ViewColumnIcon />
        </IconButton>
      </Tooltip>
      <Menu anchorEl={anchor} open={Boolean(anchor)} onClose={() => setAnchor(null)}>
        {columns.map((c) => {
          const checked = c.locked ? true : (visible[c.id] ?? true);
          return (
            <MenuItem key={c.id} dense disabled={c.locked} onClick={() => onToggle(c.id)}>
              <ListItemIcon sx={{ minWidth: 0 }}>
                <Checkbox edge="start" size="small" checked={checked} tabIndex={-1} disableRipple />
              </ListItemIcon>
              <ListItemText primary={c.label} />
            </MenuItem>
          );
        })}
      </Menu>
    </>
  );
}
