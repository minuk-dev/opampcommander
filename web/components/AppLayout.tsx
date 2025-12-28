'use client';

import { AppBar, Box, Drawer, List, ListItem, ListItemButton, ListItemIcon, ListItemText, Toolbar, Typography } from '@mui/material';
import { Dashboard as DashboardIcon, Group as GroupIcon, Computer as ComputerIcon } from '@mui/icons-material';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ReactNode } from 'react';

const drawerWidth = 240;

const menuItems = [
  { text: 'Dashboard', icon: <DashboardIcon />, href: '/' },
  { text: 'Agents', icon: <ComputerIcon />, href: '/agents' },
  { text: 'Agent Groups', icon: <GroupIcon />, href: '/agentgroups' },
];

export default function AppLayout({ children }: { children: ReactNode }) {
  const pathname = usePathname();

  return (
    <Box sx={{ display: 'flex' }}>
      <AppBar position="fixed" sx={{ zIndex: (theme) => theme.zIndex.drawer + 1 }}>
        <Toolbar>
          <Typography variant="h6" noWrap component="div">
            OpAMP Commander
          </Typography>
        </Toolbar>
      </AppBar>
      <Drawer
        variant="permanent"
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          [`& .MuiDrawer-paper`]: { width: drawerWidth, boxSizing: 'border-box' },
        }}
      >
        <Toolbar />
        <Box sx={{ overflow: 'auto' }}>
          <List>
            {menuItems.map((item) => (
              <ListItem key={item.text} disablePadding>
                <ListItemButton
                  component={Link}
                  href={item.href}
                  selected={pathname === item.href}
                >
                  <ListItemIcon>{item.icon}</ListItemIcon>
                  <ListItemText primary={item.text} />
                </ListItemButton>
              </ListItem>
            ))}
          </List>
        </Box>
      </Drawer>
      <Box component="main" sx={{ flexGrow: 1, p: 3 }}>
        <Toolbar />
        {children}
      </Box>
    </Box>
  );
}
