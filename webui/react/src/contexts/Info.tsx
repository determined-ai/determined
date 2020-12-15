import { generateContext } from 'contexts';
import { DeterminedInfo } from 'types';

const contextProvider = generateContext<DeterminedInfo>({
  initialState: {
    clusterId: '',
    clusterName: '',
    isTelemetryEnabled: false,
    masterId: '',
    version: process.env.VERSION || '',
  },
  name: 'DeterminedInfo',
});

export default contextProvider;
