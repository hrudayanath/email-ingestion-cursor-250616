import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  MenuItem,
  Grid,
  CircularProgress,
  Chip,
  IconButton,
  Tooltip,
} from '@mui/material';
import { DataGrid, GridColDef, GridRenderCellParams, GridPaginationModel } from '@mui/x-data-grid';
import { Summarize as SummarizeIcon, Analytics as AnalyticsIcon } from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { api, Email, EmailListParams } from '../api/client';

type GridValueGetterParams = {
  row: Email;
  value: any;
};

const EmailList: React.FC = () => {
  const navigate = useNavigate();
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState('');
  const [provider, setProvider] = useState<'all' | 'google' | 'microsoft'>('all');

  const { data, isLoading } = useQuery({
    queryKey: ['emails', page, pageSize, search, provider],
    queryFn: () => {
      const params: EmailListParams = {
        page: page + 1,
        limit: pageSize,
        search: search || undefined,
        provider: provider === 'all' ? undefined : provider,
      };
      return api.listEmails(params);
    },
  });

  const columns: GridColDef<Email>[] = [
    {
      field: 'subject',
      headerName: 'Subject',
      flex: 1,
      renderCell: (params: GridRenderCellParams<Email>) => (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 1,
            cursor: 'pointer',
          }}
          onClick={() => navigate(`/emails/${params.row.id}`)}
        >
          <Typography>{params.value}</Typography>
          {params.row.summary && (
            <Tooltip title="Summarized">
              <SummarizeIcon color="primary" fontSize="small" />
            </Tooltip>
          )}
          {params.row.entities && params.row.entities.length > 0 && (
            <Tooltip title="Analyzed">
              <AnalyticsIcon color="secondary" fontSize="small" />
            </Tooltip>
          )}
        </Box>
      ),
    },
    {
      field: 'from',
      headerName: 'From',
      flex: 1,
      valueGetter: (params: GridValueGetterParams) => params.row.from,
    },
    {
      field: 'provider',
      headerName: 'Provider',
      width: 120,
      renderCell: (params: GridRenderCellParams<Email>) => {
        const provider = params.row.accountId.split(':')[0] as 'google' | 'microsoft';
        return (
          <Chip
            label={provider === 'google' ? 'Gmail' : 'Outlook'}
            color={provider === 'google' ? 'primary' : 'secondary'}
            size="small"
          />
        );
      },
    },
    {
      field: 'date',
      headerName: 'Received',
      width: 180,
      valueGetter: (params: GridValueGetterParams) => new Date(params.row.date).toLocaleString(),
    },
    {
      field: 'actions',
      headerName: 'Actions',
      width: 100,
      renderCell: (params: GridRenderCellParams<Email>) => (
        <Box>
          <Tooltip title="View Details">
            <IconButton size="small" onClick={() => navigate(`/emails/${params.row.id}`)}>
              <AnalyticsIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>
      ),
    },
  ];

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h4" component="h1" gutterBottom>
        Emails
      </Typography>
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="Search"
                value={search}
                onChange={e => setSearch(e.target.value)}
                placeholder="Search by subject, sender, or content..."
              />
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                select
                label="Provider"
                value={provider}
                onChange={e => setProvider(e.target.value as 'all' | 'google' | 'microsoft')}
              >
                <MenuItem value="all">All Providers</MenuItem>
                <MenuItem value="google">Gmail</MenuItem>
                <MenuItem value="microsoft">Outlook</MenuItem>
              </TextField>
            </Grid>
          </Grid>
        </CardContent>
      </Card>
      <Card>
        <CardContent>
          <DataGrid<Email>
            rows={data?.emails || []}
            columns={columns}
            rowCount={data?.total || 0}
            pageSizeOptions={[10, 25, 50]}
            paginationModel={{ page, pageSize }}
            onPaginationModelChange={(model: GridPaginationModel) => {
              setPage(model.page);
              setPageSize(model.pageSize);
            }}
            paginationMode="server"
            loading={isLoading}
            autoHeight
            disableRowSelectionOnClick
          />
        </CardContent>
      </Card>
    </Box>
  );
};

export default EmailList;
