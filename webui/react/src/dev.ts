/* Tools and tweaks for dev environments */
import { globalStorage } from 'globalStorage';
import * as Api from 'services/api';
import { updateDetApi } from 'services/apiConfig';

export const setServerAddress = (address: string): void => {
  const serverAddress = address.replace(/\/\s*$/, '');
  globalStorage.serverAddress = serverAddress;
  updateDetApi({ basePath: serverAddress });
};

window.dev = {
  resetServerAddress: () => globalStorage.removeServerAddress(),
  setServerAddress,
};

if (process.env.IS_DEV) {
  window.dev.api = Api;
}
