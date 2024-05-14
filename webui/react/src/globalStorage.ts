import { StorageManager } from 'utils/storage';

class GlobalStorage {
  private keys: Record<string, string>;
  private storage: StorageManager;

  constructor(storage: StorageManager) {
    this.storage = storage;
    this.keys = {
      authToken: 'auth-token',
      landingRedirect: 'landing-redirect',
      serverAddress: 'server-address',
    };
  }

  get authToken() {
    return this.storage.get<string>(this.keys.authToken) || '';
  }

  get serverAddress() {
    return this.storage.get<string>(this.keys.serverAddress) || '';
  }

  get landingRedirect() {
    return this.storage.get<string>(this.keys.landingRedirect) || '';
  }

  set authToken(token: string) {
    this.storage.set(this.keys.authToken, token);
  }

  set serverAddress(address: string) {
    this.storage.set(this.keys.serverAddress, address);
  }

  set landingRedirect(address: string) {
    this.storage.set(this.keys.landingRedirect, address);
  }

  removeAuthToken() {
    this.storage.remove(this.keys.authToken);
  }

  removeServerAddress() {
    this.storage.remove(this.keys.serverAddress);
  }

  removeLandingRedirect() {
    this.storage.remove(this.keys.landingRedirect);
  }
}

export const globalStorage = new GlobalStorage(
  new StorageManager({ basePath: 'global', store: window.localStorage }),
);
