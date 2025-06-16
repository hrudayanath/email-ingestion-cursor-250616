import React, { useState } from 'react';
import { useNavigate, Link as RouterLink } from 'react-router-dom';
import {
  Box,
  Button,
  Container,
  TextField,
  Typography,
  Paper,
  Divider,
  Alert,
  CircularProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Link,
} from '@mui/material';
import { Google as GoogleIcon, Microsoft as MicrosoftIcon } from '@mui/icons-material';
import { useAuth } from '../contexts/AuthContext';
import { api } from '../api/client';
import type { AuthResponse } from '../api/client';

interface LoginFormData {
  email: string;
  password: string;
  otp?: string;
}

export default function Login() {
  const navigate = useNavigate();
  const { login } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>('');
  const [show2FADialog, setShow2FADialog] = useState(false);
  const [formData, setFormData] = useState<LoginFormData>({
    email: '',
    password: '',
    otp: '',
  });
  const [otpCode, setOtpCode] = useState('');
  const [pendingAuth, setPendingAuth] = useState<{
    email: string;
    password: string;
  } | null>(null);

  const handleError = (error: Error) => {
    setLoading(false);
    if (error.message.includes('2FA required')) {
      setShow2FADialog(true);
    } else {
      setError(error.message);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setLoading(true);
      setError('');
      const response = await api.auth.login({
        email: formData.email,
        password: formData.password,
        otp: formData.otp || undefined,
      });
      login(response.token, response.user);
      navigate('/dashboard');
    } catch (error: unknown) {
      const err = error as Error;
      handleError(err);
    }
  };

  const handle2FASubmit = async () => {
    if (!formData.email || !formData.password) return;

    try {
      setLoading(true);
      setError('');
      const response = await api.auth.login({
        email: formData.email,
        password: formData.password,
        otp: otpCode,
      });
      login(response.token, response.user);
      navigate('/dashboard');
    } catch (error: unknown) {
      const err = error as Error;
      setError(err.message);
    } finally {
      setLoading(false);
      setShow2FADialog(false);
      setOtpCode('');
    }
  };

  const handleOAuthLogin = async (provider: 'google' | 'microsoft') => {
    try {
      setLoading(true);
      setError('');
      const url = await api.auth.getAuthURL(provider);
      window.location.href = url;
    } catch (error: unknown) {
      const err = error as Error;
      handleError(err);
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  return (
    <Container component="main" maxWidth="xs">
      <Box
        sx={{
          marginTop: 8,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
        }}
      >
        <Paper
          elevation={3}
          sx={{
            padding: 4,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            width: '100%',
          }}
        >
          <Typography component="h1" variant="h5" gutterBottom>
            Sign In
          </Typography>

          {error && (
            <Typography variant="body2" color="error" align="center" sx={{ mt: 2 }}>
              {error}
            </Typography>
          )}

          <Box component="form" onSubmit={handleSubmit} sx={{ width: '100%' }}>
            <TextField
              margin="normal"
              required
              fullWidth
              id="email"
              label="Email Address"
              name="email"
              autoComplete="email"
              type="email"
              value={formData.email}
              onChange={handleInputChange}
              disabled={loading}
            />
            <TextField
              margin="normal"
              required
              fullWidth
              name="password"
              label="Password"
              type="password"
              id="password"
              autoComplete="current-password"
              value={formData.password}
              onChange={handleInputChange}
              disabled={loading}
            />
            <Button
              type="submit"
              fullWidth
              variant="contained"
              sx={{ mt: 3, mb: 2 }}
              disabled={loading}
            >
              {loading ? <CircularProgress size={24} /> : 'Sign In'}
            </Button>
          </Box>

          <Divider sx={{ width: '100%', my: 2 }}>OR</Divider>

          <Box sx={{ width: '100%', display: 'flex', gap: 2 }}>
            <Button
              fullWidth
              variant="outlined"
              startIcon={<GoogleIcon />}
              onClick={() => handleOAuthLogin('google')}
              disabled={loading}
            >
              Sign in with Google
            </Button>
            <Button
              fullWidth
              variant="outlined"
              startIcon={<MicrosoftIcon />}
              onClick={() => handleOAuthLogin('microsoft')}
              disabled={loading}
            >
              Sign in with Microsoft
            </Button>
          </Box>

          <Box sx={{ mt: 3, textAlign: 'center' }}>
            <Typography variant="body2" color="text.secondary">
              Don&#39;t have an account?{' '}
              <Link component={RouterLink} to="/register" variant="body2">
                Sign up
              </Link>
            </Typography>
          </Box>
        </Paper>
      </Box>

      {/* 2FA Dialog */}
      <Dialog open={show2FADialog} onClose={() => !loading && setShow2FADialog(false)}>
        <DialogTitle>Two-Factor Authentication</DialogTitle>
        <DialogContent>
          <Typography variant="body2" sx={{ mb: 2 }}>
            Please enter the 6-digit code from your authenticator app.
          </Typography>
          <TextField
            autoFocus
            margin="dense"
            label="Authentication Code"
            type="text"
            fullWidth
            value={otpCode}
            onChange={e => setOtpCode(e.target.value)}
            disabled={loading}
            inputProps={{
              maxLength: 6,
              pattern: '[0-9]*',
            }}
          />
          {error && (
            <Alert severity="error" sx={{ mt: 2 }}>
              {error}
            </Alert>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShow2FADialog(false)} disabled={loading}>
            Cancel
          </Button>
          <Button onClick={handle2FASubmit} disabled={loading || otpCode.length !== 6}>
            {loading ? <CircularProgress size={24} /> : 'Verify'}
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}
