import axios, { AxiosError } from 'axios';

export class APIError extends Error {
  constructor(
    message: string,
    public status?: number,
    public code?: string
  ) {
    super(message);
    this.name = 'APIError';
  }
}

export const handleAPIError = (error: unknown): APIError => {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<{ message: string; code: string }>;
    const message =
      axiosError.response?.data?.message ||
      axiosError.message ||
      'An unexpected error occurred';
    const status = axiosError.response?.status;
    const code = axiosError.response?.data?.code;

    return new APIError(message, status, code);
  }

  if (error instanceof Error) {
    return new APIError(error.message);
  }

  return new APIError('An unexpected error occurred');
};

export const isAPIError = (error: unknown): error is APIError => {
  return error instanceof APIError;
}; 