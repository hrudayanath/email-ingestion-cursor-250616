import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  MenuItem,
  Grid,
  CircularProgress,
  Chip,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { addAccount, deleteAccount, fetchEmails, Account } from '../api/client';

const AccountList: React.FC = () => {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [provider, setProvider] = useState<'gmail' | 'outlook'>('gmail');
  const [isFetching, setIsFetching] = useState<string | null>(null);

  const { data: accounts, isLoading } = useQuery({
    queryKey: ['accounts'],
    queryFn: () => Promise.resolve([]), // TODO: Implement getAccounts API
  });

  const addAccountMutation = useMutation({
    mutationFn: (provider: 'gmail' | 'outlook') => addAccount(provider),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
      setOpen(false);
    },
  });

  const deleteAccountMutation = useMutation({
    mutationFn: (accountId: string) => deleteAccount(accountId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
    },
  });

  const handleFetchEmails = async (accountId: string) => {
    setIsFetching(accountId);
    try {
      await fetchEmails(accountId);
      queryClient.invalidateQueries({ queryKey: ['emails'] });
    } finally {
      setIsFetching(null);
    }
  };

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h4" component="h1">
          Email Accounts
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setOpen(true)}
        >
          Add Account
        </Button>
      </Box>

      <Grid container spacing={3}>
        {accounts?.map((account: Account) => (
          <Grid item xs={12} md={6} key={account.id}>
            <Card>
              <CardContent>
                <Box
                  sx={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'flex-start',
                  }}
                >
                  <Box>
                    <Typography variant="h6" gutterBottom>
                      {account.email}
                    </Typography>
                    <Chip
                      label={account.provider}
                      color={account.provider === 'gmail' ? 'primary' : 'secondary'}
                      size="small"
                      sx={{ mr: 1 }}
                    />
                    <Chip
                      label={account.status}
                      color={account.status === 'active' ? 'success' : 'error'}
                      size="small"
                    />
                  </Box>
                  <Box>
                    <Tooltip title="Fetch Emails">
                      <IconButton
                        onClick={() => handleFetchEmails(account.id)}
                        disabled={isFetching === account.id}
                      >
                        {isFetching === account.id ? (
                          <CircularProgress size={24} />
                        ) : (
                          <RefreshIcon />
                        )}
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Delete Account">
                      <IconButton
                        onClick={() => deleteAccountMutation.mutate(account.id)}
                        color="error"
                      >
                        <DeleteIcon />
                      </IconButton>
                    </Tooltip>
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>

      <Dialog open={open} onClose={() => setOpen(false)}>
        <DialogTitle>Add Email Account</DialogTitle>
        <DialogContent>
          <TextField
            select
            fullWidth
            label="Provider"
            value={provider}
            onChange={(e) => setProvider(e.target.value as 'gmail' | 'outlook')}
            sx={{ mt: 2 }}
          >
            <MenuItem value="gmail">Gmail</MenuItem>
            <MenuItem value="outlook">Outlook</MenuItem>
          </TextField>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button
            onClick={() => addAccountMutation.mutate(provider)}
            variant="contained"
            disabled={addAccountMutation.isPending}
          >
            Connect
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default AccountList; 