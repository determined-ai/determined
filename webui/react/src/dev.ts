/* Tools and tweaks for dev environments */
import { globalStorage } from 'globalStorage';
import { paths, routeToReactUrl, serverAddress } from 'routes/utils';
import * as Api from 'services/api';
import { updateDetApi } from 'services/apiConfig';

const onServerAddressChange = () => {
  updateDetApi({ basePath: serverAddress() });
  routeToReactUrl(paths.logout());
};

export const setServerAddress = (address: string): void => {
  const serverAddress = address.replace(/\/\s*$/, '');
  globalStorage.serverAddress = serverAddress;
  onServerAddressChange();
};

export const resetServerAddress = (): void => {
  globalStorage.removeServerAddress();
  onServerAddressChange();
};

window.dev = {
  resetServerAddress,
  setServerAddress,
};

if (process.env.IS_DEV) {
  window.dev.api = Api;
}
