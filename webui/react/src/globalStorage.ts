import { Storage } from 'utils/storage';

class GlobalStorage {
  private keys: Record<string, string>;
  private storage: Storage;

  constructor(storage: Storage) {
    this.storage = storage;
    this.keys = {
      authToken: 'auth-token',
      serverAddress: 'server-address',
    };
  }

  set authToken(token: string) {
    this.storage.set(this.keys.authToken, token);
  }

  get authToken() {
    return this.storage.get<string>(this.keys.authToken) || '';
  }

  removeAuthToken() {
    this.storage.remove(this.keys.authToken);
  }

  set serverAddress(address: string) {
    this.storage.set(this.keys.serverAddress, address);
  }

  get serverAddress() {
    return this.storage.get<string>(this.keys.serverAddress) || '';
  }

  removeServerAddress() {
    this.storage.remove(this.keys.serverAddress);
  }
}

export const globalStorage = new GlobalStorage(
  new Storage({ basePath: 'global', store: window.localStorage }),
);
