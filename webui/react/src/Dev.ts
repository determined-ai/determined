/* Tools and tweaks for dev environments */

import * as Api from 'services/api';

if (process.env.IS_DEV) {
  window.dev = window.dev || {};
  window.dev.api = Api;
}
