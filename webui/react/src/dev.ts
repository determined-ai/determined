/* Tools and tweaks for dev environments */
import { globalStorage } from 'globalStorage';
import { devControls } from 'recordReplay';
import * as Api from 'services/api';
import { updateDetApi } from 'services/apiConfig';

const setServerAddress = (address: string) => {
  const serverAddress = address.replace(/\/\s*$/, '');
  globalStorage.serverAddress = serverAddress;
  updateDetApi({ basePath: serverAddress });
};

window.dev = window.dev || {};
window.dev = {
  ...window.dev,
  ...devControls,
  resetServerAddress: () => globalStorage.removeServerAddress(),
  setServerAddress,
};

if (process.env.IS_DEV) {
  window.dev.api = Api;
}
