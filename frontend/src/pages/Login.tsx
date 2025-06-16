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

export default function Login() {
  const navigate = useNavigate();
  const { login } = useAuth();
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    email: '',
    password: '',
  });
  const [show2FADialog, setShow2FADialog] = useState(false);
  const [otpCode, setOtpCode] = useState('');
  const [pendingAuth, setPendingAuth] = useState<{
    email: string;
    password: string;
  } | null>(null);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setIsLoading(true);

    try {
      const { token, user } = await api.auth.login({
        email: formData.email,
        password: formData.password,
      });

      if (user.twoFactorEnabled) {
        setPendingAuth({ email: formData.email, password: formData.password });
        setShow2FADialog(true);
        setIsLoading(false);
        return;
      }

      login(token, user);
      navigate('/dashboard', { replace: true });
    } catch (err: any) {
      if (err.response?.status === 401) {
        setError('Invalid email or password');
      } else {
        setError(err.response?.data?.message || 'An error occurred during login');
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handle2FASubmit = async () => {
    if (!pendingAuth) return;

    setIsLoading(true);
    setError(null);

    try {
      const { token, user } = await api.auth.login({
        email: pendingAuth.email,
        password: pendingAuth.password,
        otpCode,
      });

      login(token, user);
      setShow2FADialog(false);
      navigate('/dashboard', { replace: true });
    } catch (err: any) {
      if (err.response?.status === 401) {
        setError('Invalid OTP code');
      } else {
        setError(err.response?.data?.message || 'An error occurred during 2FA verification');
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleOAuthLogin = async (provider: 'google' | 'microsoft') => {
    setError(null);
    setIsLoading(true);

    try {
      const { url } = await api.auth.getAuthURL(provider);
      window.location.href = url;
    } catch (err: any) {
      setError(err.response?.data?.message || `Failed to start ${provider} login`);
      setIsLoading(false);
    }
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
            <Alert severity="error" sx={{ width: '100%', mb: 2 }}>
              {error}
            </Alert>
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
              disabled={isLoading}
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
              disabled={isLoading}
            />
            <Button
              type="submit"
              fullWidth
              variant="contained"
              sx={{ mt: 3, mb: 2 }}
              disabled={isLoading}
            >
              {isLoading ? <CircularProgress size={24} /> : 'Sign In'}
            </Button>
          </Box>

          <Divider sx={{ width: '100%', my: 2 }}>OR</Divider>

          <Box sx={{ width: '100%', display: 'flex', gap: 2 }}>
            <Button
              fullWidth
              variant="outlined"
              startIcon={<GoogleIcon />}
              onClick={() => handleOAuthLogin('google')}
              disabled={isLoading}
            >
              Sign in with Google
            </Button>
            <Button
              fullWidth
              variant="outlined"
              startIcon={<MicrosoftIcon />}
              onClick={() => handleOAuthLogin('microsoft')}
              disabled={isLoading}
            >
              Sign in with Microsoft
            </Button>
          </Box>

          <Box sx={{ mt: 3, textAlign: 'center' }}>
            <Typography variant="body2" color="text.secondary">
              Don't have an account?{' '}
              <Link component={RouterLink} to="/register" variant="body2">
                Sign up
              </Link>
            </Typography>
          </Box>
        </Paper>
      </Box>

      {/* 2FA Dialog */}
      <Dialog open={show2FADialog} onClose={() => !isLoading && setShow2FADialog(false)}>
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
            disabled={isLoading}
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
          <Button onClick={() => setShow2FADialog(false)} disabled={isLoading}>
            Cancel
          </Button>
          <Button onClick={handle2FASubmit} disabled={isLoading || otpCode.length !== 6}>
            {isLoading ? <CircularProgress size={24} /> : 'Verify'}
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}
