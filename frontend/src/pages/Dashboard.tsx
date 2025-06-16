import React from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Box,
  Card,
  CardContent,
  Grid,
  Typography,
  CircularProgress,
} from '@mui/material';
import {
  Email as EmailIcon,
  AccountCircle as AccountIcon,
  Summarize as SummarizeIcon,
  Analytics as AnalyticsIcon,
} from '@mui/icons-material';
import { api, EmailListParams } from '../api/client';

const StatCard: React.FC<{
  title: string;
  value: string | number;
  icon: React.ReactNode;
  color: string;
}> = ({ title, value, icon, color }) => (
  <Card sx={{ height: '100%' }}>
    <CardContent>
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
        <Box
          sx={{
            backgroundColor: `${color}15`,
            borderRadius: '50%',
            p: 1,
            mr: 2,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          {React.cloneElement(icon as React.ReactElement, {
            sx: { color },
          })}
        </Box>
        <Typography variant="h6" component="div">
          {title}
        </Typography>
      </Box>
      <Typography variant="h4" component="div" sx={{ fontWeight: 'bold' }}>
        {value}
      </Typography>
    </CardContent>
  </Card>
);

const Dashboard: React.FC = () => {
  const { data: emails, isLoading: emailsLoading } = useQuery({
    queryKey: ['emails', 'dashboard'],
    queryFn: () => {
      const params: EmailListParams = {
        page: 1,
        limit: 1,
      };
      return api.listEmails(params);
    },
  });

  const { data: accounts, isLoading: accountsLoading } = useQuery({
    queryKey: ['accounts'],
    queryFn: api.listAccounts,
  });

  if (emailsLoading || accountsLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  const totalEmails = emails?.total || 0;
  const totalAccounts = accounts?.length || 0;
  const summarizedEmails = emails?.emails.filter((e) => e.summary).length || 0;
  const analyzedEmails = emails?.emails.filter((e) => e.entities && e.entities.length > 0).length || 0;

  return (
    <Box>
      <Typography variant="h4" component="h1" gutterBottom>
        Dashboard
      </Typography>
      <Grid container spacing={3}>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Total Emails"
            value={totalEmails}
            icon={<EmailIcon />}
            color="#1976d2"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Connected Accounts"
            value={totalAccounts}
            icon={<AccountIcon />}
            color="#2e7d32"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Summarized Emails"
            value={summarizedEmails}
            icon={<SummarizeIcon />}
            color="#ed6c02"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Analyzed Emails"
            value={analyzedEmails}
            icon={<AnalyticsIcon />}
            color="#9c27b0"
          />
        </Grid>
      </Grid>
    </Box>
  );
};

export default Dashboard; 