/* Tools and tweaks for dev environments */

import * as Api from 'services/api';
import * as DetSwagger from 'services/api-ts-sdk';
import { consumeStream } from 'services/apiBuilder';

// TODO remove
consumeStream<DetSwagger.V1TrialLogsResponse>(
  DetSwagger.ExperimentsApiFetchParamCreator().determinedTrialLogs(1),
  console.log,
  () => console.log('finished'),
);

if (process.env.IS_DEV) {
  window.dev = window.dev || {};
  window.dev.api = Api;
  document.title = `[dev] ${document.title}`;
}
