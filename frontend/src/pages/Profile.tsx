import React, { useState } from 'react';
import {
  Box,
  Container,
  Paper,
  Typography,
  Avatar,
  Button,
  TextField,
  Grid,
  Divider,
  Alert,
  CircularProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import { useAuth } from '../contexts/AuthContext';
import { api } from '../api/client';

export default function Profile() {
  const { user, logout } = useAuth();
  const [isEditing, setIsEditing] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [formData, setFormData] = useState({
    name: user?.name || '',
    email: user?.email || '',
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  });

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleUpdateProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setIsLoading(true);

    try {
      // TODO: Implement profile update API call
      // For now, just simulate a successful update
      await new Promise(resolve => setTimeout(resolve, 1000));
      setSuccess('Profile updated successfully');
      setIsEditing(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update profile');
    } finally {
      setIsLoading(false);
    }
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setIsLoading(true);

    if (formData.newPassword !== formData.confirmPassword) {
      setError('New passwords do not match');
      setIsLoading(false);
      return;
    }

    try {
      // TODO: Implement password change API call
      // For now, just simulate a successful update
      await new Promise(resolve => setTimeout(resolve, 1000));
      setSuccess('Password changed successfully');
      setFormData(prev => ({
        ...prev,
        currentPassword: '',
        newPassword: '',
        confirmPassword: '',
      }));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to change password');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDeleteAccount = async () => {
    setError(null);
    setIsLoading(true);

    try {
      // TODO: Implement account deletion API call
      // For now, just simulate a successful deletion
      await new Promise(resolve => setTimeout(resolve, 1000));
      logout();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete account');
      setShowDeleteDialog(false);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Container maxWidth="md">
      <Box sx={{ mt: 4, mb: 4 }}>
        <Paper sx={{ p: 4 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 4 }}>
            <Avatar src={user?.picture} alt={user?.name} sx={{ width: 100, height: 100, mr: 3 }} />
            <Box>
              <Typography variant="h4" gutterBottom>
                {user?.name}
              </Typography>
              <Typography variant="body1" color="text.secondary">
                {user?.email}
              </Typography>
            </Box>
          </Box>

          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}

          {success && (
            <Alert severity="success" sx={{ mb: 2 }}>
              {success}
            </Alert>
          )}

          <Divider sx={{ my: 3 }} />

          {/* Profile Information */}
          <Box component="form" onSubmit={handleUpdateProfile}>
            <Typography variant="h6" gutterBottom>
              Profile Information
            </Typography>
            <Grid container spacing={3}>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Name"
                  name="name"
                  value={formData.name}
                  onChange={handleInputChange}
                  disabled={!isEditing || isLoading}
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Email"
                  name="email"
                  value={formData.email}
                  disabled
                  helperText="Email cannot be changed"
                />
              </Grid>
            </Grid>
            <Box sx={{ mt: 2 }}>
              {isEditing ? (
                <>
                  <Button type="submit" variant="contained" disabled={isLoading} sx={{ mr: 1 }}>
                    {isLoading ? <CircularProgress size={24} /> : 'Save Changes'}
                  </Button>
                  <Button
                    variant="outlined"
                    onClick={() => setIsEditing(false)}
                    disabled={isLoading}
                  >
                    Cancel
                  </Button>
                </>
              ) : (
                <Button variant="outlined" onClick={() => setIsEditing(true)} disabled={isLoading}>
                  Edit Profile
                </Button>
              )}
            </Box>
          </Box>

          <Divider sx={{ my: 3 }} />

          {/* Change Password */}
          <Box component="form" onSubmit={handleChangePassword}>
            <Typography variant="h6" gutterBottom>
              Change Password
            </Typography>
            <Grid container spacing={3}>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Current Password"
                  name="currentPassword"
                  type="password"
                  value={formData.currentPassword}
                  onChange={handleInputChange}
                  disabled={isLoading}
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="New Password"
                  name="newPassword"
                  type="password"
                  value={formData.newPassword}
                  onChange={handleInputChange}
                  disabled={isLoading}
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Confirm New Password"
                  name="confirmPassword"
                  type="password"
                  value={formData.confirmPassword}
                  onChange={handleInputChange}
                  disabled={isLoading}
                />
              </Grid>
            </Grid>
            <Box sx={{ mt: 2 }}>
              <Button type="submit" variant="contained" color="primary" disabled={isLoading}>
                {isLoading ? <CircularProgress size={24} /> : 'Change Password'}
              </Button>
            </Box>
          </Box>

          <Divider sx={{ my: 3 }} />

          {/* Danger Zone */}
          <Box>
            <Typography variant="h6" color="error" gutterBottom>
              Danger Zone
            </Typography>
            <Button
              variant="outlined"
              color="error"
              onClick={() => setShowDeleteDialog(true)}
              disabled={isLoading}
            >
              Delete Account
            </Button>
          </Box>
        </Paper>
      </Box>

      {/* Delete Account Confirmation Dialog */}
      <Dialog
        open={showDeleteDialog}
        onClose={() => setShowDeleteDialog(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Delete Account</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete your account? This action cannot be undone and all your
            data will be permanently deleted.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowDeleteDialog(false)} disabled={isLoading}>
            Cancel
          </Button>
          <Button
            onClick={handleDeleteAccount}
            color="error"
            variant="contained"
            disabled={isLoading}
          >
            {isLoading ? <CircularProgress size={24} /> : 'Delete Account'}
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}
