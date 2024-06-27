export function webServerUrl(): string {
  const serverAddess = process.env.PW_SERVER_ADDRESS;
  if (serverAddess === undefined) {
    throw new Error(`Expected PW_SERVER_ADDRESS to be set. ${JSON.stringify(process.env)}`);
  }
  return serverAddess;
}

export function apiUrl(): string {
  return process.env.PW_BACKEND_SERVER_ADDRESS ?? webServerUrl();
}

export function detMasterURL(): string {
  return process.env.PW_BACKEND_SERVER_ADDRESS ?? 'localhost:8080';
}
