import axios from 'axios';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

export const apiClient = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Types
export interface Account {
  id: string;
  type: 'gmail' | 'outlook';
  email: string;
  createdAt: string;
  updatedAt: string;
}

export interface Email {
  id: string;
  accountId: string;
  messageId: string;
  from: string;
  to: string[];
  cc: string[];
  bcc: string[];
  subject: string;
  body: string;
  htmlBody: string;
  receivedAt: string;
  createdAt: string;
  updatedAt: string;
  labels: string[];
  summary?: string;
  nerEntities?: NEREntity[];
}

export interface NEREntity {
  text: string;
  type: string;
  startPos: number;
  endPos: number;
  confidence: number;
}

export interface EmailListResponse {
  emails: Email[];
  total: number;
  page: number;
  limit: number;
}

// API functions
export const api = {
  // Account operations
  addAccount: async (type: 'gmail' | 'outlook'): Promise<{ authUrl: string }> => {
    const response = await apiClient.post('/accounts', { type });
    return response.data;
  },

  deleteAccount: async (accountId: string): Promise<void> => {
    await apiClient.delete(`/accounts/${accountId}`);
  },

  fetchEmails: async (accountId: string): Promise<void> => {
    await apiClient.get(`/accounts/${accountId}/emails`);
  },

  // Email operations
  listEmails: async (page = 1, limit = 20, accountId?: string): Promise<EmailListResponse> => {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
    });
    if (accountId) {
      params.append('account_id', accountId);
    }
    const response = await apiClient.get(`/emails?${params.toString()}`);
    return response.data;
  },

  getEmail: async (emailId: string): Promise<Email> => {
    const response = await apiClient.get(`/emails/${emailId}`);
    return response.data;
  },

  summarizeEmail: async (emailId: string): Promise<void> => {
    await apiClient.post(`/emails/${emailId}/summarize`);
  },

  performNER: async (emailId: string): Promise<void> => {
    await apiClient.post(`/emails/${emailId}/ner`);
  },
}; 