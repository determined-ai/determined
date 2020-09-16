/* Tools and tweaks for dev environments */

import { globalStorage } from 'globalStorage';
import * as Api from 'services/api';

const setServerAddress = (address: string) => {
  const serverAddress = address.replace(/\/\s*$/, '');
  globalStorage.serverAddress = serverAddress;
  Api.updateDetApi({ basePath: serverAddress });
};

window.dev = {
  resetServerAddress: () => globalStorage.removeServerAddress(),
  setServerAddress,
};

if (process.env.IS_DEV) {
  window.dev.api = Api;
}
