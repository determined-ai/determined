import { generateContext } from 'contexts';
import { DeterminedInfo } from 'types';

const contextProvider = generateContext<DeterminedInfo>({
  initialState: {
    clusterId: '',
    clusterName: '',
    masterId: '',
    ssoProviders: [],
    telemetry: { enabled: false },
    version: process.env.VERSION || '',
  },
  name: 'DeterminedInfo',
});

export default contextProvider;
