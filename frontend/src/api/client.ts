import axios from 'axios';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

const client = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add response interceptor for error handling
client.interceptors.response.use(
  response => response,
  error => {
    if (error.response?.status === 401) {
      // Handle unauthorized access
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// Types
export interface Account {
  id: string;
  provider: 'google' | 'microsoft';
  email: string;
  name: string;
  picture?: string;
  createdAt: string;
  updatedAt: string;
  lastSyncAt: string;
  isActive: boolean;
}

export interface Email {
  id: string;
  accountId: string;
  messageId: string;
  subject: string;
  from: string;
  to: string[];
  cc: string[];
  bcc: string[];
  date: string;
  body: string;
  summary?: string;
  entities?: NEREntity[];
  createdAt: string;
  updatedAt: string;
}

export interface NEREntity {
  text: string;
  type: string;
  start: number;
  end: number;
}

export interface EmailListResponse {
  emails: Email[];
  total: number;
  page: number;
  limit: number;
  hasMore: boolean;
}

// User-related interfaces
export interface UserPreferences {
  theme: 'light' | 'dark' | 'system';
  emailNotifications: boolean;
  language: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  picture?: string;
  provider?: 'local' | 'google' | 'microsoft';
  providerId?: string;
  emailVerified: boolean;
  twoFactorEnabled: boolean;
  preferences?: UserPreferences;
  createdAt: string;
  updatedAt: string;
  lastLogin?: string;
}

interface UpdateProfileRequest {
  name: string;
  picture?: string;
}

interface ChangePasswordRequest {
  currentPassword: string;
  newPassword: string;
}

interface LoginRequest {
  email: string;
  password: string;
  otpCode?: string;
}

interface RegisterRequest {
  email: string;
  password: string;
  name: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface AuthURLResponse {
  url: string;
}

export interface EmailListParams {
  page?: number;
  limit?: number;
  search?: string;
  provider?: 'google' | 'microsoft';
}

// API functions
export const api = {
  // OAuth methods
  getAuthURL: async (provider: 'google' | 'microsoft') => {
    const response = await client.get<{ url: string }>(`/oauth/auth/${provider}`);
    return response.data;
  },

  handleCallback: async (provider: 'google' | 'microsoft', code: string, state: string) => {
    const response = await client.post<{
      tokens: {
        access_token: string;
        refresh_token: string;
        expires_at: string;
        token_type: string;
      };
      user_info: {
        id: string;
        email: string;
        name: string;
        picture?: string;
      };
      account: {
        provider: string;
        email: string;
        name: string;
        picture?: string;
        access_token: string;
        refresh_token: string;
        expires_at: string;
        token_type: string;
      };
    }>(`/oauth/callback/${provider}`, { code, state });
    return response.data;
  },

  refreshToken: async (provider: 'google' | 'microsoft', refreshToken: string) => {
    const response = await client.post<{
      access_token: string;
      refresh_token: string;
      expires_at: string;
      token_type: string;
    }>(`/oauth/refresh/${provider}`, { refresh_token: refreshToken });
    return response.data;
  },

  // Account methods
  addAccount: async (type: 'google' | 'microsoft') => {
    const response = await client.post<{ authUrl: string }>('/accounts', { type });
    return response.data;
  },

  deleteAccount: async (accountId: string) => {
    await client.delete(`/accounts/${accountId}`);
  },

  listAccounts: async () => {
    const response = await client.get<Account[]>('/accounts');
    return response.data;
  },

  // Email methods
  fetchEmails: async (accountId: string) => {
    const response = await client.post<{ count: number }>(`/accounts/${accountId}/emails/fetch`);
    return response.data;
  },

  listEmails: async ({ page = 1, limit = 20, search, provider }: EmailListParams = {}) => {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
    });
    if (search) {
      params.append('search', search);
    }
    if (provider) {
      params.append('provider', provider);
    }
    const response = await client.get<EmailListResponse>(`/emails?${params.toString()}`);
    return response.data;
  },

  getEmail: async (emailId: string) => {
    const response = await client.get<Email>(`/emails/${emailId}`);
    return response.data;
  },

  summarizeEmail: async (emailId: string) => {
    const response = await client.post<{ summary: string }>(`/emails/${emailId}/summarize`);
    return response.data;
  },

  performNER: async (emailId: string) => {
    const response = await client.post<{ entities: NEREntity[] }>(`/emails/${emailId}/ner`);
    return response.data;
  },

  // User management
  auth: {
    register: async (data: RegisterRequest): Promise<AuthResponse> => {
      const response = await client.post('/auth/register', data);
      return response.data;
    },

    login: async (data: LoginRequest): Promise<AuthResponse> => {
      const response = await client.post('/auth/login', data);
      return response.data;
    },

    getAuthURL: async (provider: 'google' | 'microsoft'): Promise<AuthURLResponse> => {
      const response = await client.get<AuthURLResponse>(`/auth/${provider}/url`);
      return response.data;
    },

    logout: () => {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
    },
  },

  user: {
    getProfile: async (): Promise<User> => {
      const response = await client.get('/profile');
      return response.data;
    },

    updateProfile: async (data: UpdateProfileRequest): Promise<User> => {
      const response = await client.put('/profile', data);
      return response.data;
    },

    changePassword: async (data: ChangePasswordRequest): Promise<void> => {
      await client.put('/profile/password', data);
    },

    enable2FA: async (): Promise<{ secret: string }> => {
      const response = await client.post('/profile/2fa/enable');
      return response.data;
    },

    disable2FA: async (code: string): Promise<void> => {
      await client.post('/profile/2fa/disable', { code });
    },

    verify2FA: async (code: string): Promise<void> => {
      await client.post('/profile/2fa/verify', { code });
    },

    updatePreferences: async (preferences: UserPreferences): Promise<void> => {
      await client.put('/profile/preferences', preferences);
    },

    deleteAccount: async (password?: string): Promise<void> => {
      await client.delete('/profile', { data: { password } });
    },
  },
};
