'use client';

import { Box, Typography, Grid, Card, CardContent } from '@mui/material';
import { Computer as ComputerIcon, Group as GroupIcon } from '@mui/icons-material';

export default function Home() {
  return (
    <Box>
      <Typography variant="h4" component="h1" gutterBottom>
        Dashboard
      </Typography>
      <Typography variant="body1" color="text.secondary" paragraph>
        Welcome to OpAMP Commander Web Interface
      </Typography>

      <Grid container spacing={3} sx={{ mt: 2 }}>
        <Grid xs={12} md={6}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" gap={2}>
                <ComputerIcon fontSize="large" color="primary" />
                <Box>
                  <Typography variant="h6">Agents</Typography>
                  <Typography variant="body2" color="text.secondary">
                    Manage and monitor your OpAMP agents
                  </Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid xs={12} md={6}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" gap={2}>
                <GroupIcon fontSize="large" color="primary" />
                <Box>
                  <Typography variant="h6">Agent Groups</Typography>
                  <Typography variant="body2" color="text.secondary">
                    Organize agents into groups
                  </Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}
