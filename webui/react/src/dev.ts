/* Tools and tweaks for dev environments */
import { globalStorage } from 'globalStorage';
import { paths, routeToReactUrl } from 'routes/utils';
import * as Api from 'services/api';
import { updateDetApi } from 'services/apiConfig';

export const setServerAddress = (address: string): void => {
  const serverAddress = address.replace(/\/\s*$/, '');
  globalStorage.serverAddress = serverAddress;
  updateDetApi({ basePath: serverAddress });
  routeToReactUrl(paths.login());
};

window.dev = {
  resetServerAddress: () => globalStorage.removeServerAddress(),
  setServerAddress,
};

if (process.env.IS_DEV) {
  window.dev.api = Api;
}
