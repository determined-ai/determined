/* Tools and tweaks for dev environments */

import * as Api from 'services/api';
import { updateFavicon } from 'utils/browser';

if (process.env.IS_DEV) {
  window.dev = window.dev || {};
  window.dev.api = Api;
  document.title = `[dev] ${document.title}`;
  updateFavicon('/favicons/favicon-dev.png');
}
