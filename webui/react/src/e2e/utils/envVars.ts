export function getEnvVar(key: string): string {
  const value = process.env[key];
  if (value === undefined) {
    throw new Error(`Expected ${key} to be set. ${JSON.stringify(process.env)}`);
  }
  return value;
}

export const baseUrl = (): string => getEnvVar('PW_BASE_URL');
export const username = (): string => getEnvVar('PW_USERNAME');
export const password = (): string => getEnvVar('PW_PASSWORD');

export const isEE = (): boolean => Boolean(JSON.parse(process.env.PW_EE ?? '""'));
export const apiUrl = (): string => process.env.PW_SERVER_ADDRESS ?? baseUrl();
export const detMasterURL = (): string => process.env.PW_DET_MASTER ?? 'localhost:8080';
export const detPath = (): string => process.env.PW_DET_PATH || 'det';

export const defaultLandingURL = (): RegExp => isEE() ? /workspaces/ : /dashboard/;
export const defaultLandingTitle = (): string => isEE() ? 'Workspaces' : 'Home';
