const getEnvVar = (key: string): string => {
  const value = process.env[key];
  if (!value) {
    throw new Error(`Environment variable ${key} is not set`);
  }
  return value;
};

export const API_URL = getEnvVar('REACT_APP_API_URL');
export const ENV = getEnvVar('REACT_APP_ENV');

export const isDevelopment = ENV === 'development';
export const isProduction = ENV === 'production';
export const isTest = ENV === 'test';
